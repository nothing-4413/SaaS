package report

type Summary struct {
	OrganizationID string       `json:"organization_id"`
	Orders         OrderSummary `json:"orders"`
	Inventory      StockSummary `json:"inventory"`
}

type OrderSummary struct {
	Total          int   `json:"total"`
	Pending        int   `json:"pending"`
	Confirmed      int   `json:"confirmed"`
	Paid           int   `json:"paid"`
	Shipped        int   `json:"shipped"`
	Completed      int   `json:"completed"`
	Cancelled      int   `json:"cancelled"`
	Refunded       int   `json:"refunded"`
	ConfirmedCents int64 `json:"confirmed_cents"`
}

type StockSummary struct {
	SKUs      int   `json:"skus"`
	OnHand    int64 `json:"on_hand"`
	Reserved  int64 `json:"reserved"`
	Available int64 `json:"available"`
	LowStock  int   `json:"low_stock"`
}
