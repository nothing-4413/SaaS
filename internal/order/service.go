package order

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/idgen"
)

type Service struct {
	store     Store
	inventory *inventory.Service
	now       func() time.Time
	mu        sync.Mutex
	events    *outbox.Service
}

func NewService(store Store, inv *inventory.Service) *Service {
	return &Service{store: store, inventory: inv, now: time.Now}
}
func (s *Service) SetEventService(events *outbox.Service) { s.events = events }
func (s *Service) emit(v Order, eventType string) error {
	if s.events == nil {
		return nil
	}
	_, e := s.events.Enqueue(v.OrganizationID, "order", v.ID, eventType, eventType, v)
	return e
}
func (s *Service) id() string { return idgen.New() }
func (s *Service) Create(org string, in CreateInput) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(org) == "" || strings.TrimSpace(in.IdempotencyKey) == "" || len(in.Lines) == 0 {
		return Order{}, ErrInvalidInput
	}
	if old, e := s.store.FindByKey(org, in.IdempotencyKey); e == nil {
		return old, nil
	}
	total := int64(0)
	for _, l := range in.Lines {
		if strings.TrimSpace(l.SKUID) == "" || strings.TrimSpace(l.WarehouseID) == "" || l.Quantity <= 0 || l.UnitPriceCents < 0 {
			return Order{}, ErrInvalidInput
		}
		total += l.Quantity * l.UnitPriceCents
	}
	id := s.id()
	reserved := 0
	for i, l := range in.Lines {
		_, e := s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":reserve:" + fmt.Sprint(i)})
		if e != nil {
			for j := 0; j < reserved; j++ {
				p := in.Lines[j]
				_, _ = s.inventory.Release(org, p.WarehouseID, p.SKUID, inventory.StockOperationInput{Quantity: p.Quantity, IdempotencyKey: id + ":rollback:" + fmt.Sprint(j)})
			}
			return Order{}, e
		}
		reserved++
	}
	now := s.now().UTC()
	v := Order{ID: id, OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Status: StatusPending, Lines: append([]Line(nil), in.Lines...), TotalCents: total, CreatedAt: now, UpdatedAt: now}
	if e := s.store.Create(v); e != nil {
		for i, l := range v.Lines {
			_, _ = s.inventory.Release(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: v.ID + ":store-rollback:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	if e := s.emit(v, "order.created"); e != nil {
		_ = s.store.Delete(v.ID)
		for i, l := range v.Lines {
			_, _ = s.inventory.Release(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: v.ID + ":event-rollback:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	return v, nil
}
func (s *Service) Get(org, id string) (Order, error) {
	v, e := s.store.Get(id)
	if e != nil || v.OrganizationID != org {
		return Order{}, ErrNotFound
	}
	return v, nil
}
func (s *Service) List(org string) []Order { return s.store.List(org) }
func (s *Service) Confirm(org, id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, e := s.Get(org, id)
	if e != nil {
		return Order{}, e
	}
	if v.Status == StatusConfirmed {
		return v, nil
	}
	if v.Status == StatusCancelled {
		return Order{}, ErrInvalidInput
	}
	previous := v
	for i, l := range v.Lines {
		if _, e = s.inventory.ConsumeReserved(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":deduct:" + fmt.Sprint(i)}); e != nil {
			return Order{}, e
		}
	}
	v.Status = StatusConfirmed
	v.UpdatedAt = s.now().UTC()
	if e = s.store.Put(v); e != nil {
		for i, l := range v.Lines {
			_, _ = s.inventory.Receive(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":put-rollback-receive:" + fmt.Sprint(i)})
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":put-rollback-reserve:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	if e = s.emit(v, "order.confirmed"); e != nil {
		_ = s.store.Put(previous)
		for i, l := range v.Lines {
			_, _ = s.inventory.Receive(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":event-rollback-receive:" + fmt.Sprint(i)})
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":event-rollback-reserve:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	return v, nil
}
func (s *Service) Cancel(org, id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, e := s.Get(org, id)
	if e != nil {
		return Order{}, e
	}
	if v.Status == StatusCancelled {
		return v, nil
	}
	if v.Status == StatusConfirmed {
		return Order{}, ErrInvalidInput
	}
	previous := v
	for i, l := range v.Lines {
		if _, e = s.inventory.Release(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":cancel:" + fmt.Sprint(i)}); e != nil {
			return Order{}, e
		}
	}
	v.Status = StatusCancelled
	v.UpdatedAt = s.now().UTC()
	if e = s.store.Put(v); e != nil {
		for i, l := range v.Lines {
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":put-rollback-reserve:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	if e = s.emit(v, "order.cancelled"); e != nil {
		_ = s.store.Put(previous)
		for i, l := range v.Lines {
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: id + ":event-rollback-reserve:" + fmt.Sprint(i)})
		}
		return Order{}, e
	}
	return v, nil
}
