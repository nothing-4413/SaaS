package order

import (
	"errors"
	"sync"
)

var (
	ErrNotFound     = errors.New("order not found")
	ErrConflict     = errors.New("order already exists")
	ErrInvalidInput = errors.New("invalid input")
)

type Store interface {
	Create(Order) error
	Delete(string) error
	Get(string) (Order, error)
	Put(Order) error
	List(string) []Order
	FindByKey(string, string) (Order, error)
}
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Order
	keys  map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: map[string]Order{}, keys: map[string]string{}}
}
func (s *MemoryStore) Create(v Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := v.OrganizationID + "\x00" + v.IdempotencyKey
	if _, ok := s.keys[k]; ok {
		return ErrConflict
	}
	s.items[v.ID] = v
	s.keys[k] = v.ID
	return nil
}
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	delete(s.keys, v.OrganizationID+"\x00"+v.IdempotencyKey)
	return nil
}
func (s *MemoryStore) Get(id string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) Put(v Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[v.ID]; !ok {
		return ErrNotFound
	}
	s.items[v.ID] = v
	return nil
}
func (s *MemoryStore) List(org string) []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Order{}
	for _, v := range s.items {
		if v.OrganizationID == org {
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) FindByKey(org, key string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.keys[org+"\x00"+key]
	if !ok {
		return Order{}, ErrNotFound
	}
	return s.items[id], nil
}
