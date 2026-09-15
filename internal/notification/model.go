package notification

type OrderNotification struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}
