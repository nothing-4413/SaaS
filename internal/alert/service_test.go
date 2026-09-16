package alert

import (
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/outbox"
	"testing"
)

func TestScanEnqueuesLowStockOnce(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	_, _ = inv.Receive("o", "w", "sku", inventory.StockOperationInput{Quantity: 1, IdempotencyKey: "r"})
	events := outbox.NewService(outbox.NewMemoryStore())
	s := NewService(inv, events)
	n, e := s.Scan("o", 2)
	if e != nil || n != 1 {
		t.Fatalf("n=%d e=%v", n, e)
	}
	n, e = s.Scan("o", 2)
	if e != nil || n != 0 {
		t.Fatalf("duplicate alert n=%d e=%v", n, e)
	}
}
