package alert

import (
	"database/sql"
	"errors"

	"github.com/nothing-4413/saas/internal/platform/sqlctx"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) Put(value Rule) error {
	_, err := s.db.ExecContext(sqlctx.Context(), `
		INSERT INTO stock_alert_rules (organization_id,threshold,enabled,updated_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (organization_id) DO UPDATE
		SET threshold=EXCLUDED.threshold,enabled=EXCLUDED.enabled,updated_at=EXCLUDED.updated_at`,
		value.OrganizationID, value.Threshold, value.Enabled, value.UpdatedAt)
	return err
}

func (s *PostgresStore) Get(org string) (Rule, error) {
	var value Rule
	err := s.db.QueryRowContext(sqlctx.Context(), `SELECT organization_id,threshold,enabled,updated_at FROM stock_alert_rules WHERE organization_id=$1`, org).
		Scan(&value.OrganizationID, &value.Threshold, &value.Enabled, &value.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Rule{}, ErrNotFound
	}
	return value, err
}

func (s *PostgresStore) ListEnabled() []Rule {
	rows, err := s.db.QueryContext(sqlctx.Context(), `SELECT organization_id,threshold,enabled,updated_at FROM stock_alert_rules WHERE enabled=true ORDER BY organization_id`)
	if err != nil {
		return []Rule{}
	}
	defer rows.Close()
	values := []Rule{}
	for rows.Next() {
		var value Rule
		if err := rows.Scan(&value.OrganizationID, &value.Threshold, &value.Enabled, &value.UpdatedAt); err == nil {
			values = append(values, value)
		}
	}
	return values
}
