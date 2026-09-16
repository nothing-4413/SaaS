package alert

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/outbox"
)

var ErrInvalidInput = errors.New("invalid alert input")

type Service struct {
	stocks interface {
		List(string) []inventory.Stock
	}
	events *outbox.Service
}

func NewService(stocks interface {
	List(string) []inventory.Stock
}, events *outbox.Service) *Service { return &Service{stocks: stocks, events: events} }
func (s *Service) Scan(org string, threshold int64) (int, error) {
	if strings.TrimSpace(org) == "" || threshold < 0 || s.stocks == nil || s.events == nil {
		return 0, ErrInvalidInput
	}
	count := 0
	for _, v := range s.stocks.List(org) {
		if v.Available <= threshold {
			key := fmt.Sprintf("%s:%d", v.UpdatedAt.UTC().Format("20060102150405.000000000"), v.Available)
			_, e := s.events.Enqueue(org, "stock", v.WarehouseID+":"+v.SKUID, "stock.low", key, v)
			if e == nil {
				count++
			}
		}
	}
	return count, nil
}
