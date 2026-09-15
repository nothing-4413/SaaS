package inventory

import (
	"strings"
	"sync"
	"time"
)

type operation struct {
	org, warehouse, sku, action string
	quantity                    int64
}
type Service struct {
	store      Store
	now        func() time.Time
	mu         sync.Mutex
	operations map[string]operation
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now, operations: make(map[string]operation)}
}
func (s *Service) Get(org, warehouse, sku string) (Stock, error) {
	return s.store.Get(org, warehouse, sku)
}
func (s *Service) List(org string) []Stock { return s.store.List(org) }

func (s *Service) apply(org, warehouse, sku, action string, in StockOperationInput) (Stock, error) {
	if strings.TrimSpace(org) == "" || strings.TrimSpace(warehouse) == "" || strings.TrimSpace(sku) == "" || in.Quantity <= 0 || strings.TrimSpace(in.IdempotencyKey) == "" {
		return Stock{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	opKey := org + "\x00" + in.IdempotencyKey
	if prev, ok := s.operations[opKey]; ok {
		if prev.warehouse != warehouse || prev.sku != sku || prev.action != action || prev.quantity != in.Quantity {
			return Stock{}, ErrConflict
		}
		return s.store.Get(org, warehouse, sku)
	}
	v, err := s.store.Get(org, warehouse, sku)
	if err != nil {
		v = Stock{OrganizationID: org, WarehouseID: warehouse, SKUID: sku}
	}
	switch action {
	case "receive":
		v.OnHand += in.Quantity
	case "reserve":
		if v.OnHand-v.Reserved < in.Quantity {
			return Stock{}, ErrInsufficient
		}
		v.Reserved += in.Quantity
	case "release":
		if v.Reserved < in.Quantity {
			return Stock{}, ErrInvalidInput
		}
		v.Reserved -= in.Quantity
	case "deduct":
		if v.OnHand < in.Quantity {
			return Stock{}, ErrInsufficient
		}
		v.OnHand -= in.Quantity
		if v.Reserved >= in.Quantity {
			v.Reserved -= in.Quantity
		}
	default:
		return Stock{}, ErrInvalidInput
	}
	v.Available = v.OnHand - v.Reserved
	v.UpdatedAt = s.now().UTC()
	if err = s.store.Put(v); err != nil {
		return Stock{}, err
	}
	s.operations[opKey] = operation{org: org, warehouse: warehouse, sku: sku, action: action, quantity: in.Quantity}
	return v, nil
}
func (s *Service) Receive(org, warehouse, sku string, in StockOperationInput) (Stock, error) {
	return s.apply(org, warehouse, sku, "receive", in)
}
func (s *Service) Reserve(org, warehouse, sku string, in StockOperationInput) (Stock, error) {
	return s.apply(org, warehouse, sku, "reserve", in)
}
func (s *Service) Release(org, warehouse, sku string, in StockOperationInput) (Stock, error) {
	return s.apply(org, warehouse, sku, "release", in)
}
func (s *Service) Deduct(org, warehouse, sku string, in StockOperationInput) (Stock, error) {
	return s.apply(org, warehouse, sku, "deduct", in)
}
