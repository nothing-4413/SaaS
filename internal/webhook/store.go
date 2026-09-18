package webhook

import (
	"errors"
	"sync"
)

var (
	ErrNotFound = errors.New("webhook subscription not found")
	ErrConflict = errors.New("webhook subscription already exists")
)

type Store interface {
	Create(Subscription) error
	List(string) []Subscription
	Delete(string, string) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Subscription
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: make(map[string]Subscription)} }

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
