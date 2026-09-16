package outbox

import (
	"testing"
	"time"
)

func TestEnqueueClaimAndDeduplicate(t *testing.T) {
	s := NewService(NewMemoryStore())
	e, err := s.Enqueue("org", "order", "o1", "order.created", "create", map[string]string{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enqueue("org", "order", "o1", "order.created", "create", map[string]string{"x": "z"}); err != ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	claimed := s.Claim(10)
	if len(claimed) != 1 || claimed[0].ID != e.ID {
		t.Fatalf("claimed=%+v", claimed)
	}
	if err = s.Succeed(e.ID); err != nil {
		t.Fatal(err)
	}
	v, _ := s.Get(e.ID)
	if v.Status != StatusPublished {
		t.Fatalf("status=%s", v.Status)
	}
}
func TestFailedEventRetriesThenTerminates(t *testing.T) {
	s := NewService(NewMemoryStore())
	e, _ := s.Enqueue("org", "order", "o2", "order.created", "create", nil)
	for i := 0; i < 5; i++ {
		claimed := s.Claim(1)
		if len(claimed) == 0 {
			t.Fatal("event not claimable")
		}
		if err := s.Fail(e.ID, "broker unavailable"); err != nil {
			t.Fatal(err)
		}
		v, _ := s.Get(e.ID)
		if i < 4 && v.Status != StatusPending {
			t.Fatalf("attempt %d status=%s", i, v.Status)
		}
		if i < 4 {
			v.NextAttemptAt = time.Time{}
			_ = s.store.MarkFailed(e.ID, time.Time{}, v.LastError, false)
		}
	}
	v, _ := s.Get(e.ID)
	if v.Status != StatusFailed {
		t.Fatalf("expected terminal failure, got %s", v.Status)
	}
}

func TestProcessingLeaseCanBeReclaimed(t *testing.T) {
	s := NewService(NewMemoryStore())
	e, _ := s.Enqueue("org", "order", "lease", "created", "k", nil)
	first := s.Claim(1)
	if len(first) != 1 {
		t.Fatal("first claim missing")
	}
	store := s.store.(*MemoryStore)
	store.mu.Lock()
	v := store.items[e.ID]
	v.ClaimedUntil = time.Now().Add(-time.Second)
	store.items[e.ID] = v
	store.mu.Unlock()
	second := s.Claim(1)
	if len(second) != 1 || second[0].ID != e.ID {
		t.Fatal("expired lease was not reclaimed")
	}
}
