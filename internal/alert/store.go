package alert

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("stock alert rule not found")

type Store interface {
	Put(Rule) error
	Get(string) (Rule, error)
	ListEnabled() []Rule
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Rule
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: make(map[string]Rule)} }

func (s *MemoryStore) Put(value Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[value.OrganizationID] = value
	return nil
}

func (s *MemoryStore) Get(org string) (Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.items[org]
	if !ok {
		return Rule{}, ErrNotFound
	}
	return value, nil
}

func (s *MemoryStore) ListEnabled() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := []Rule{}
	for _, value := range s.items {
		if value.Enabled {
			values = append(values, value)
		}
	}
	return values
}
