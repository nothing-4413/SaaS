package order

import (
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/outbox"
	"testing"
	"time"
)

func TestOrderCreateEventFailureRollsBackReservation(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "s", inventory.StockOperationInput{Quantity: 2, IdempotencyKey: "seed"})
	s := NewService(NewMemoryStore(), inv)
	events := outbox.NewService(failingStore{})
	s.SetEventService(events)
	if _, e := s.Create("o", CreateInput{IdempotencyKey: "x", Lines: []Line{{WarehouseID: "w", SKUID: "s", Quantity: 1}}}); e == nil {
		t.Fatal("expected event failure")
	}
	v, _ := inv.Get("o", "w", "s")
	if v.Reserved != 0 || v.Available != 2 {
		t.Fatalf("reservation not rolled back: %+v", v)
	}
}

type failingStore struct{}

func (failingStore) Enqueue(outbox.Event) error                       { return outbox.ErrConflict }
func (failingStore) Claim(int, time.Time) []outbox.Event              { return nil }
func (failingStore) MarkPublished(string, time.Time) error            { return nil }
func (failingStore) MarkFailed(string, time.Time, string, bool) error { return nil }
func (failingStore) Get(string) (outbox.Event, error)                 { return outbox.Event{}, outbox.ErrNotFound }
func (failingStore) List(outbox.Status) []outbox.Event                { return nil }
