package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"github.com/nothing-4413/saas/internal/product"
	"github.com/nothing-4413/saas/internal/report"
)

func testHandler() http.Handler {
	authService := auth.NewService(auth.NewMemoryStore())
	inv := inventory.NewService(inventory.NewMemoryStore())
	orders := order.NewService(order.NewMemoryStore(), inv)
	authHandler := auth.NewHandler(authService)
	return apiHandler{authPublic: authHandler, userRead: authHandler, userWrite: authHandler, roleManage: authHandler, product: product.NewHandler(product.NewService(product.NewMemoryStore())), inventory: inventory.NewHandler(inv), order: order.NewHandler(orders), report: report.NewHandler(report.NewService(orders, inv)), readiness: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}
}
func TestHealthAndOrganizationRoutes(t *testing.T) {
	h := testHandler()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("health=%d", w.Code)
	}
	r = httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(`{"name":"Acme","owner_email":"owner@example.com","owner_name":"Owner","owner_password":"password123"}`))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("organization=%d body=%s", w.Code, w.Body.String())
	}
}
