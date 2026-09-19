package webhook

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("webhook subscription not found")
	ErrConflict = errors.New("webhook subscription already exists")
)

type Store interface {
	Create(Subscription) error
	List(string) []Subscription
	Delete(string, string) error
	IsDelivered(eventID, subscriptionID string) bool
	MarkDelivered(eventID, subscriptionID string, at time.Time) error
}

type MemoryStore struct {
	mu        sync.RWMutex
	items     map[string]Subscription
	delivered map[string]time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]Subscription), delivered: make(map[string]time.Time)}
}

func (s *MemoryStore) Create(value Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[value.ID]; ok {
		return ErrConflict
	}
	s.items[value.ID] = value
	return nil
}

func (s *MemoryStore) List(org string) []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := []Subscription{}
	for _, value := range s.items {
		if value.OrganizationID == org {
			value.EventTypes = append([]string(nil), value.EventTypes...)
			values = append(values, value)
		}
	}
	return values
}

func (s *MemoryStore) Delete(org, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[id]
	if !ok || value.OrganizationID != org {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func deliveryKey(eventID, subscriptionID string) string { return eventID + "\x00" + subscriptionID }

func (s *MemoryStore) IsDelivered(eventID, subscriptionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.delivered[deliveryKey(eventID, subscriptionID)]
	return ok
}

func (s *MemoryStore) MarkDelivered(eventID, subscriptionID string, at time.Time) error {
	if eventID == "" || subscriptionID == "" {
		return ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delivered[deliveryKey(eventID, subscriptionID)] = at
	return nil
}
