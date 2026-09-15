package product

import "testing"

func TestSKUProductTenantAndCode(t *testing.T) {
	s := NewService(NewMemoryStore())
	p, _ := s.CreateProduct("a", CreateProductInput{Name: "Widget"})
	if _, e := s.CreateSKU("b", p.ID, CreateSKUInput{Code: "W", Name: "W"}); e != ErrInvalidInput {
		t.Fatalf("cross tenant sku: %v", e)
	}
	if _, e := s.CreateSKU("a", p.ID, CreateSKUInput{Code: "W", Name: "W"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CreateSKU("a", p.ID, CreateSKUInput{Code: "W", Name: "W2"}); e != ErrConflict {
		t.Fatalf("duplicate code: %v", e)
	}
}
