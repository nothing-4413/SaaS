package outbox

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("outbox event not found")
	ErrConflict     = errors.New("outbox deduplication conflict")
	ErrInvalidState = errors.New("invalid outbox event state transition")
)

const maxClaimAttempts = 5

type Store interface {
	Enqueue(Event) error
	Claim(limit int, now time.Time) []Event
	MarkPublished(id string, at time.Time) error
	MarkFailed(id string, next time.Time, reason string, terminal bool) error
	Get(id string) (Event, error)
	List(status Status) []Event
}

type MemoryStore struct {
	mu    sync.Mutex
	items map[string]Event
	byKey map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: map[string]Event{}, byKey: map[string]string{}}
}
func dedupKey(v Event) string {
	return v.OrganizationID + "\x00" + v.AggregateType + "\x00" + v.AggregateID + "\x00" + v.Type + "\x00" + v.DedupKey
}
func (s *MemoryStore) Enqueue(v Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := dedupKey(v)
	if old, ok := s.byKey[key]; ok {
		if old == v.ID {
			return nil
		}
		return ErrConflict
	}
	s.items[v.ID] = v
	s.byKey[key] = v.ID
	return nil
}
func (s *MemoryStore) Claim(limit int, now time.Time) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		return []Event{}
	}
	out := make([]Event, 0, limit)
	for id, v := range s.items {
		if len(out) >= limit {
			break
		}
		claimable := v.Status == StatusPending && v.Attempts < maxClaimAttempts && !v.NextAttemptAt.After(now)
		if v.Status == StatusProcessing && !v.ClaimedUntil.After(now) {
			if v.Attempts >= maxClaimAttempts {
				v.Status = StatusFailed
				v.LastError = "processing lease expired after maximum attempts"
				v.ClaimedUntil = time.Time{}
				s.items[id] = v
				continue
			}
			claimable = true
		}
		if claimable {
			v.Status = StatusProcessing
			v.Attempts++
			v.ClaimedUntil = now.Add(5 * time.Minute)
			s.items[id] = v
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) MarkPublished(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	if v.Status != StatusProcessing {
		return ErrInvalidState
	}
	v.Status = StatusPublished
	v.ClaimedUntil = time.Time{}
	v.PublishedAt = &at
	v.LastError = ""
	s.items[id] = v
	return nil
}
func (s *MemoryStore) MarkFailed(id string, next time.Time, reason string, terminal bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	if v.Status != StatusProcessing {
		return ErrInvalidState
	}
	v.LastError = reason
	v.NextAttemptAt = next
	v.ClaimedUntil = time.Time{}
	if terminal {
		v.Status = StatusFailed
	} else {
		v.Status = StatusPending
	}
	s.items[id] = v
	return nil
}
func (s *MemoryStore) Get(id string) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return Event{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) List(status Status) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Event{}
	for _, v := range s.items {
		if status == "" || v.Status == status {
			out = append(out, v)
		}
	}
	return out
}
