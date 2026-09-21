package report

import (
	"errors"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
)

var ErrInvalidInput = errors.New("invalid report input")

type OrderProvider interface{ List(string) []order.Order }
type StockProvider interface {
	List(string) []inventory.Stock
}

type Service struct {
	orders OrderProvider
	stocks StockProvider
}

func NewService(orders OrderProvider, stocks StockProvider) *Service {
	return &Service{orders: orders, stocks: stocks}
}

func (s *Service) Summary(org string, lowStockThreshold int64) (Summary, error) {
	if strings.TrimSpace(org) == "" || lowStockThreshold < 0 {
		return Summary{}, ErrInvalidInput
	}
	result := Summary{OrganizationID: org}
	for _, o := range s.orders.List(org) {
		result.Orders.Total++
		switch o.Status {
		case order.StatusPending:
			result.Orders.Pending++
		case order.StatusConfirmed:
			result.Orders.Confirmed++
			result.Orders.ConfirmedCents += o.TotalCents
		case order.StatusPaid:
			result.Orders.Paid++
			result.Orders.ConfirmedCents += o.TotalCents
		case order.StatusShipped:
			result.Orders.Shipped++
			result.Orders.ConfirmedCents += o.TotalCents
		case order.StatusCompleted:
			result.Orders.Completed++
			result.Orders.ConfirmedCents += o.TotalCents
		case order.StatusCancelled:
			result.Orders.Cancelled++
		case order.StatusRefunded:
			result.Orders.Refunded++
		}
	}
	for _, stock := range s.stocks.List(org) {
		result.Inventory.SKUs++
		result.Inventory.OnHand += stock.OnHand
		result.Inventory.Reserved += stock.Reserved
		result.Inventory.Available += stock.Available
		if stock.Available <= lowStockThreshold {
			result.Inventory.LowStock++
		}
	}
	return result, nil
}
