package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nothing-4413/saas/internal/platform/idgen"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func orderPGError(err error) error {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return ErrConflict
	}
	return err
}

func orderLineOrder(lines []Line) []int {
	indices := make([]int, len(lines))
	for i := range lines {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := lines[indices[i]], lines[indices[j]]
		if left.WarehouseID != right.WarehouseID {
			return left.WarehouseID < right.WarehouseID
		}
		return left.SKUID < right.SKUID
	})
	return indices
}

func (s *PostgresStore) CreateAtomic(v Order) (Order, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()
	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM orders WHERE organization_id=$1 AND idempotency_key=$2`, v.OrganizationID, v.IdempotencyKey).Scan(&existingID)
	if err == nil {
		_ = tx.Rollback()
		return s.Get(existingID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Order{}, err
	}
	for _, i := range orderLineOrder(v.Lines) {
		line := v.Lines[i]
		var onHand, reserved int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO inventory_stocks (organization_id,warehouse_id,sku_id,on_hand,reserved,updated_at)
			VALUES ($1,$2,$3,0,0,$4) ON CONFLICT DO NOTHING
			RETURNING on_hand,reserved`, v.OrganizationID, line.WarehouseID, line.SKUID, v.CreatedAt).Scan(&onHand, &reserved)
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.QueryRowContext(ctx, `SELECT on_hand,reserved FROM inventory_stocks WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3 FOR UPDATE`, v.OrganizationID, line.WarehouseID, line.SKUID).Scan(&onHand, &reserved)
		}
		if err != nil {
			return Order{}, err
		}
		if onHand-reserved < line.Quantity {
			return Order{}, errors.New("insufficient available stock")
		}
		reserved += line.Quantity
		if _, err = tx.ExecContext(ctx, `UPDATE inventory_stocks SET reserved=$4,updated_at=$5 WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3`, v.OrganizationID, line.WarehouseID, line.SKUID, reserved, v.CreatedAt); err != nil {
			return Order{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO orders (id,organization_id,idempotency_key,status,total_cents,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, v.ID, v.OrganizationID, v.IdempotencyKey, v.Status, v.TotalCents, v.CreatedAt, v.UpdatedAt); err != nil {
		if errors.Is(orderPGError(err), ErrConflict) {
			_ = tx.Rollback()
			return s.FindByKey(v.OrganizationID, v.IdempotencyKey)
		}
		return Order{}, err
	}
	for _, line := range v.Lines {
		if _, err = tx.ExecContext(ctx, `INSERT INTO order_lines (order_id,organization_id,warehouse_id,sku_id,quantity,unit_price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.OrganizationID, line.WarehouseID, line.SKUID, line.Quantity, line.UnitPriceCents); err != nil {
			return Order{}, err
		}
	}
	if err = insertOrderEvent(ctx, tx, v, "order.created"); err != nil {
		return Order{}, err
	}
	if err = tx.Commit(); err != nil {
		return Order{}, err
	}
	return v, nil
}

func insertOrderEvent(ctx context.Context, tx *sql.Tx, value Order, eventType string) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	now := value.UpdatedAt
	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox_events (id,organization_id,aggregate_type,aggregate_id,event_type,dedup_key,payload,status,attempts,next_attempt_at,last_error,created_at,claimed_until,published_at)
		VALUES ($1,$2,'order',$3,$4,$4,$5,'pending',0,$6,'',$6,NULL,NULL)
		ON CONFLICT (organization_id,aggregate_type,aggregate_id,event_type,dedup_key) DO NOTHING`, idgen.New(), value.OrganizationID, value.ID, eventType, payload, now)
	return err
}

func (s *PostgresStore) TransitionAtomic(org, id string, target Status, now time.Time) (Order, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()
	var value Order
	err = tx.QueryRowContext(ctx, `SELECT id,organization_id,idempotency_key,status,total_cents,created_at,updated_at FROM orders WHERE organization_id=$1 AND id=$2 FOR UPDATE`, org, id).
		Scan(&value.ID, &value.OrganizationID, &value.IdempotencyKey, &value.Status, &value.TotalCents, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	value.Lines = linesTx(ctx, tx, id)
	if value.Status == target {
		_ = tx.Rollback()
		return value, nil
	}
	if value.Status != StatusPending {
		return Order{}, ErrInvalidInput
	}
	for _, i := range orderLineOrder(value.Lines) {
		line := value.Lines[i]
		var onHand, reserved int64
		if err = tx.QueryRowContext(ctx, `SELECT on_hand,reserved FROM inventory_stocks WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3 FOR UPDATE`, org, line.WarehouseID, line.SKUID).Scan(&onHand, &reserved); err != nil {
			return Order{}, err
		}
		if target == StatusConfirmed {
			if reserved < line.Quantity {
				return Order{}, errors.New("insufficient reserved stock")
			}
			onHand -= line.Quantity
			reserved -= line.Quantity
		} else {
			if reserved < line.Quantity {
				return Order{}, errors.New("invalid reserved stock")
			}
			reserved -= line.Quantity
		}
		if _, err = tx.ExecContext(ctx, `UPDATE inventory_stocks SET on_hand=$4,reserved=$5,updated_at=$6 WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3`, org, line.WarehouseID, line.SKUID, onHand, reserved, now); err != nil {
			return Order{}, err
		}
		action := "release"
		if target == StatusConfirmed {
			action = "consume_reserved"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_operations (organization_id,warehouse_id,sku_id,action,quantity,idempotency_key,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, org, line.WarehouseID, line.SKUID, action, line.Quantity, fmt.Sprintf("order:%s:%s:%d", target, id, i), now); err != nil {
			return Order{}, err
		}
	}
	value.Status = target
	value.UpdatedAt = now
	if _, err = tx.ExecContext(ctx, `UPDATE orders SET status=$2,updated_at=$3 WHERE organization_id=$1 AND id=$4`, org, target, now, id); err != nil {
		return Order{}, err
	}
	eventType := "order.cancelled"
	if target == StatusConfirmed {
		eventType = "order.confirmed"
	}
	if err = insertOrderEvent(ctx, tx, value, eventType); err != nil {
		return Order{}, err
	}
	if err = tx.Commit(); err != nil {
		return Order{}, err
	}
	return value, nil
}

func linesTx(ctx context.Context, tx *sql.Tx, id string) []Line {
	rows, err := tx.QueryContext(ctx, `SELECT sku_id,warehouse_id,quantity,unit_price_cents FROM order_lines WHERE order_id=$1 ORDER BY id`, id)
	if err != nil {
		return []Line{}
	}
	defer rows.Close()
	lines := []Line{}
	for rows.Next() {
		var line Line
		if rows.Scan(&line.SKUID, &line.WarehouseID, &line.Quantity, &line.UnitPriceCents) == nil {
			lines = append(lines, line)
		}
	}
	return lines
}
func (s *PostgresStore) Create(v Order) error {
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `INSERT INTO orders (id,organization_id,idempotency_key,status,total_cents,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, v.ID, v.OrganizationID, v.IdempotencyKey, v.Status, v.TotalCents, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return orderPGError(e)
	}
	for _, l := range v.Lines {
		if _, e = tx.ExecContext(ctx, `INSERT INTO order_lines (order_id,organization_id,warehouse_id,sku_id,quantity,unit_price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.OrganizationID, l.WarehouseID, l.SKUID, l.Quantity, l.UnitPriceCents); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) Delete(id string) error {
	result, e := s.db.ExecContext(context.Background(), `DELETE FROM orders WHERE id=$1`, id)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) lines(id string) []Line {
	rows, e := s.db.QueryContext(context.Background(), `SELECT sku_id,warehouse_id,quantity,unit_price_cents FROM order_lines WHERE order_id=$1 ORDER BY id`, id)
	if e != nil {
		return []Line{}
	}
	defer rows.Close()
	out := []Line{}
	for rows.Next() {
		var v Line
		if rows.Scan(&v.SKUID, &v.WarehouseID, &v.Quantity, &v.UnitPriceCents) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) Get(id string) (Order, error) {
	var v Order
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,idempotency_key,status,total_cents,created_at,updated_at FROM orders WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.IdempotencyKey, &v.Status, &v.TotalCents, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if e != nil {
		return Order{}, e
	}
	v.Lines = s.lines(id)
	return v, nil
}
func (s *PostgresStore) Put(v Order) error {
	result, e := s.db.ExecContext(context.Background(), `UPDATE orders SET status=$2,total_cents=$3,updated_at=$4 WHERE id=$1`, v.ID, v.Status, v.TotalCents, v.UpdatedAt)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) List(org string) []Order {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,idempotency_key,status,total_cents,created_at,updated_at FROM orders WHERE organization_id=$1 ORDER BY created_at`, org)
	if e != nil {
		return []Order{}
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var v Order
		if rows.Scan(&v.ID, &v.OrganizationID, &v.IdempotencyKey, &v.Status, &v.TotalCents, &v.CreatedAt, &v.UpdatedAt) == nil {
			v.Lines = s.lines(v.ID)
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) FindByKey(org, key string) (Order, error) {
	var id string
	e := s.db.QueryRowContext(context.Background(), `SELECT id FROM orders WHERE organization_id=$1 AND idempotency_key=$2`, org, key).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if e != nil {
		return Order{}, e
	}
	return s.Get(id)
}
