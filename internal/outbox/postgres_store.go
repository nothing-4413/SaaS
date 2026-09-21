package outbox

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nothing-4413/saas/internal/platform/sqlctx"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func outboxPGError(err error) error {
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (s *PostgresStore) Enqueue(v Event) error {
	_, e := s.db.ExecContext(sqlctx.Context(), `INSERT INTO outbox_events (id,organization_id,aggregate_type,aggregate_id,event_type,dedup_key,payload,status,attempts,next_attempt_at,last_error,created_at,claimed_until,published_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULL,NULL)`, v.ID, v.OrganizationID, v.AggregateType, v.AggregateID, v.Type, v.DedupKey, v.Payload, v.Status, v.Attempts, v.NextAttemptAt, v.LastError, v.CreatedAt)
	return outboxPGError(e)
}
func scanEvent(scanner interface{ Scan(...interface{}) error }) (Event, error) {
	var v Event
	var claimed, published sql.NullTime
	e := scanner.Scan(&v.ID, &v.OrganizationID, &v.AggregateType, &v.AggregateID, &v.Type, &v.DedupKey, &v.Payload, &v.Status, &v.Attempts, &v.NextAttemptAt, &claimed, &v.LastError, &v.CreatedAt, &published)
	if claimed.Valid {
		v.ClaimedUntil = claimed.Time
	}
	if published.Valid {
		v.PublishedAt = &published.Time
	}
	return v, e
}

const eventColumns = `id,organization_id,aggregate_type,aggregate_id,event_type,dedup_key,payload,status,attempts,next_attempt_at,claimed_until,last_error,created_at,published_at`

func (s *PostgresStore) Claim(limit int, now time.Time) []Event {
	if limit <= 0 {
		return []Event{}
	}
	ctx := sqlctx.Context()
	_, _ = s.db.ExecContext(ctx, `UPDATE outbox_events SET status='failed',claimed_until=NULL,last_error='processing lease expired after maximum attempts' WHERE status='processing' AND claimed_until <= $1 AND attempts >= $2`, now, maxClaimAttempts)
	query := `WITH candidates AS (SELECT id FROM outbox_events WHERE ((status='pending' AND next_attempt_at <= $1) OR (status='processing' AND claimed_until <= $1)) AND attempts < $2 ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $3) UPDATE outbox_events e SET status='processing',attempts=e.attempts+1,claimed_until=$1 + interval '5 minutes' FROM candidates c WHERE e.id=c.id RETURNING e.` + eventColumns
	rows, e := s.db.QueryContext(ctx, query, now, maxClaimAttempts, limit)
	if e != nil {
		return []Event{}
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		v, e := scanEvent(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *PostgresStore) transitionError(id string) error {
	var status Status
	e := s.db.QueryRowContext(sqlctx.Context(), `SELECT status FROM outbox_events WHERE id=$1`, id).Scan(&status)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	return ErrInvalidState
}
func (s *PostgresStore) MarkPublished(id string, at time.Time) error {
	result, e := s.db.ExecContext(sqlctx.Context(), `UPDATE outbox_events SET status='published',claimed_until=NULL,published_at=$2,last_error='' WHERE id=$1 AND status='processing'`, id, at)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return s.transitionError(id)
	}
	return nil
}
func (s *PostgresStore) MarkFailed(id string, next time.Time, reason string, terminal bool) error {
	status := StatusPending
	if terminal {
		status = StatusFailed
	}
	result, e := s.db.ExecContext(sqlctx.Context(), `UPDATE outbox_events SET status=$2,claimed_until=NULL,next_attempt_at=$3,last_error=$4 WHERE id=$1 AND status='processing'`, id, status, next, reason)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return s.transitionError(id)
	}
	return nil
}
func (s *PostgresStore) Get(id string) (Event, error) {
	v, e := scanEvent(s.db.QueryRowContext(sqlctx.Context(), `SELECT `+eventColumns+` FROM outbox_events WHERE id=$1`, id))
	if errors.Is(e, sql.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	return v, e
}
func (s *PostgresStore) List(status Status) []Event {
	query := `SELECT ` + eventColumns + ` FROM outbox_events`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at`
	rows, e := s.db.QueryContext(sqlctx.Context(), query, args...)
	if e != nil {
		return []Event{}
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		v, e := scanEvent(rows)
		if e == nil {
			out = append(out, v)
		}
	}
	return out
}
