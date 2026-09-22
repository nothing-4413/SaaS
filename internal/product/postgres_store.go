package product

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nothing-4413/saas/internal/platform/sqlctx"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func productPGError(err error) error {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (s *PostgresStore) CreateProduct(v Product) error {
	_, e := s.db.ExecContext(sqlctx.Context(), `INSERT INTO products (id,organization_id,name,description,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Name, v.Description, v.CreatedAt)
	return productPGError(e)
}
func (s *PostgresStore) UpdateProduct(v Product) error {
	r, e := s.db.ExecContext(sqlctx.Context(), `UPDATE products SET name=$1,description=$2 WHERE id=$3 AND organization_id=$4`, v.Name, v.Description, v.ID, v.OrganizationID)
	if e != nil {
		return productPGError(e)
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) ListProducts(org string) []Product {
	rows, e := s.db.QueryContext(sqlctx.Context(), `SELECT id,organization_id,name,description,created_at FROM products WHERE organization_id=$1 ORDER BY created_at`, org)
	if e != nil {
		return []Product{}
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		var v Product
		if rows.Scan(&v.ID, &v.OrganizationID, &v.Name, &v.Description, &v.CreatedAt) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) GetProduct(id string) (Product, error) {
	var v Product
	e := s.db.QueryRowContext(sqlctx.Context(), `SELECT id,organization_id,name,description,created_at FROM products WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.Name, &v.Description, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) CreateSKU(v SKU) error {
	_, e := s.db.ExecContext(sqlctx.Context(), `INSERT INTO skus (id,organization_id,product_id,code,name,price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.OrganizationID, v.ProductID, v.Code, v.Name, v.PriceCents)
	return productPGError(e)
}
func (s *PostgresStore) UpdateSKU(v SKU) error {
	r, e := s.db.ExecContext(sqlctx.Context(), `UPDATE skus SET code=$1,name=$2,price_cents=$3 WHERE id=$4 AND organization_id=$5`, v.Code, v.Name, v.PriceCents, v.ID, v.OrganizationID)
	if e != nil {
		return productPGError(e)
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) GetSKU(id string) (SKU, error) {
	var v SKU
	e := s.db.QueryRowContext(sqlctx.Context(), `SELECT id,organization_id,product_id,code,name,price_cents FROM skus WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.ProductID, &v.Code, &v.Name, &v.PriceCents)
	if errors.Is(e, sql.ErrNoRows) {
		return SKU{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) ListSKUs(productID string) []SKU {
	rows, e := s.db.QueryContext(sqlctx.Context(), `SELECT id,organization_id,product_id,code,name,price_cents FROM skus WHERE product_id=$1 ORDER BY code`, productID)
	if e != nil {
		return []SKU{}
	}
	defer rows.Close()
	out := []SKU{}
	for rows.Next() {
		var v SKU
		if rows.Scan(&v.ID, &v.OrganizationID, &v.ProductID, &v.Code, &v.Name, &v.PriceCents) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) CreateWarehouse(v Warehouse) error {
	_, e := s.db.ExecContext(sqlctx.Context(), `INSERT INTO warehouses (id,organization_id,name,address,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Name, v.Address, v.CreatedAt)
	return productPGError(e)
}
func (s *PostgresStore) UpdateWarehouse(v Warehouse) error {
	r, e := s.db.ExecContext(sqlctx.Context(), `UPDATE warehouses SET name=$1,address=$2 WHERE id=$3 AND organization_id=$4`, v.Name, v.Address, v.ID, v.OrganizationID)
	if e != nil {
		return productPGError(e)
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) ListWarehouses(org string) []Warehouse {
	rows, e := s.db.QueryContext(sqlctx.Context(), `SELECT id,organization_id,name,address,created_at FROM warehouses WHERE organization_id=$1 ORDER BY created_at`, org)
	if e != nil {
		return []Warehouse{}
	}
	defer rows.Close()
	out := []Warehouse{}
	for rows.Next() {
		var v Warehouse
		if rows.Scan(&v.ID, &v.OrganizationID, &v.Name, &v.Address, &v.CreatedAt) == nil {
			out = append(out, v)
		}
	}
	return out
}
