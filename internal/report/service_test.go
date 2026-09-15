package report

import (
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"testing"
)

func TestSummaryIsTenantScoped(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("a", "w", "sku", inventory.StockOperationInput{Quantity: 5, IdempotencyKey: "seed"})
	orders := order.NewService(order.NewMemoryStore(), inv)
	_, err := orders.Create("a", order.CreateInput{IdempotencyKey: "o1", Lines: []order.Line{{SKUID: "sku", WarehouseID: "w", Quantity: 2, UnitPriceCents: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(orders, inv)
	v, err := s.Summary("a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if v.Orders.Total != 1 || v.Inventory.OnHand != 5 || v.Inventory.Reserved != 2 || v.Inventory.Available != 3 {
		t.Fatalf("unexpected summary: %+v", v)
	}
	if v.Inventory.LowStock != 0 {
		t.Fatalf("unexpected low stock count: %d", v.Inventory.LowStock)
	}
	if other, _ := s.Summary("b", 0); other.Orders.Total != 0 || other.Inventory.SKUs != 0 {
		t.Fatalf("cross-tenant leakage: %+v", other)
	}
}
