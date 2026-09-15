package inventory

import (
	"sync"
	"testing"
)

func TestConcurrentReserveAndIdempotency(t *testing.T) {
	s := NewService(NewMemoryStore())
	if _, e := s.Receive("o", "w", "sku", StockOperationInput{Quantity: 10, IdempotencyKey: "in-1"}); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := s.Reserve("o", "w", "sku", StockOperationInput{Quantity: 1, IdempotencyKey: string(rune(i + 100))})
			if e == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if success != 10 {
		t.Fatalf("reserved %d, want 10", success)
	}
	a, e := s.Reserve("o", "w", "sku", StockOperationInput{Quantity: 2, IdempotencyKey: "same"})
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Reserve("o", "w", "sku", StockOperationInput{Quantity: 2, IdempotencyKey: "same"})
	if e != nil || a.Reserved != b.Reserved {
		t.Fatalf("idempotency failed: %#v %#v %v", a, b, e)
	}
}

func TestDeductConsumesReservedStock(t *testing.T) {
	s := NewService(NewMemoryStore())
	if _, e := s.Receive("o", "w", "sku", StockOperationInput{Quantity: 5, IdempotencyKey: "r"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Reserve("o", "w", "sku", StockOperationInput{Quantity: 3, IdempotencyKey: "v"}); e != nil {
		t.Fatal(e)
	}
	v, e := s.Deduct("o", "w", "sku", StockOperationInput{Quantity: 2, IdempotencyKey: "d"})
	if e != nil {
		t.Fatal(e)
	}
	if v.OnHand != 3 || v.Reserved != 1 || v.Available != 2 {
		t.Fatalf("unexpected stock: %+v", v)
	}
}
