package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/product"
)

type apiHandler struct{ auth, product, inventory http.Handler }

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if strings.Contains(path, "/stock") || strings.HasSuffix(path, "/stocks") {
		h.inventory.ServeHTTP(w, r)
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
	inventoryHandler := inventory.NewHandler(inventory.NewService(inventory.NewMemoryStore()))
	handler := apiHandler{auth: auth.NewHandler(service), product: productHandler, inventory: inventoryHandler}

	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
