package order

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
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
