package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nothing-4413/saas/internal/alert"
	"github.com/nothing-4413/saas/internal/config"
	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/notification"
	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/postgres"
	"github.com/nothing-4413/saas/internal/webhook"
	workerMetrics "github.com/nothing-4413/saas/internal/worker"
)

func main() {
	cfg := config.Load()
	db, err := postgres.Open(context.Background(), cfg.PostgresURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	events := outbox.NewService(outbox.NewPostgresStore(db))
	dispatcher := notification.NewService(events)
	webhooks := webhook.NewSubscriptionService(webhook.NewPostgresStore(db), webhook.Sender{})
	alertRules := alert.NewRuleService(alert.NewPostgresStore(db))
	alertScanner := alert.NewService(inventory.NewService(inventory.NewPostgresStore(db)), events)
	metrics := workerMetrics.NewMetrics()
	healthAddr := os.Getenv("WORKER_HEALTH_ADDR")
	if healthAddr == "" {
		healthAddr = ":9090"
	}
	healthServer := &http.Server{Addr: healthAddr, Handler: metrics.Handler(), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("worker health server failed: %v", err)
			metrics.AddError()
		}
	}()
	intervalMS := config.IntEnv("OUTBOX_POLL_INTERVAL_MS", 1000)
	if intervalMS <= 0 {
		intervalMS = 1000
	}
	interval := time.Duration(intervalMS) * time.Millisecond
	batchSize := config.IntEnv("OUTBOX_BATCH_SIZE", 100)
	if batchSize <= 0 {
		batchSize = 100
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	alertIntervalSeconds := config.IntEnv("STOCK_ALERT_INTERVAL_SECONDS", 60)
	if alertIntervalSeconds <= 0 {
		alertIntervalSeconds = 60
	}
	alertTicker := time.NewTicker(time.Duration(alertIntervalSeconds) * time.Second)
	defer alertTicker.Stop()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("outbox worker started interval=%s batch_size=%d", interval, batchSize)
	for {
		select {
		case <-ticker.C:
			processed, failed := dispatcher.DispatchOnce(batchSize, webhooks.Deliver)
			metrics.AddDispatch(processed, failed)
			if processed > 0 || failed > 0 {
				log.Printf("outbox batch processed=%d failed=%d", processed, failed)
			}
		case <-alertTicker.C:
			metrics.AddAlertScan()
			for _, rule := range alertRules.ListEnabled() {
				if _, err := alertScanner.Scan(rule.OrganizationID, rule.Threshold); err != nil {
					log.Printf("stock alert scan failed organization_id=%s error=%v", rule.OrganizationID, err)
					metrics.AddError()
				}
			}
		case <-signals:
			log.Print("outbox worker stopped")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = healthServer.Shutdown(shutdownCtx)
			cancel()
			return
		}
	}
}
