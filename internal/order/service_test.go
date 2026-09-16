package order

import (
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/outbox"
	"testing"
)

func TestOrderReserveConfirmAndIdempotency(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku", inventory.StockOperationInput{Quantity: 5, IdempotencyKey: "seed"})
	s := NewService(NewMemoryStore(), inv)
	events := outbox.NewService(outbox.NewMemoryStore())
	s.SetEventService(events)
	in := CreateInput{IdempotencyKey: "order-1", Lines: []Line{{SKUID: "sku", WarehouseID: "w", Quantity: 2, UnitPriceCents: 100}}}
	a, e := s.Create("o", in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Create("o", in)
	if e != nil || a.ID != b.ID {
		t.Fatalf("idempotency failed: %v", e)
	}
	if _, e = s.Confirm("o", a.ID); e != nil {
		t.Fatal(e)
	}
	v, _ := inv.Get("o", "w", "sku")
	if v.OnHand != 3 || v.Reserved != 0 {
		t.Fatalf("unexpected stock: %+v", v)
	}
	if got := len(events.List("")); got != 2 {
		t.Fatalf("expected create+confirm events, got %d", got)
	}
}

func TestCancelReleasesReservedStock(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku", inventory.StockOperationInput{Quantity: 5, IdempotencyKey: "seed"})
	s := NewService(NewMemoryStore(), inv)
	v, err := s.Create("o", CreateInput{IdempotencyKey: "cancel-me", Lines: []Line{{SKUID: "sku", WarehouseID: "w", Quantity: 3, UnitPriceCents: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Cancel("o", v.ID); err != nil {
		t.Fatal(err)
	}
	stock, _ := inv.Get("o", "w", "sku")
	if stock.OnHand != 5 || stock.Reserved != 0 || stock.Available != 5 {
		t.Fatalf("unexpected stock after cancel: %+v", stock)
	}
}

func TestCreateRollsBackEarlierReservationsOnFailure(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku-a", inventory.StockOperationInput{Quantity: 2, IdempotencyKey: "seed-a"})
	s := NewService(NewMemoryStore(), inv)
	_, err := s.Create("o", CreateInput{IdempotencyKey: "rollback", Lines: []Line{
		{SKUID: "sku-a", WarehouseID: "w", Quantity: 2, UnitPriceCents: 10},
		{SKUID: "sku-b", WarehouseID: "w", Quantity: 1, UnitPriceCents: 20},
	}})
	if err == nil {
		t.Fatal("expected second line to fail")
	}
	stock, _ := inv.Get("o", "w", "sku-a")
	if stock.Reserved != 0 || stock.Available != 2 {
		t.Fatalf("reservation was not rolled back: %+v", stock)
	}
}
