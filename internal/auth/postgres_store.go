package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

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
func (s *PostgresStore) CreateOrganizationOwner(org Organization, role Role, user User) error {
	permissions, err := json.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO organizations (id,name,created_at) VALUES ($1,$2,$3)`, org.ID, org.Name, org.CreatedAt); err != nil {
		return pgError(err)
	}
	if _, err = tx.Exec(`INSERT INTO roles (id,organization_id,name,permissions) VALUES ($1,$2,$3,$4)`, role.ID, role.OrganizationID, role.Name, permissions); err != nil {
		return pgError(err)
	}
	if _, err = tx.Exec(`INSERT INTO users (id,organization_id,email,name,password_hash,active,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, user.ID, user.OrganizationID, user.Email, user.Name, user.PasswordHash, user.Active, user.CreatedAt); err != nil {
		return pgError(err)
	}
	if _, err = tx.Exec(`INSERT INTO user_roles (organization_id,user_id,role_id) VALUES ($1,$2,$3)`, org.ID, user.ID, role.ID); err != nil {
		return pgError(err)
	}
	return tx.Commit()
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
func (s *PostgresStore) UpdateRole(v Role) error {
	b, err := json.Marshal(v.Permissions)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `UPDATE roles SET name=$3,permissions=$4 WHERE organization_id=$1 AND id=$2`, v.OrganizationID, v.ID, v.Name, b)
	if err != nil {
		return pgError(err)
	}
	return nil
}
func (s *PostgresStore) CreateUser(v User) error {
	tx, e := s.db.BeginTx(context.Background(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`INSERT INTO users (id,organization_id,email,name,password_hash,active,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, v.ID, v.OrganizationID, v.Email, v.Name, v.PasswordHash, v.Active, v.CreatedAt)
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
func (s *PostgresStore) UpdateUser(v User) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE users SET name=$3,password_hash=$4,active=$5 WHERE organization_id=$1 AND id=$2`, v.OrganizationID, v.ID, v.Name, v.PasswordHash, v.Active)
	if err != nil {
		return pgError(err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	if _, err = tx.Exec(`DELETE FROM user_roles WHERE organization_id=$1 AND user_id=$2`, v.OrganizationID, v.ID); err != nil {
		return err
	}
	for _, roleID := range v.RoleIDs {
		if _, err = tx.Exec(`INSERT INTO user_roles (organization_id,user_id,role_id) VALUES ($1,$2,$3)`, v.OrganizationID, v.ID, roleID); err != nil {
			return pgError(err)
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
	rows, e := s.db.QueryContext(context.Background(), `SELECT id,organization_id,email,name,password_hash,active,created_at FROM users WHERE organization_id=$1 ORDER BY created_at`, org)
	if e != nil {
		return []User{}
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var v User
		if rows.Scan(&v.ID, &v.OrganizationID, &v.Email, &v.Name, &v.PasswordHash, &v.Active, &v.CreatedAt) == nil {
			v.RoleIDs = s.rolesForUser(v.ID)
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) GetUser(id string) (User, error) {
	var v User
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,email,name,password_hash,active,created_at FROM users WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.Email, &v.Name, &v.PasswordHash, &v.Active, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if e != nil {
		return User{}, e
	}
	v.RoleIDs = s.rolesForUser(v.ID)
	return v, nil
}
func (s *PostgresStore) FindUserByEmail(org, email string) (User, error) {
	var v User
	e := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,email,name,password_hash,active,created_at FROM users WHERE organization_id=$1 AND email=$2`, org, email).Scan(&v.ID, &v.OrganizationID, &v.Email, &v.Name, &v.PasswordHash, &v.Active, &v.CreatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if e != nil {
		return User{}, e
	}
	v.RoleIDs = s.rolesForUser(v.ID)
	return v, nil
}

func (s *PostgresStore) CreateSession(v Session) error {
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO auth_sessions (id,organization_id,user_id,expires_at,created_at) VALUES ($1,$2,$3,$4,$5)`, v.ID, v.OrganizationID, v.UserID, v.ExpiresAt, v.CreatedAt)
	return pgError(err)
}
func (s *PostgresStore) GetSession(id string) (Session, error) {
	var v Session
	var revoked sql.NullTime
	err := s.db.QueryRowContext(context.Background(), `SELECT id,organization_id,user_id,expires_at,created_at,revoked_at FROM auth_sessions WHERE id=$1`, id).Scan(&v.ID, &v.OrganizationID, &v.UserID, &v.ExpiresAt, &v.CreatedAt, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if revoked.Valid {
		v.RevokedAt = &revoked.Time
	}
	return v, err
}
func (s *PostgresStore) RevokeSession(id string, at time.Time) error {
	result, err := s.db.ExecContext(context.Background(), `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,$2) WHERE id=$1`, id, at)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}
