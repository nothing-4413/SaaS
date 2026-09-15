package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var ErrInvalidInput = errors.New("invalid outbox event")

type Service struct {
	store       Store
	now         func() time.Time
	seq         uint64
	maxAttempts int
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now, maxAttempts: 5} }
func (s *Service) Enqueue(org, aggregateType, aggregateID, eventType, dedupKey string, payload interface{}) (Event, error) {
	if strings.TrimSpace(org) == "" || strings.TrimSpace(aggregateType) == "" || strings.TrimSpace(aggregateID) == "" || strings.TrimSpace(eventType) == "" || strings.TrimSpace(dedupKey) == "" {
		return Event{}, ErrInvalidInput
	}
	b, e := json.Marshal(payload)
	if e != nil {
		return Event{}, e
	}
	now := s.now().UTC()
	v := Event{ID: fmt.Sprintf("%d-%d", now.UnixNano(), atomic.AddUint64(&s.seq, 1)), OrganizationID: org, AggregateType: aggregateType, AggregateID: aggregateID, Type: eventType, DedupKey: dedupKey, Payload: b, Status: StatusPending, NextAttemptAt: now, CreatedAt: now}
	if e = s.store.Enqueue(v); e != nil {
		return Event{}, e
	}
	return v, nil
}
func (s *Service) Claim(limit int) []Event { return s.store.Claim(limit, s.now().UTC()) }
func (s *Service) Succeed(id string) error { return s.store.MarkPublished(id, s.now().UTC()) }
func (s *Service) Fail(id string, reason string) error {
	v, e := s.store.Get(id)
	if e != nil {
		return e
	}
	terminal := v.Attempts >= s.maxAttempts
	backoff := time.Duration(v.Attempts*v.Attempts) * time.Second
	return s.store.MarkFailed(id, s.now().UTC().Add(backoff), reason, terminal)
}
func (s *Service) Get(id string) (Event, error) { return s.store.Get(id) }
func (s *Service) List(status Status) []Event   { return s.store.List(status) }
