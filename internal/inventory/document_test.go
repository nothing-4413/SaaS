package inventory

import (
	"sync"
	"testing"
)

func TestReceiptAndIssueDocumentsAreIdempotent(t *testing.T) {
	s := NewService(NewMemoryStore())
	in := DocumentInput{IdempotencyKey: "r1", Lines: []DocumentLine{{WarehouseID: "w", SKUID: "sku", Quantity: 5}}}
	a, e := s.CreateReceipt("o", in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.CreateReceipt("o", in)
	if e != nil || a.ID != b.ID {
		t.Fatalf("receipt idempotency failed: %v", e)
	}
	if _, e = s.CreateIssue("o", DocumentInput{IdempotencyKey: "i1", Lines: []DocumentLine{{WarehouseID: "w", SKUID: "sku", Quantity: 2}}}); e != nil {
		t.Fatal(e)
	}
	v, _ := s.Get("o", "w", "sku")
	if v.OnHand != 3 {
		t.Fatalf("on hand=%d", v.OnHand)
	}
}

func TestConcurrentDocumentsHaveUniqueIDs(t *testing.T) {
	s := NewService(NewMemoryStore())
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, e := s.CreateReceipt("o", DocumentInput{IdempotencyKey: string(rune(i + 100)), Lines: []DocumentLine{{WarehouseID: "w", SKUID: "s", Quantity: 1}}})
			if e == nil {
				ids <- v.ID
			}
		}(i)
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}
