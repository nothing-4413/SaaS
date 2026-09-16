package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/config"
	"github.com/nothing-4413/saas/internal/httpx"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"github.com/nothing-4413/saas/internal/product"
	"github.com/nothing-4413/saas/internal/report"
)

type apiHandler struct{ auth, product, inventory, order, report http.Handler }

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "healthz" && r.Method == http.MethodGet {
		httpx.HealthHandler(w, r)
		return
	}
	if strings.Contains(path, "/stock") || strings.HasSuffix(path, "/stocks") || strings.Contains(path, "/receipts") || strings.Contains(path, "/issues") {
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
	cfg := config.Load()
	store := auth.NewMemoryStore()
	service := auth.NewService(store)
	productHandler := product.NewHandler(product.NewService(product.NewMemoryStore()))
	inventoryService := inventory.NewService(inventory.NewMemoryStore())
	inventoryHandler := inventory.NewHandler(inventoryService)
	orderService := order.NewService(order.NewMemoryStore(), inventoryService)
	orderHandler := order.NewHandler(orderService)
	reportHandler := report.NewHandler(report.NewService(orderService, inventoryService))
	handler := apiHandler{auth: auth.NewHandler(service), product: productHandler, inventory: inventoryHandler, order: orderHandler, report: reportHandler}

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Println("api listening on", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
