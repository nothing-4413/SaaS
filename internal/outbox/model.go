package outbox

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
)

type Event struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	AggregateType  string     `json:"aggregate_type"`
	AggregateID    string     `json:"aggregate_id"`
	Type           string     `json:"type"`
	DedupKey       string     `json:"dedup_key"`
	Payload        []byte     `json:"payload"`
	Status         Status     `json:"status"`
	Attempts       int        `json:"attempts"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
}
