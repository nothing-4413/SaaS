package audit

import "testing"

func TestAuditIsTenantScoped(t *testing.T) {
	s := NewService(NewMemoryStore())
	if _, e := s.Record("a", "u", "create", "order", "1", nil); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Record("b", "u", "create", "order", "2", nil); e != nil {
		t.Fatal(e)
	}
	if got := len(s.List("a")); got != 1 {
		t.Fatalf("got %d entries", got)
	}
}
