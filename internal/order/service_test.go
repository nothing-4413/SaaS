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

func TestOrderLifecycleAndRefundRestoresStock(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku", inventory.StockOperationInput{Quantity: 5, IdempotencyKey: "seed-lifecycle"})
	s := NewService(NewMemoryStore(), inv)
	v, err := s.Create("o", CreateInput{IdempotencyKey: "lifecycle", Lines: []Line{{SKUID: "sku", WarehouseID: "w", Quantity: 2, UnitPriceCents: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pay("o", v.ID); err == nil {
		t.Fatal("payment before confirmation should fail")
	}
	steps := []struct {
		name string
		call func(string, string) (Order, error)
	}{
		{name: "confirm", call: s.Confirm},
		{name: "pay", call: s.Pay},
		{name: "ship", call: s.Ship},
		{name: "complete", call: s.Complete},
	}
	for _, step := range steps {
		v, err = step.call("o", v.ID)
		if err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}
	stock, _ := inv.Get("o", "w", "sku")
	if stock.OnHand != 3 || stock.Reserved != 0 {
		t.Fatalf("unexpected consumed stock: %+v", stock)
	}
	v, err = s.Refund("o", v.ID)
	if err != nil || v.Status != StatusRefunded {
		t.Fatalf("refund: value=%+v err=%v", v, err)
	}
	stock, _ = inv.Get("o", "w", "sku")
	if stock.OnHand != 5 || stock.Reserved != 0 || stock.Available != 5 {
		t.Fatalf("unexpected restored stock: %+v", stock)
	}
	if duplicate, err := s.Refund("o", v.ID); err != nil || duplicate.Status != StatusRefunded {
		t.Fatalf("duplicate refund should be idempotent: value=%+v err=%v", duplicate, err)
	}
}

func TestConfirmRollsBackEarlierLinesWhenLaterLineFails(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "a", inventory.StockOperationInput{Quantity: 2, IdempotencyKey: "seed-a-confirm"})
	_, _ = inv.Receive("o", "w", "b", inventory.StockOperationInput{Quantity: 2, IdempotencyKey: "seed-b-confirm"})
	s := NewService(NewMemoryStore(), inv)
	v, err := s.Create("o", CreateInput{IdempotencyKey: "partial-confirm", Lines: []Line{
		{SKUID: "a", WarehouseID: "w", Quantity: 1},
		{SKUID: "b", WarehouseID: "w", Quantity: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = inv.Release("o", "w", "b", inventory.StockOperationInput{Quantity: 1, IdempotencyKey: "corrupt-reservation"})
	if _, err = s.Confirm("o", v.ID); err == nil {
		t.Fatal("expected confirmation to fail")
	}
	stockA, _ := inv.Get("o", "w", "a")
	if stockA.OnHand != 2 || stockA.Reserved != 1 || stockA.Available != 1 {
		t.Fatalf("first line was not rolled back: %+v", stockA)
	}
	stored, _ := s.Get("o", v.ID)
	if stored.Status != StatusPending {
		t.Fatalf("status changed after failed confirmation: %s", stored.Status)
	}
}
