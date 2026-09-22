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
func (s *Service) UpdateProduct(org, id string, in CreateProductInput) (Product, error) {
	v, err := s.store.GetProduct(id)
	if err != nil {
		return Product{}, err
	}
	if v.OrganizationID != org || strings.TrimSpace(in.Name) == "" {
		return Product{}, ErrInvalidInput
	}
	v.Name, v.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	return v, s.store.UpdateProduct(v)
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
func (s *Service) UpdateSKU(org, id string, in CreateSKUInput) (SKU, error) {
	v, err := s.store.GetSKU(id)
	if err != nil {
		return SKU{}, err
	}
	if v.OrganizationID != org || strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" || in.PriceCents < 0 {
		return SKU{}, ErrInvalidInput
	}
	v.Code, v.Name, v.PriceCents = strings.TrimSpace(in.Code), strings.TrimSpace(in.Name), in.PriceCents
	return v, s.store.UpdateSKU(v)
}
func (s *Service) CreateWarehouse(org string, in CreateWarehouseInput) (Warehouse, error) {
	n := strings.TrimSpace(in.Name)
	if n == "" {
		return Warehouse{}, ErrInvalidInput
	}
	v := Warehouse{ID: s.id(), OrganizationID: org, Name: n, Address: strings.TrimSpace(in.Address), CreatedAt: s.now().UTC()}
	return v, s.store.CreateWarehouse(v)
}
func (s *Service) UpdateWarehouse(org, id string, in CreateWarehouseInput) (Warehouse, error) {
	for _, v := range s.store.ListWarehouses(org) {
		if v.ID == id {
			if strings.TrimSpace(in.Name) == "" {
				return Warehouse{}, ErrInvalidInput
			}
			v.Name, v.Address = strings.TrimSpace(in.Name), strings.TrimSpace(in.Address)
			return v, s.store.UpdateWarehouse(v)
		}
	}
	return Warehouse{}, ErrNotFound
}
func (s *Service) ListProducts(org string) []Product     { return s.store.ListProducts(org) }
func (s *Service) ListSKUs(product string) []SKU         { return s.store.ListSKUs(product) }
func (s *Service) ListWarehouses(org string) []Warehouse { return s.store.ListWarehouses(org) }
