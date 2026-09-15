package order

import (
	"github.com/nothing-4413/saas/internal/inventory"
	"testing"
)

func TestOrderReserveConfirmAndIdempotency(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku", inventory.StockOperationInput{Quantity: 5, IdempotencyKey: "seed"})
	s := NewService(NewMemoryStore(), inv)
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
}
