package audit

import (
	"errors"
	"sync"
	"time"
)

var ErrInvalidInput = errors.New("invalid audit entry")

type Store interface {
	Append(Entry) error
	List(string) []Entry
}
type MemoryStore struct {
	mu    sync.RWMutex
	items []Entry
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: []Entry{}} }
func (s *MemoryStore) Append(v Entry) error {
	if v.OrganizationID == "" || v.Action == "" || v.ResourceType == "" || v.ResourceID == "" || v.CreatedAt.IsZero() {
		return ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, v)
	return nil
}
func (s *MemoryStore) List(org string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Entry{}
	for _, v := range s.items {
		if v.OrganizationID == org {
			v.CreatedAt = v.CreatedAt.In(time.UTC)
			out = append(out, v)
		}
	}
	return out
}
