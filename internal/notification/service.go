package notification

import (
	"errors"
	"time"

	"github.com/nothing-4413/saas/internal/outbox"
)

var ErrInvalidInput = errors.New("invalid notification input")

type Sender func(outbox.Event) error
type Service struct{ events *outbox.Service }

func NewService(events *outbox.Service) *Service { return &Service{events: events} }

func (s *Service) Enqueue(org, aggregateID, eventType, dedupKey string, payload interface{}) (outbox.Event, error) {
	if s == nil || s.events == nil {
		return outbox.Event{}, ErrInvalidInput
	}
	return s.events.Enqueue(org, "notification", aggregateID, eventType, dedupKey, payload)
}

func (s *Service) DispatchOnce(limit int, sender Sender) (processed, failed int) {
	if s == nil || s.events == nil || sender == nil {
		return 0, 0
	}
	for _, event := range s.events.Claim(limit) {
		if err := sender(event); err != nil {
			_ = s.events.Fail(event.ID, err.Error())
			failed++
			continue
		}
		_ = s.events.Succeed(event.ID)
		processed++
	}
	return processed, failed
}

func (s *Service) RunUntilEmpty(sender Sender) (int, int) {
	processed, failed := 0, 0
	for {
		p, f := s.DispatchOnce(100, sender)
		processed += p
		failed += f
		if p == 0 && f == 0 {
			return processed, failed
		}
		time.Sleep(0)
	}
}
