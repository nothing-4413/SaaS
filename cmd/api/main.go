package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"github.com/nothing-4413/saas/internal/product"
	"github.com/nothing-4413/saas/internal/report"
)

type apiHandler struct{ auth, product, inventory, order, report http.Handler }

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if strings.Contains(path, "/stock") || strings.HasSuffix(path, "/stocks") {
		h.inventory.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/orders") {
		h.order.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/reports/") {
		h.report.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/products") || strings.Contains(path, "/warehouses") {
		h.product.ServeHTTP(w, r)
		return
	}
	h.auth.ServeHTTP(w, r)
}

func main() {
	store := auth.NewMemoryStore()
	service := auth.NewService(store)
	productHandler := product.NewHandler(product.NewService(product.NewMemoryStore()))
	inventoryService := inventory.NewService(inventory.NewMemoryStore())
	inventoryHandler := inventory.NewHandler(inventoryService)
	orderService := order.NewService(order.NewMemoryStore(), inventoryService)
	orderHandler := order.NewHandler(orderService)
	reportHandler := report.NewHandler(report.NewService(orderService, inventoryService))
	handler := apiHandler{auth: auth.NewHandler(service), product: productHandler, inventory: inventoryHandler, order: orderHandler, report: reportHandler}

	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
