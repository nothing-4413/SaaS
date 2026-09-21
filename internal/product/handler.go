package product

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/nothing-4413/saas/internal/platform/pagination"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) >= 3 && p[0] == "organizations" {
		org := p[1]
		if p[2] == "products" {
			if len(p) == 3 && r.Method == http.MethodPost {
				h.createProduct(w, r, org)
				return
			}
			if len(p) == 3 && r.Method == http.MethodGet {
				h.listProducts(w, r, org)
				return
			}
			if len(p) == 5 && p[4] == "skus" && r.Method == http.MethodPost {
				h.createSKU(w, r, org, p[3])
				return
			}
			if len(p) == 5 && p[4] == "skus" && r.Method == http.MethodGet {
				h.listSKUs(w, r, p[3])
				return
			}
		}
		if p[2] == "warehouses" {
			if len(p) == 3 && r.Method == http.MethodPost {
				h.createWarehouse(w, r, org)
				return
			}
			if len(p) == 3 && r.Method == http.MethodGet {
				h.listWarehouses(w, r, org)
				return
			}
		}
	}
	http.NotFound(w, r)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request, org string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListProducts(org)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.Name), q) || strings.Contains(strings.ToLower(v.Description), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		if p.Sort == "name" {
			left, right := strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
			if left == right {
				left, right = items[i].ID, items[j].ID
			}
			if left == right {
				return false
			}
			if p.Desc {
				return left > right
			}
			return left < right
		}
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			if items[i].ID == items[j].ID {
				return false
			}
			if p.Desc {
				return items[i].ID > items[j].ID
			}
			return items[i].ID < items[j].ID
		}
		if p.Desc {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}

func (h *Handler) listSKUs(w http.ResponseWriter, r *http.Request, productID string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListSKUs(productID)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.Code), q) || strings.Contains(strings.ToLower(v.Name), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := strings.ToLower(items[i].Code), strings.ToLower(items[j].Code)
		if p.Sort == "name" {
			left, right = strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
		}
		if left == right {
			left, right = items[i].ID, items[j].ID
		}
		if left == right {
			return false
		}
		if p.Desc {
			return left > right
		}
		return left < right
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}

func (h *Handler) listWarehouses(w http.ResponseWriter, r *http.Request, org string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListWarehouses(org)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.Name), q) || strings.Contains(strings.ToLower(v.Address), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
		if p.Sort == "created_at" {
			if items[i].CreatedAt.Equal(items[j].CreatedAt) {
				left, right = items[i].ID, items[j].ID
			} else {
				if p.Desc {
					return items[i].CreatedAt.After(items[j].CreatedAt)
				}
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
		}
		if left == right {
			left, right = items[i].ID, items[j].ID
		}
		if left == right {
			return false
		}
		if p.Desc {
			return left > right
		}
		return left < right
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}
func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if e := json.NewDecoder(r.Body).Decode(v); e != nil {
		writeError(w, 400, "invalid JSON")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, m string) {
	writeJSON(w, status, map[string]string{"error": m})
}
func stat(e error) int {
	if errors.Is(e, ErrNotFound) {
		return 404
	}
	if errors.Is(e, ErrConflict) {
		return 409
	}
	return 400
}
func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request, o string) {
	var in CreateProductInput
	if !decode(w, r, &in) {
		return
	}
	v, e := h.service.CreateProduct(o, in)
	if e != nil {
		writeError(w, stat(e), e.Error())
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) createSKU(w http.ResponseWriter, r *http.Request, o, p string) {
	var in CreateSKUInput
	if !decode(w, r, &in) {
		return
	}
	v, e := h.service.CreateSKU(o, p, in)
	if e != nil {
		writeError(w, stat(e), e.Error())
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) createWarehouse(w http.ResponseWriter, r *http.Request, o string) {
	var in CreateWarehouseInput
	if !decode(w, r, &in) {
		return
	}
	v, e := h.service.CreateWarehouse(o, in)
	if e != nil {
		writeError(w, stat(e), e.Error())
		return
	}
	writeJSON(w, 201, v)
}
