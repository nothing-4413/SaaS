package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func pgError(err error) error {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (s *PostgresStore) CreateOrganization(v Organization) error {
	_, e := s.db.ExecContext(context.Background(), `INSERT INTO organizations (id,name,created_at) VALUES ($1,$2,$3)`, v.ID, v.Name, v.CreatedAt)
	return pgError(e)
}
func (s *PostgresStore) GetOrganization(id string) (Organization, error) {
	var v Organization
	e := s.db.QueryRowContext(context.Background(), `SELECT id,name,created_at FROM organizations WHERE id=$1`, id).Scan(&v.ID, &v.Name, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) CreateRole(v Role) error {
	b, e := json.Marshal(v.Permissions)
	if e != nil {
		return e
	}
	_, e = s.db.ExecContext(context.Background(), `INSERT INTO roles (id,organization_id,name,permissions) VALUES ($1,$2,$3,$4)`, v.ID, v.OrganizationID, v.Name, b)
	return pgError(e)
}
func scanRole(scanner interface{ Scan(...interface{}) error }) (Role, error) {
	var v Role
	var b []byte
	e := scanner.Scan(&v.ID, &v.OrganizationID, &v.Name, &b)
	if e != nil {
		return Role{}, e
	}
	e = json.Unmarshal(b, &v.Permissions)
	return v, e
}
func (s *PostgresStore) ListRoles(org string) []Role {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,name,permissions FROM roles WHERE organization_id=$1 ORDER BY name`, org)
	if e != nil {
		return []Role{}
	}
	defer rows.Close()
	out := []Role{}
	for rows.Next() {
		v, e := scanRole(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) GetRole(id string) (Role, error) {
	v, e := scanRole(s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,name,permissions FROM roles WHERE id=$1`, id))
	if errors.Is(e, sql.ErrNoRows) {
		return Role{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) CreateUser(v User) error {
	tx, e := s.db.BeginTx(context.Background(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`INSERT INTO users (id,organization_id,email,name,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.Email, v.Name, v.CreatedAt)
	if e != nil {
		return pgError(e)
	}
	for _, roleID := range v.RoleIDs {
		if _, e = tx.Exec(`INSERT INTO user_roles (organization_id,user_id,role_id) VALUES ($1,$2,$3)`, v.OrganizationID, v.ID, roleID); e != nil {
			return pgError(e)
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) rolesForUser(id string) []string {
	rows, e := s.db.QueryContext(context.Background(), `SELECT role_id FROM user_roles WHERE user_id=$1 ORDER BY role_id`, id)
	if e != nil {
		return []string{}
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var role string
		if rows.Scan(&role) == nil {
			out = append(out, role)
		}
	}
	return out
}
func (s *PostgresStore) ListUsers(org string) []User {
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,email,name,created_at FROM users WHERE organization_id=$1 ORDER BY created_at`, org)
	if e != nil {
		return []User{}
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var v User
		if rows.Scan(&v.ID, &v.OrganizationID, &v.Email, &v.Name, &v.CreatedAt) == nil {
			v.RoleIDs = s.rolesForUser(v.ID)
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) GetUser(id string) (User, error) {
	var v User
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,email,name,created_at FROM users WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.Email, &v.Name, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if e != nil {
		return User{}, e
	}
	v.RoleIDs = s.rolesForUser(v.ID)
	return v, nil
}
