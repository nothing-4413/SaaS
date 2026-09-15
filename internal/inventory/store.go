package inventory

import (
	"errors"
	"sync"
)

var (
	ErrNotFound     = errors.New("stock not found")
	ErrConflict     = errors.New("idempotency key already used")
	ErrInsufficient = errors.New("insufficient available stock")
	ErrInvalidInput = errors.New("invalid input")
)

type Store interface {
	Get(string, string, string) (Stock, error)
	Put(Stock) error
	List(string) []Stock
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Stock
}

func NewMemoryStore() *MemoryStore          { return &MemoryStore{items: make(map[string]Stock)} }
func key(org, warehouse, sku string) string { return org + "\x00" + warehouse + "\x00" + sku }
func (s *MemoryStore) Get(org, warehouse, sku string) (Stock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key(org, warehouse, sku)]
	if !ok {
		return Stock{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) Put(v Stock) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key(v.OrganizationID, v.WarehouseID, v.SKUID)] = v
	return nil
}
func (s *MemoryStore) List(org string) []Stock {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Stock, 0)
	for _, v := range s.items {
		if v.OrganizationID == org {
			out = append(out, v)
		}
	}
	return out
}
