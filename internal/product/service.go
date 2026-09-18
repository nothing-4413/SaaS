package product

import (
	"errors"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/platform/idgen"
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }
func (s *Service) id() string         { return idgen.New() }
func (s *Service) CreateProduct(org string, in CreateProductInput) (Product, error) {
	n := strings.TrimSpace(in.Name)
	if n == "" {
		return Product{}, ErrInvalidInput
	}
	v := Product{ID: s.id(), OrganizationID: org, Name: n, Description: strings.TrimSpace(in.Description), CreatedAt: s.now().UTC()}
	return v, s.store.CreateProduct(v)
}
func (s *Service) CreateSKU(org, productID string, in CreateSKUInput) (SKU, error) {
	p, e := s.store.GetProduct(productID)
	if e != nil {
		return SKU{}, e
	}
	if p.OrganizationID != org || strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" || in.PriceCents < 0 {
		return SKU{}, ErrInvalidInput
	}
	v := SKU{ID: s.id(), OrganizationID: org, ProductID: productID, Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), PriceCents: in.PriceCents}
	return v, s.store.CreateSKU(v)
}
func (s *Service) CreateWarehouse(org string, in CreateWarehouseInput) (Warehouse, error) {
	n := strings.TrimSpace(in.Name)
	if n == "" {
		return Warehouse{}, ErrInvalidInput
	}
	v := Warehouse{ID: s.id(), OrganizationID: org, Name: n, Address: strings.TrimSpace(in.Address), CreatedAt: s.now().UTC()}
	return v, s.store.CreateWarehouse(v)
}
func (s *Service) ListProducts(org string) []Product     { return s.store.ListProducts(org) }
func (s *Service) ListSKUs(product string) []SKU         { return s.store.ListSKUs(product) }
func (s *Service) ListWarehouses(org string) []Warehouse { return s.store.ListWarehouses(org) }
