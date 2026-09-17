package outbox

import (
	"testing"
	"time"
)

func TestStateTransitionsRequireClaim(t *testing.T) {
	s := NewMemoryStore()
	e := Event{ID: "e", OrganizationID: "o", AggregateType: "order", AggregateID: "1", Type: "created", DedupKey: "k", Status: StatusPending, NextAttemptAt: time.Now()}
	if err := s.Enqueue(e); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkPublished("e", time.Now()); err != ErrInvalidState {
		t.Fatalf("pending publish err=%v", err)
	}
	if len(s.Claim(1, time.Now())) != 1 {
		t.Fatal("claim missing")
	}
	if err := s.MarkPublished("e", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed("e", time.Now(), "late", false); err != ErrInvalidState {
		t.Fatalf("published failure err=%v", err)
	}
}
