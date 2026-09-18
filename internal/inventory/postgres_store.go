package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func documentLineOrder(lines []DocumentLine) []int {
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

func (s *PostgresStore) CreateDocumentAtomic(v Document) (Document, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback()
	var oldID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM inventory_documents WHERE organization_id=$1 AND document_type=$2 AND idempotency_key=$3`, v.OrganizationID, v.Type, v.IdempotencyKey).Scan(&oldID)
	if err == nil {
		_ = tx.Rollback()
		return s.GetDocument(oldID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Document{}, err
	}
	for _, i := range documentLineOrder(v.Lines) {
		line := v.Lines[i]
		var onHand, reserved int64
		if v.Type == DocumentReceipt {
			if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_stocks (organization_id,warehouse_id,sku_id,on_hand,reserved,updated_at) VALUES ($1,$2,$3,0,0,$4) ON CONFLICT DO NOTHING`, v.OrganizationID, line.WarehouseID, line.SKUID, v.CreatedAt); err != nil {
				return Document{}, err
			}
		}
		if err = tx.QueryRowContext(ctx, `SELECT on_hand,reserved FROM inventory_stocks WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3 FOR UPDATE`, v.OrganizationID, line.WarehouseID, line.SKUID).Scan(&onHand, &reserved); err != nil {
			return Document{}, err
		}
		if v.Type == DocumentReceipt {
			onHand += line.Quantity
		} else {
			if onHand-reserved < line.Quantity {
				return Document{}, ErrInsufficient
			}
			onHand -= line.Quantity
		}
		if _, err = tx.ExecContext(ctx, `UPDATE inventory_stocks SET on_hand=$4,updated_at=$5 WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3`, v.OrganizationID, line.WarehouseID, line.SKUID, onHand, v.CreatedAt); err != nil {
			return Document{}, err
		}
		action := "receive"
		if v.Type == DocumentIssue {
			action = "deduct"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_operations (organization_id,warehouse_id,sku_id,action,quantity,idempotency_key,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, v.OrganizationID, line.WarehouseID, line.SKUID, action, line.Quantity, fmt.Sprintf("document:%s:%s:%d", v.Type, v.IdempotencyKey, i), v.CreatedAt); err != nil {
			return Document{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_documents (id,organization_id,document_type,idempotency_key,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Type, v.IdempotencyKey, v.CreatedAt); err != nil {
		if errors.Is(inventoryPGError(err), ErrConflict) {
			_ = tx.Rollback()
			return s.FindDocumentByKey(v.OrganizationID, v.Type, v.IdempotencyKey)
		}
		return Document{}, err
	}
	for _, line := range v.Lines {
		if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_document_lines (organization_id,document_id,warehouse_id,sku_id,quantity) VALUES ($1,$2,$3,$4,$5)`, v.OrganizationID, v.ID, line.WarehouseID, line.SKUID, line.Quantity); err != nil {
			return Document{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Document{}, err
	}
	return v, nil
}

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func inventoryPGError(err error) error {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (s *PostgresStore) existingOperation(org, key, warehouse, sku, action string, quantity int64) (Stock, error) {
	var oldWarehouse, oldSKU, oldAction string
	var oldQty int64
	err := s.db.QueryRowContext(context.Background(), `SELECT warehouse_id,sku_id,action,quantity FROM inventory_operations WHERE organization_id=$1 AND idempotency_key=$2`, org, key).Scan(&oldWarehouse, &oldSKU, &oldAction, &oldQty)
	if errors.Is(err, sql.ErrNoRows) {
		return Stock{}, ErrNotFound
	}
	if err != nil {
		return Stock{}, err
	}
	if oldWarehouse != warehouse || oldSKU != sku || oldAction != action || oldQty != quantity {
		return Stock{}, ErrConflict
	}
	return s.Get(org, warehouse, sku)
}
func (s *PostgresStore) Get(org, warehouse, sku string) (Stock, error) {
	var v Stock
	e := s.db.QueryRowContext(context.Background(), `SELECT organization_id,warehouse_id,sku_id,on_hand,reserved,on_hand-reserved,updated_at FROM inventory_stocks WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3`, org, warehouse, sku).Scan(&v.OrganizationID, &v.WarehouseID, &v.SKUID, &v.OnHand, &v.Reserved, &v.Available, &v.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Stock{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) Put(v Stock) error {
	_, e := s.db.ExecContext(context.Background(), `INSERT INTO inventory_stocks (organization_id,warehouse_id,sku_id,on_hand,reserved,updated_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (organization_id,warehouse_id,sku_id) DO UPDATE SET on_hand=EXCLUDED.on_hand,reserved=EXCLUDED.reserved,updated_at=EXCLUDED.updated_at`, v.OrganizationID, v.WarehouseID, v.SKUID, v.OnHand, v.Reserved, v.UpdatedAt)
	return e
}
func (s *PostgresStore) ImportStocks(values []Stock, at time.Time) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, value := range values {
		_, err = tx.ExecContext(context.Background(), `
			INSERT INTO inventory_stocks (organization_id,warehouse_id,sku_id,on_hand,reserved,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (organization_id,warehouse_id,sku_id) DO UPDATE
			SET on_hand=EXCLUDED.on_hand,reserved=EXCLUDED.reserved,updated_at=EXCLUDED.updated_at`,
			value.OrganizationID, value.WarehouseID, value.SKUID, value.OnHand, value.Reserved, at)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) List(org string) []Stock {
	rows, e := s.db.QueryContext(context.Background(), `SELECT organization_id,warehouse_id,sku_id,on_hand,reserved,on_hand-reserved,updated_at FROM inventory_stocks WHERE organization_id=$1 ORDER BY warehouse_id,sku_id`, org)
	if e != nil {
		return []Stock{}
	}
	defer rows.Close()
	out := []Stock{}
	for rows.Next() {
		var v Stock
		if rows.Scan(&v.OrganizationID, &v.WarehouseID, &v.SKUID, &v.OnHand, &v.Reserved, &v.Available, &v.UpdatedAt) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) Apply(org, warehouse, sku, action string, quantity int64, idempotencyKey string, at time.Time) (Stock, error) {
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return Stock{}, e
	}
	defer tx.Rollback()
	var oldWarehouse, oldSKU, oldAction string
	var oldQty int64
	e = tx.QueryRowContext(ctx, `SELECT warehouse_id,sku_id,action,quantity FROM inventory_operations WHERE organization_id=$1 AND idempotency_key=$2`, org, idempotencyKey).Scan(&oldWarehouse, &oldSKU, &oldAction, &oldQty)
	if e == nil {
		if oldWarehouse != warehouse || oldSKU != sku || oldAction != action || oldQty != quantity {
			return Stock{}, ErrConflict
		}
		_ = tx.Rollback()
		return s.Get(org, warehouse, sku)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return Stock{}, e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO inventory_stocks (organization_id,warehouse_id,sku_id,on_hand,reserved,updated_at) VALUES ($1,$2,$3,0,0,$4) ON CONFLICT DO NOTHING`, org, warehouse, sku, at); e != nil {
		return Stock{}, e
	}
	var onHand, reserved int64
	e = tx.QueryRowContext(ctx, `SELECT on_hand,reserved FROM inventory_stocks WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3 FOR UPDATE`, org, warehouse, sku).Scan(&onHand, &reserved)
	if e != nil {
		return Stock{}, e
	}
	switch action {
	case "receive":
		onHand += quantity
	case "reserve":
		if onHand-reserved < quantity {
			return Stock{}, ErrInsufficient
		}
		reserved += quantity
	case "release":
		if reserved < quantity {
			return Stock{}, ErrInvalidInput
		}
		reserved -= quantity
	case "deduct":
		if onHand-reserved < quantity {
			return Stock{}, ErrInsufficient
		}
		onHand -= quantity
	case "consume_reserved":
		if reserved < quantity {
			return Stock{}, ErrInsufficient
		}
		onHand -= quantity
		reserved -= quantity
	default:
		return Stock{}, ErrInvalidInput
	}
	if _, e = tx.ExecContext(ctx, `UPDATE inventory_stocks SET on_hand=$4,reserved=$5,updated_at=$6 WHERE organization_id=$1 AND warehouse_id=$2 AND sku_id=$3`, org, warehouse, sku, onHand, reserved, at); e != nil {
		return Stock{}, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO inventory_operations (organization_id,warehouse_id,sku_id,action,quantity,idempotency_key,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, org, warehouse, sku, action, quantity, idempotencyKey, at)
	if e != nil {
		if errors.Is(inventoryPGError(e), ErrConflict) {
			_ = tx.Rollback()
			return s.existingOperation(org, idempotencyKey, warehouse, sku, action, quantity)
		}
		return Stock{}, e
	}
	if e = tx.Commit(); e != nil {
		return Stock{}, e
	}
	return Stock{OrganizationID: org, WarehouseID: warehouse, SKUID: sku, OnHand: onHand, Reserved: reserved, Available: onHand - reserved, UpdatedAt: at}, nil
}
func (s *PostgresStore) CreateDocument(v Document) error {
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `INSERT INTO inventory_documents (id,organization_id,document_type,idempotency_key,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Type, v.IdempotencyKey, v.CreatedAt)
	if e != nil {
		return inventoryPGError(e)
	}
	for _, l := range v.Lines {
		if _, e = tx.ExecContext(ctx, `INSERT INTO inventory_document_lines (organization_id,document_id,warehouse_id,sku_id,quantity) VALUES ($1,$2,$3,$4,$5)`, v.OrganizationID, v.ID, l.WarehouseID, l.SKUID, l.Quantity); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) lines(id string) []DocumentLine {
	rows, e := s.db.QueryContext(context.Background(), `SELECT warehouse_id,sku_id,quantity FROM inventory_document_lines WHERE document_id=$1 ORDER BY id`, id)
	if e != nil {
		return []DocumentLine{}
	}
	defer rows.Close()
	out := []DocumentLine{}
	for rows.Next() {
		var v DocumentLine
		if rows.Scan(&v.WarehouseID, &v.SKUID, &v.Quantity) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) GetDocument(id string) (Document, error) {
	var v Document
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,document_type,idempotency_key,created_at FROM inventory_documents WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.Type, &v.IdempotencyKey, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if e != nil {
		return Document{}, e
	}
	v.Lines = s.lines(id)
	return v, nil
}
func (s *PostgresStore) ListDocuments(org string, typ DocumentType) []Document {
	query := `SELECT id,organization_id,document_type,idempotency_key,created_at FROM inventory_documents WHERE organization_id=$1`
	args := []interface{}{org}
	if typ != "" {
		query += ` AND document_type=$2`
		args = append(args, typ)
	}
	query += ` ORDER BY created_at`
	rows, e := s.db.QueryContext(context.Background(), query, args...)
	if e != nil {
		return []Document{}
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		var v Document
		if rows.Scan(&v.ID, &v.OrganizationID, &v.Type, &v.IdempotencyKey, &v.CreatedAt) == nil {
			v.Lines = s.lines(v.ID)
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) FindDocumentByKey(org string, typ DocumentType, key string) (Document, error) {
	var id string
	e := s.db.QueryRowContext(context.Background(), `SELECT id FROM inventory_documents WHERE organization_id=$1 AND document_type=$2 AND idempotency_key=$3`, org, typ, key).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if e != nil {
		return Document{}, e
	}
	return s.GetDocument(id)
}
