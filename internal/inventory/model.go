package inventory

import "time"

type Stock struct {
	OrganizationID string    `json:"organization_id"`
	WarehouseID    string    `json:"warehouse_id"`
	SKUID          string    `json:"sku_id"`
	OnHand         int64     `json:"on_hand"`
	Reserved       int64     `json:"reserved"`
	Available      int64     `json:"available"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type StockOperationInput struct {
	Quantity       int64  `json:"quantity"`
	IdempotencyKey string `json:"idempotency_key"`
}
