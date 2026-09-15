package notification

import (
	"errors"
	"testing"

	"github.com/nothing-4413/saas/internal/outbox"
)

func TestDispatchMarksSuccessAndRetriesFailure(t *testing.T) {
	events := outbox.NewService(outbox.NewMemoryStore())
	s := NewService(events)
	if _, err := s.Enqueue("org", "order-1", "order.confirmed", "n1", map[string]string{"channel": "email"}); err != nil {
		t.Fatal(err)
	}
	processed, failed := s.DispatchOnce(10, func(outbox.Event) error { return nil })
	if processed != 1 || failed != 0 {
		t.Fatalf("processed=%d failed=%d", processed, failed)
	}

	if _, err := s.Enqueue("org", "order-2", "order.confirmed", "n2", nil); err != nil {
		t.Fatal(err)
	}
	_, failed = s.DispatchOnce(10, func(outbox.Event) error { return errors.New("provider unavailable") })
	if failed != 1 {
		t.Fatalf("failed=%d", failed)
	}
}
