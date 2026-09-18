package inventory

import (
	"errors"
	"sync"
	"time"
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
	CreateDocument(Document) error
	GetDocument(string) (Document, error)
	ListDocuments(string, DocumentType) []Document
	FindDocumentByKey(string, DocumentType, string) (Document, error)
}

type AtomicStore interface {
	Apply(org, warehouse, sku, action string, quantity int64, idempotencyKey string, at time.Time) (Stock, error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	items     map[string]Stock
	documents map[string]Document
	docKeys   map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]Stock), documents: make(map[string]Document), docKeys: make(map[string]string)}
}
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

func documentKey(org string, typ DocumentType, idempotencyKey string) string {
	return org + "\x00" + string(typ) + "\x00" + idempotencyKey
}
func (s *MemoryStore) CreateDocument(v Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := documentKey(v.OrganizationID, v.Type, v.IdempotencyKey)
	if _, ok := s.docKeys[k]; ok {
		return ErrConflict
	}
	s.documents[v.ID] = v
	s.docKeys[k] = v.ID
	return nil
}
func (s *MemoryStore) GetDocument(id string) (Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.documents[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) ListDocuments(org string, typ DocumentType) []Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Document{}
	for _, v := range s.documents {
		if v.OrganizationID == org && (typ == "" || v.Type == typ) {
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) FindDocumentByKey(org string, typ DocumentType, idempotencyKey string) (Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.docKeys[documentKey(org, typ, idempotencyKey)]
	if !ok {
		return Document{}, ErrNotFound
	}
	return s.documents[id], nil
}
