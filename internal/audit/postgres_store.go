package audit

import (
	"database/sql"
	"encoding/json"

	"github.com/nothing-4413/saas/internal/platform/sqlctx"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) Append(v Entry) error {
	metadata, err := json.Marshal(v.Metadata)
	if err != nil {
		return err
	}
	var actor interface{}
	if v.ActorUserID != "" {
		actor = v.ActorUserID
	}
	_, err = s.db.ExecContext(sqlctx.Context(), `
		INSERT INTO audit_logs (id,organization_id,actor_user_id,action,resource_type,resource_id,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		v.ID, v.OrganizationID, actor, v.Action, v.ResourceType, v.ResourceID, metadata, v.CreatedAt)
	return err
}

func (s *PostgresStore) List(org string) []Entry {
	rows, err := s.db.QueryContext(sqlctx.Context(), `
		SELECT id,organization_id,actor_user_id,action,resource_type,resource_id,metadata,created_at
		FROM audit_logs WHERE organization_id=$1 ORDER BY created_at DESC LIMIT 500`, org)
	if err != nil {
		return []Entry{}
	}
	defer rows.Close()
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		var actor sql.NullString
		var metadata []byte
		if err := rows.Scan(&entry.ID, &entry.OrganizationID, &actor, &entry.Action, &entry.ResourceType, &entry.ResourceID, &metadata, &entry.CreatedAt); err != nil {
			continue
		}
		entry.ActorUserID = actor.String
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			entry.Metadata = map[string]interface{}{}
		}
		entries = append(entries, entry)
	}
	return entries
}
