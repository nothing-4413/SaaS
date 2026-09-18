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

	"github.com/nothing-4413/saas/internal/audit"
	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/config"
	"github.com/nothing-4413/saas/internal/export"
	"github.com/nothing-4413/saas/internal/httpx"
	"github.com/nothing-4413/saas/internal/importer"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/postgres"
	"github.com/nothing-4413/saas/internal/product"
	"github.com/nothing-4413/saas/internal/report"
)

type apiHandler struct {
	authPublic, userRead, userWrite, roleManage                http.Handler
	product, inventory, order, report, export, importer, audit http.Handler
	metrics, readiness                                         http.Handler
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "healthz" && r.Method == http.MethodGet {
		httpx.HealthHandler(w, r)
		return
	}
	if path == "readyz" && r.Method == http.MethodGet {
		h.readiness.ServeHTTP(w, r)
		return
	}
	if path == "metrics" && r.Method == http.MethodGet {
		h.metrics.ServeHTTP(w, r)
		return
	}
	parts := strings.Split(path, "/")
	if path == "organizations" || (len(parts) == 3 && parts[0] == "organizations" && parts[2] == "sessions") {
		h.authPublic.ServeHTTP(w, r)
		return
	}
	if len(parts) == 3 && parts[0] == "organizations" && parts[2] == "users" {
		if r.Method == http.MethodGet {
			h.userRead.ServeHTTP(w, r)
		} else {
			h.userWrite.ServeHTTP(w, r)
		}
		return
	}
	if len(parts) == 3 && parts[0] == "organizations" && parts[2] == "roles" {
		h.roleManage.ServeHTTP(w, r)
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
	if strings.Contains(path, "/exports/") {
		h.export.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/imports/") {
		h.importer.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/audit-logs") {
		h.audit.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/products") || strings.Contains(path, "/warehouses") {
		h.product.ServeHTTP(w, r)
		return
	}
	h.authPublic.ServeHTTP(w, r)
}

func main() {
	cfg := config.Load()
	db, err := postgres.Open(context.Background(), cfg.PostgresURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	store := auth.NewPostgresStore(db)
	service := auth.NewService(store)
	productHandler := product.NewHandler(product.NewService(product.NewPostgresStore(db)))
	inventoryService := inventory.NewService(inventory.NewPostgresStore(db))
	inventoryHandler := inventory.NewHandler(inventoryService)
	orderService := order.NewService(order.NewPostgresStore(db), inventoryService)
	orderService.SetEventService(outbox.NewService(outbox.NewPostgresStore(db)))
	orderHandler := order.NewHandler(orderService)
	reportHandler := report.NewHandler(report.NewService(orderService, inventoryService))
	exportHandler := export.NewHandler(orderService, inventoryService)
	importerHandler := importer.NewHandler(inventoryService)
	metrics := httpx.NewMetrics()
	authHandler := auth.NewHandler(service, cfg.AuthTokenSecret)
	auditService := audit.NewService(audit.NewPostgresStore(db))
	protect := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.RequireTokenPermission(service, cfg.AuthTokenSecret, permission, audit.Middleware(auditService, handler))
	}
	base := apiHandler{
		authPublic: authHandler,
		userRead:   protect(auth.PermissionUserRead, authHandler),
		userWrite:  protect(auth.PermissionUserWrite, authHandler),
		roleManage: protect(auth.PermissionRoleManage, authHandler),
		product:    protect(auth.PermissionProductManage, productHandler),
		inventory:  protect(auth.PermissionInventoryManage, inventoryHandler),
		order:      protect(auth.PermissionOrderManage, orderHandler),
		report:     protect(auth.PermissionReportRead, reportHandler),
		export:     protect(auth.PermissionReportRead, exportHandler),
		importer:   protect(auth.PermissionInventoryManage, importerHandler),
		audit:      protect(auth.PermissionAuditRead, audit.NewHandler(auditService)),
		metrics:    metrics,
		readiness:  httpx.ReadinessHandler(db),
	}
	limiter := httpx.NewRateLimiter(120, time.Minute)
	handler := httpx.Chain(httpx.SecurityHeaders(httpx.MaxBodyBytes(2<<20, limiter.Middleware(metrics.Wrap(base)))))

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
