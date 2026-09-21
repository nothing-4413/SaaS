package product

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProductsPaginationAndQuery(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)
	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		if _, err := service.CreateProduct("org", CreateProductInput{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/organizations/org/products?page=2&page_size=1&q=a&sort=name", nil)
	res := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body struct {
		Items    []Product `json:"items"`
		Page     int       `json:"page"`
		PageSize int       `json:"page_size"`
		Total    int       `json:"total"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Page != 2 || body.PageSize != 1 || body.Total != 3 || len(body.Items) != 1 || body.Items[0].Name != "Beta" {
		t.Fatalf("body=%+v", body)
	}
}

func TestListProductsRejectsInvalidPageSize(t *testing.T) {
	h := NewHandler(NewService(NewMemoryStore()))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/organizations/org/products?page_size=101", strings.NewReader("")))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", res.Code)
	}
}
