package inventory

import "testing"

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
