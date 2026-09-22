package product

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

type Store interface {
	CreateProduct(Product) error
	UpdateProduct(Product) error
	ListProducts(string) []Product
	GetProduct(string) (Product, error)
	CreateSKU(SKU) error
	UpdateSKU(SKU) error
	GetSKU(string) (SKU, error)
	ListSKUs(string) []SKU
	CreateWarehouse(Warehouse) error
	UpdateWarehouse(Warehouse) error
	ListWarehouses(string) []Warehouse
}

type MemoryStore struct {
	mu         sync.RWMutex
	products   map[string]Product
	skus       map[string]SKU
	warehouses map[string]Warehouse
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{products: map[string]Product{}, skus: map[string]SKU{}, warehouses: map[string]Warehouse{}}
}
func (s *MemoryStore) CreateProduct(v Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[v.ID]; ok {
		return ErrConflict
	}
	s.products[v.ID] = v
	return nil
}
func (s *MemoryStore) UpdateProduct(v Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.products[v.ID]
	if !ok || old.OrganizationID != v.OrganizationID {
		return ErrNotFound
	}
	for id, x := range s.products {
		if id != v.ID && x.OrganizationID == v.OrganizationID && x.Name == v.Name {
			return ErrConflict
		}
	}
	s.products[v.ID] = v
	return nil
}
func (s *MemoryStore) ListProducts(org string) []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Product{}
	for _, v := range s.products {
		if v.OrganizationID == org {
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) GetProduct(id string) (Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.products[id]
	if !ok {
		return Product{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) CreateSKU(v SKU) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.skus {
		if x.OrganizationID == v.OrganizationID && x.Code == v.Code {
			return ErrConflict
		}
	}
	s.skus[v.ID] = v
	return nil
}
func (s *MemoryStore) UpdateSKU(v SKU) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.skus[v.ID]
	if !ok || old.OrganizationID != v.OrganizationID {
		return ErrNotFound
	}
	for id, x := range s.skus {
		if id != v.ID && x.OrganizationID == v.OrganizationID && x.Code == v.Code {
			return ErrConflict
		}
	}
	s.skus[v.ID] = v
	return nil
}
func (s *MemoryStore) GetSKU(id string) (SKU, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.skus[id]
	if !ok {
		return SKU{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) ListSKUs(productID string) []SKU {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []SKU{}
	for _, v := range s.skus {
		if v.ProductID == productID {
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) CreateWarehouse(v Warehouse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.warehouses {
		if x.OrganizationID == v.OrganizationID && x.Name == v.Name {
			return ErrConflict
		}
	}
	s.warehouses[v.ID] = v
	return nil
}
func (s *MemoryStore) UpdateWarehouse(v Warehouse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.warehouses[v.ID]
	if !ok || old.OrganizationID != v.OrganizationID {
		return ErrNotFound
	}
	for id, x := range s.warehouses {
		if id != v.ID && x.OrganizationID == v.OrganizationID && x.Name == v.Name {
			return ErrConflict
		}
	}
	s.warehouses[v.ID] = v
	return nil
}
func (s *MemoryStore) ListWarehouses(org string) []Warehouse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Warehouse{}
	for _, v := range s.warehouses {
		if v.OrganizationID == org {
			out = append(out, v)
		}
	}
	return out
}
