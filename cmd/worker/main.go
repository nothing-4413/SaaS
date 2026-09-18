package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nothing-4413/saas/internal/config"
	"github.com/nothing-4413/saas/internal/notification"
	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/postgres"
	"github.com/nothing-4413/saas/internal/webhook"
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
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("outbox worker started interval=%s batch_size=%d", interval, batchSize)
	for {
		select {
		case <-ticker.C:
			processed, failed := dispatcher.DispatchOnce(batchSize, webhooks.Deliver)
			if processed > 0 || failed > 0 {
				log.Printf("outbox batch processed=%d failed=%d", processed, failed)
			}
		case <-signals:
			log.Print("outbox worker stopped")
			return
		}
	}
}
