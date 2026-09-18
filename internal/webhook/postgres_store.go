package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func webhookPGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

func (s *PostgresStore) Create(value Subscription) error {
	events, err := json.Marshal(value.EventTypes)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `
		INSERT INTO webhook_subscriptions (id,organization_id,url,secret,event_types,active,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, value.ID, value.OrganizationID, value.URL, value.Secret, events, value.Active, value.CreatedAt)
	return webhookPGError(err)
}

func (s *PostgresStore) List(org string) []Subscription {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id,organization_id,url,secret,event_types,active,created_at
		FROM webhook_subscriptions WHERE organization_id=$1 ORDER BY created_at`, org)
	if err != nil {
		return []Subscription{}
	}
	defer rows.Close()
	values := []Subscription{}
	for rows.Next() {
		var value Subscription
		var events []byte
		if err := rows.Scan(&value.ID, &value.OrganizationID, &value.URL, &value.Secret, &events, &value.Active, &value.CreatedAt); err != nil {
			continue
		}
		if err := json.Unmarshal(events, &value.EventTypes); err == nil {
			values = append(values, value)
		}
	}
	return values
}

func (s *PostgresStore) Delete(org, id string) error {
	result, err := s.db.ExecContext(context.Background(), `DELETE FROM webhook_subscriptions WHERE organization_id=$1 AND id=$2`, org, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
