package product

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
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
	_, e := s.db.ExecContext(context.Background(), `INSERT INTO products (id,organization_id,name,description,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Name, v.Description, v.CreatedAt)
	return productPGError(e)
}
func (s *PostgresStore) ListProducts(org string) []Product {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,name,description,created_at FROM products WHERE organization_id=$1 ORDER BY created_at`, org)
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
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,name,description,created_at FROM products WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.Name, &v.Description, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) CreateSKU(v SKU) error {
	_, e := s.db.ExecContext(context.Background(), `INSERT INTO skus (id,organization_id,product_id,code,name,price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.OrganizationID, v.ProductID, v.Code, v.Name, v.PriceCents)
	return productPGError(e)
}
func (s *PostgresStore) ListSKUs(productID string) []SKU {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,product_id,code,name,price_cents FROM skus WHERE product_id=$1 ORDER BY code`, productID)
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
	_, e := s.db.ExecContext(context.Background(), `INSERT INTO warehouses (id,organization_id,name,address,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Name, v.Address, v.CreatedAt)
	return productPGError(e)
}
func (s *PostgresStore) ListWarehouses(org string) []Warehouse {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,name,address,created_at FROM warehouses WHERE organization_id=$1 ORDER BY created_at`, org)
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
