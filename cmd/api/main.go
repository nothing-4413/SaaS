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

	"github.com/nothing-4413/saas/internal/alert"
	"github.com/nothing-4413/saas/internal/audit"
	"github.com/nothing-4413/saas/internal/auth"
	"github.com/nothing-4413/saas/internal/config"
	"github.com/nothing-4413/saas/internal/export"
	"github.com/nothing-4413/saas/internal/httpx"
	"github.com/nothing-4413/saas/internal/importer"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/idgen"
	"github.com/nothing-4413/saas/internal/platform/postgres"
	"github.com/nothing-4413/saas/internal/product"
	"github.com/nothing-4413/saas/internal/report"
	"github.com/nothing-4413/saas/internal/webhook"
)

type apiHandler struct {
	authPublic, userRead, userWrite, roleRead, roleManage                      http.Handler
	product, inventory, order, report, export, importer, audit, webhook, alert http.Handler
	metrics, readiness                                                         http.Handler
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
	if path == "organizations" || (len(parts) == 3 && parts[0] == "organizations" && parts[2] == "sessions" && r.Method == http.MethodPost) {
		h.authPublic.ServeHTTP(w, r)
		return
	}
	if len(parts) == 3 && parts[0] == "organizations" && parts[2] == "sessions" {
		h.authPublic.ServeHTTP(w, r)
		return
	}
	if len(parts) == 2 && parts[0] == "organizations" {
		if r.Method == http.MethodGet {
			h.userRead.ServeHTTP(w, r)
		} else {
			h.userWrite.ServeHTTP(w, r)
		}
		return
	}
	if len(parts) >= 3 && parts[0] == "organizations" && parts[2] == "users" {
		if r.Method == http.MethodGet {
			h.userRead.ServeHTTP(w, r)
		} else {
			h.userWrite.ServeHTTP(w, r)
		}
		return
	}
	if len(parts) >= 3 && parts[0] == "organizations" && parts[2] == "roles" {
		if r.Method == http.MethodGet {
			h.roleRead.ServeHTTP(w, r)
			return
		}
		h.roleManage.ServeHTTP(w, r)
		return
	}
	if len(parts) == 4 && parts[0] == "organizations" && parts[2] == "alerts" && parts[3] == "stock" {
		h.alert.ServeHTTP(w, r)
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
	if strings.Contains(path, "/audit-logs") {
		h.audit.ServeHTTP(w, r)
		return
	}
	if strings.Contains(path, "/webhooks") {
		h.webhook.ServeHTTP(w, r)
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
	if err := cfg.ValidateAPI(); err != nil {
		log.Fatal(err)
	}
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
	events := outbox.NewService(outbox.NewPostgresStore(db))
	orderService := order.NewService(order.NewPostgresStore(db), inventoryService)
	orderService.SetCatalog(product.NewPostgresStore(db))
	orderService.SetEventService(events)
	orderHandler := order.NewHandler(orderService)
	reportHandler := report.NewHandler(report.NewService(orderService, inventoryService))
	exportHandler := export.NewHandler(orderService, inventoryService)
	importerHandler := importer.NewHandler(inventoryService)
	metrics := httpx.NewMetrics()
	authSecrets := append([]string{cfg.AuthTokenSecret}, cfg.AuthTokenPreviousSecrets...)
	authHandler := auth.NewHandler(service, authSecrets...)
	authHandler.SetPasswordResetNotifier(func(org, email, token string) error {
		_, err := events.Enqueue(org, "password_reset", email, "auth.password_reset_requested", idgen.New(), map[string]string{"email": email, "token": token})
		return err
	})
	auditService := audit.NewService(audit.NewPostgresStore(db))
	webhookService := webhook.NewSubscriptionService(webhook.NewPostgresStore(db), webhook.Sender{})
	alertHandler := alert.NewHandler(alert.NewRuleService(alert.NewPostgresStore(db)))
	protect := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.RequireTokenPermissions(service, authSecrets, permission, audit.Middleware(auditService, handler))
	}
	protectResolved := func(resolve func(*http.Request) auth.Permission, handler http.Handler) http.Handler {
		return auth.RequireTokenPermissionResolver(service, authSecrets, resolve, audit.Middleware(auditService, handler))
	}
	methodPermission := func(read, write auth.Permission) func(*http.Request) auth.Permission {
		return func(r *http.Request) auth.Permission {
			if r.Method == http.MethodGet {
				return read
			}
			return write
		}
	}
	base := apiHandler{
		authPublic: authHandler,
		userRead:   protect(auth.PermissionUserRead, authHandler),
		userWrite:  protect(auth.PermissionUserWrite, authHandler),
		roleRead:   protect(auth.PermissionRoleRead, authHandler),
		roleManage: protect(auth.PermissionRoleManage, authHandler),
		product:    protectResolved(methodPermission(auth.PermissionProductRead, auth.PermissionProductWrite), productHandler),
		inventory:  protectResolved(methodPermission(auth.PermissionInventoryRead, auth.PermissionInventoryWrite), inventoryHandler),
		order: protectResolved(func(r *http.Request) auth.Permission {
			if r.Method == http.MethodGet {
				return auth.PermissionOrderRead
			}
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) >= 5 {
				return auth.PermissionOrderApprove
			}
			return auth.PermissionOrderWrite
		}, orderHandler),
		report:    protect(auth.PermissionReportRead, reportHandler),
		export:    protect(auth.PermissionReportExport, exportHandler),
		importer:  protect(auth.PermissionInventoryImport, importerHandler),
		audit:     protect(auth.PermissionAuditRead, audit.NewHandler(auditService)),
		webhook:   protect(auth.PermissionWebhookManage, webhook.NewHandler(webhookService)),
		alert:     protectResolved(methodPermission(auth.PermissionInventoryRead, auth.PermissionInventoryWrite), alertHandler),
		metrics:   metrics,
		readiness: httpx.ReadinessHandler(db),
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
