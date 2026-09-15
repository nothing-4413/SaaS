package product

import "time"

type Product struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type SKU struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ProductID      string `json:"product_id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	PriceCents     int64  `json:"price_cents"`
}

type Warehouse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Address        string    `json:"address,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateProductInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type CreateSKUInput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
}
type CreateWarehouseInput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}
