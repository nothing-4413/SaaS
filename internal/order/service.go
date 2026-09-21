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
	total := int64(0)
	for _, l := range in.Lines {
		if strings.TrimSpace(l.SKUID) == "" || strings.TrimSpace(l.WarehouseID) == "" || l.Quantity <= 0 || l.UnitPriceCents < 0 {
			return Order{}, ErrInvalidInput
		}
		total += l.Quantity * l.UnitPriceCents
	}
	id := s.id()
	now := s.now().UTC()
	v := Order{ID: id, OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Status: StatusPending, Lines: append([]Line(nil), in.Lines...), TotalCents: total, CreatedAt: now, UpdatedAt: now}
	if store, ok := s.store.(TransactionalStore); ok {
		return store.CreateAtomic(v)
	}
	if old, e := s.store.FindByKey(org, in.IdempotencyKey); e == nil {
		return old, nil
	}
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
	return s.transition(org, id, StatusConfirmed)
}
func (s *Service) Pay(org, id string) (Order, error) {
	return s.transition(org, id, StatusPaid)
}
func (s *Service) Ship(org, id string) (Order, error) {
	return s.transition(org, id, StatusShipped)
}
func (s *Service) Complete(org, id string) (Order, error) {
	return s.transition(org, id, StatusCompleted)
}
func (s *Service) Refund(org, id string) (Order, error) {
	return s.transition(org, id, StatusRefunded)
}
func (s *Service) Cancel(org, id string) (Order, error) {
	return s.transition(org, id, StatusCancelled)
}

func (s *Service) transition(org, id string, target Status) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if store, ok := s.store.(TransactionalStore); ok {
		return store.TransitionAtomic(org, id, target, s.now().UTC())
	}
	v, e := s.Get(org, id)
	if e != nil {
		return Order{}, e
	}
	if v.Status == target {
		return v, nil
	}
	if !CanTransition(v.Status, target) {
		return Order{}, ErrInvalidInput
	}
	previous := v
	completed := 0
	for i, l := range v.Lines {
		if action := transitionAction(target); action != "" {
			if _, e = s.applyInventoryAction(org, l, action, id+":"+string(target)+":"+fmt.Sprint(i)); e != nil {
				s.rollbackInventory(org, v.Lines[:completed], target, id+":action-rollback:")
				return Order{}, e
			}
			completed++
		}
	}
	v.Status = target
	v.UpdatedAt = s.now().UTC()
	if e = s.store.Put(v); e != nil {
		s.rollbackInventory(org, v.Lines, target, id+":put-rollback:")
		return Order{}, e
	}
	if e = s.emit(v, "order."+string(target)); e != nil {
		_ = s.store.Put(previous)
		s.rollbackInventory(org, v.Lines, target, id+":event-rollback:")
		return Order{}, e
	}
	return v, nil
}

func transitionAction(target Status) string {
	switch target {
	case StatusConfirmed:
		return "consume_reserved"
	case StatusCancelled:
		return "release"
	case StatusRefunded:
		return "receive"
	default:
		return ""
	}
}

func (s *Service) applyInventoryAction(org string, line Line, action, key string) (inventory.Stock, error) {
	in := inventory.StockOperationInput{Quantity: line.Quantity, IdempotencyKey: key}
	switch action {
	case "consume_reserved":
		return s.inventory.ConsumeReserved(org, line.WarehouseID, line.SKUID, in)
	case "release":
		return s.inventory.Release(org, line.WarehouseID, line.SKUID, in)
	case "receive":
		return s.inventory.Receive(org, line.WarehouseID, line.SKUID, in)
	default:
		return inventory.Stock{}, ErrInvalidInput
	}
}

func (s *Service) rollbackInventory(org string, lines []Line, target Status, prefix string) {
	for i, l := range lines {
		key := prefix + fmt.Sprint(i)
		switch target {
		case StatusConfirmed:
			_, _ = s.inventory.Receive(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: key + ":receive"})
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: key + ":reserve"})
		case StatusCancelled:
			_, _ = s.inventory.Reserve(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: key + ":reserve"})
		case StatusRefunded:
			_, _ = s.inventory.Deduct(org, l.WarehouseID, l.SKUID, inventory.StockOperationInput{Quantity: l.Quantity, IdempotencyKey: key + ":deduct"})
		}
	}
}
