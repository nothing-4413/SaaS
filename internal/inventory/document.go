package inventory

import "time"

type DocumentType string

const (
	DocumentReceipt DocumentType = "receipt"
	DocumentIssue   DocumentType = "issue"
)

type Document struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Type           DocumentType   `json:"type"`
	IdempotencyKey string         `json:"idempotency_key"`
	Lines          []DocumentLine `json:"lines"`
	CreatedAt      time.Time      `json:"created_at"`
}
type DocumentLine struct {
	WarehouseID string `json:"warehouse_id"`
	SKUID       string `json:"sku_id"`
	Quantity    int64  `json:"quantity"`
}
type DocumentInput struct {
	IdempotencyKey string         `json:"idempotency_key"`
	Lines          []DocumentLine `json:"lines"`
}
