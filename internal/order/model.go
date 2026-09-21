package order

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusPaid      Status = "paid"
	StatusShipped   Status = "shipped"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
	StatusRefunded  Status = "refunded"
)

// CanTransition describes the explicit order state machine. Inventory is
// reserved while pending, consumed on confirmation, and restored on refund.
func CanTransition(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusConfirmed || to == StatusCancelled
	case StatusConfirmed:
		return to == StatusPaid
	case StatusPaid:
		return to == StatusShipped || to == StatusRefunded
	case StatusShipped:
		return to == StatusCompleted || to == StatusRefunded
	case StatusCompleted:
		return to == StatusRefunded
	default:
		return false
	}
}

type Line struct {
	SKUID          string `json:"sku_id"`
	WarehouseID    string `json:"warehouse_id"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}
type Order struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Status         Status    `json:"status"`
	Lines          []Line    `json:"lines"`
	TotalCents     int64     `json:"total_cents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type CreateInput struct {
	IdempotencyKey string `json:"idempotency_key"`
	Lines          []Line `json:"lines"`
}
