//go:build integration

// This file is the only place where the Postgres persistence layer is actually
// executed: the default `go test ./...` run ignores it because of the build tag
// and because TEST_DATABASE_URL is normally unset.
//
// Run it against a throwaway database:
//
//	docker compose up -d postgres
//	docker compose run --rm migrate
//	TEST_DATABASE_URL='postgres://saas:saas@localhost:5432/saas?sslmode=disable' \
//	  go test -tags integration ./internal/inventory/...
package inventory

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/postgres"
)

// tenant is one isolated set of rows: every test builds its own organization,
// product, sku and warehouse with fresh UUIDs, so concurrent runs and repeated
// runs never share state. organizations is the root of the ON DELETE CASCADE
// chain, so deleting it removes everything the test created.
type tenant struct {
	db        *sql.DB
	org       string
	product   string
	warehouse string
	sku       string
}

func newTenant(t *testing.T) tenant {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the Postgres integration tests")
	}
	db, err := postgres.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	fx := tenant{
		db:        db,
		org:       integrationUUID(t),
		product:   integrationUUID(t),
		warehouse: integrationUUID(t),
		sku:       integrationUUID(t),
	}
	// Column names and constraints below come from migrations/*.up.sql:
	// organizations(id,name) 000001:2, products(id,organization_id,name)
	// 000001:37, skus(id,organization_id,product_id,code,name,price_cents)
	// 000001:46, warehouses(id,organization_id,name) 000001:58.
	seeds := []struct {
		query string
		args  []interface{}
	}{
		{`INSERT INTO organizations (id,name) VALUES ($1,$2)`, []interface{}{fx.org, "integration-test"}},
		{`INSERT INTO products (id,organization_id,name) VALUES ($1,$2,$3)`, []interface{}{fx.product, fx.org, "integration-test-product"}},
		{`INSERT INTO skus (id,organization_id,product_id,code,name,price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, []interface{}{fx.sku, fx.org, fx.product, "TEST-SKU", "Test SKU", 100}},
		{`INSERT INTO warehouses (id,organization_id,name) VALUES ($1,$2,$3)`, []interface{}{fx.warehouse, fx.org, "integration-test-warehouse"}},
	}
	for _, seed := range seeds {
		if _, err := db.ExecContext(context.Background(), seed.query, seed.args...); err != nil {
			t.Fatalf("seed %s: %v", seed.query, err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM organizations WHERE id=$1`, fx.org)
	})
	return fx
}

func (fx tenant) store() *PostgresStore { return NewPostgresStore(fx.db) }

// integrationUUID formats 16 random bytes as a v4 UUID: every identifier in the
// schema is a uuid column and the project has no uuid dependency.
func integrationUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// concurrentApply runs the same stock operation from many goroutines at once
// and returns how many callers were accepted, how many were rejected as
// insufficient, and any error that was neither.
func concurrentApply(store *PostgresStore, org, warehouse, sku, action string, workers, quantity int, prefix string) (accepted, rejected int, unexpected []error) {
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = store.Apply(org, warehouse, sku, action, int64(quantity), fmt.Sprintf("%s-%d", prefix, i), time.Now().UTC())
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, ErrInsufficient):
			rejected++
		default:
			unexpected = append(unexpected, err)
		}
	}
	return accepted, rejected, unexpected
}

// TestConcurrentDeductNeverOversells is the invariant the in-memory store gets
// for free from its mutex but which really lives in SQL: the SELECT ... FOR
// UPDATE plus the availability re-check must not let concurrent deducts take
// the same unit twice.
func TestConcurrentDeductNeverOversells(t *testing.T) {
	fx := newTenant(t)
	store := fx.store()
	const onHand, workers, quantity = 30, 20, 3
	if _, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", onHand, "seed-receive", time.Now().UTC()); err != nil {
		t.Fatalf("seed receive: %v", err)
	}

	accepted, rejected, unexpected := concurrentApply(store, fx.org, fx.warehouse, fx.sku, "deduct", workers, quantity, "deduct")
	for _, err := range unexpected {
		t.Errorf("deduct: unexpected error %v", err)
	}
	if want := onHand / quantity; accepted != want {
		t.Fatalf("deducts accepted=%d want=%d: stock was oversold or an update was lost", accepted, want)
	}
	if rejected != workers-accepted {
		t.Fatalf("deducts rejected=%d want=%d", rejected, workers-accepted)
	}
	final, err := store.Get(fx.org, fx.warehouse, fx.sku)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if final.OnHand != 0 || final.Reserved != 0 || final.Available != 0 {
		t.Fatalf("on_hand=%d reserved=%d available=%d want 0/0/0", final.OnHand, final.Reserved, final.Available)
	}
}

// TestConcurrentReserveNeverOversells covers the reserve path, which the order
// flow uses before confirmation and which must leave available >= 0 as well.
func TestConcurrentReserveNeverOversells(t *testing.T) {
	fx := newTenant(t)
	store := fx.store()
	const onHand, workers, quantity = 30, 20, 3
	if _, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", onHand, "seed-receive", time.Now().UTC()); err != nil {
		t.Fatalf("seed receive: %v", err)
	}

	accepted, rejected, unexpected := concurrentApply(store, fx.org, fx.warehouse, fx.sku, "reserve", workers, quantity, "reserve")
	for _, err := range unexpected {
		t.Errorf("reserve: unexpected error %v", err)
	}
	if want := onHand / quantity; accepted != want {
		t.Fatalf("reserves accepted=%d want=%d: stock was oversold or an update was lost", accepted, want)
	}
	if rejected != workers-accepted {
		t.Fatalf("reserves rejected=%d want=%d", rejected, workers-accepted)
	}
	final, err := store.Get(fx.org, fx.warehouse, fx.sku)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if final.OnHand != onHand || final.Reserved != onHand || final.Available != 0 {
		t.Fatalf("on_hand=%d reserved=%d available=%d want %d/%d/0", final.OnHand, final.Reserved, final.Available, onHand, onHand)
	}
}

// TestApplyReplayIsIdempotentAndMismatchConflicts covers both halves of the
// idempotency contract of inventory_operations (UNIQUE (organization_id,
// idempotency_key), migrations/000003_inventory_operations.up.sql:10): a
// repeated key must return the stored result instead of booking again, and the
// same key with a different payload must be refused.
func TestApplyReplayIsIdempotentAndMismatchConflicts(t *testing.T) {
	fx := newTenant(t)
	store := fx.store()
	at := time.Now().UTC()
	first, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", 10, "receive-1", at)
	if err != nil {
		t.Fatalf("first receive: %v", err)
	}
	if first.OnHand != 10 {
		t.Fatalf("on_hand=%d want=10", first.OnHand)
	}

	replay, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", 10, "receive-1", at.Add(time.Second))
	if err != nil {
		t.Fatalf("replayed receive: %v", err)
	}
	if replay.OnHand != 10 {
		t.Fatalf("replayed receive booked twice: on_hand=%d want=10", replay.OnHand)
	}

	if _, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", 999, "receive-1", at); !errors.Is(err, ErrConflict) {
		t.Fatalf("same key with a different quantity: err=%v want ErrConflict", err)
	}
	if _, err := store.Apply(fx.org, fx.warehouse, fx.sku, "deduct", 10, "receive-1", at); !errors.Is(err, ErrConflict) {
		t.Fatalf("same key with a different action: err=%v want ErrConflict", err)
	}

	after, err := store.Get(fx.org, fx.warehouse, fx.sku)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if after.OnHand != 10 || after.Reserved != 0 {
		t.Fatalf("rejected replays changed stock: on_hand=%d reserved=%d want 10/0", after.OnHand, after.Reserved)
	}
}

// TestCreateDocumentAtomicAppliesIssueOnce pins the document path, the only
// other writer of the inventory ledger. Documents are deduplicated by
// UNIQUE (organization_id, document_type, idempotency_key)
// (migrations/000002_inventory_documents.up.sql:7).
func TestCreateDocumentAtomicAppliesIssueOnce(t *testing.T) {
	fx := newTenant(t)
	store := fx.store()
	if _, err := store.Apply(fx.org, fx.warehouse, fx.sku, "receive", 10, "seed-receive", time.Now().UTC()); err != nil {
		t.Fatalf("seed receive: %v", err)
	}

	doc := Document{
		ID:             integrationUUID(t),
		OrganizationID: fx.org,
		Type:           DocumentIssue,
		IdempotencyKey: "issue-1",
		CreatedAt:      time.Now().UTC(),
		Lines:          []DocumentLine{{WarehouseID: fx.warehouse, SKUID: fx.sku, Quantity: 4}},
	}
	created, err := store.CreateDocumentAtomic(doc)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.ID != doc.ID {
		t.Fatalf("created id=%s want=%s", created.ID, doc.ID)
	}

	// A retry carries a fresh document ID but the same idempotency key: it must
	// return the original document and must not deduct a second time.
	retry := doc
	retry.ID = integrationUUID(t)
	again, err := store.CreateDocumentAtomic(retry)
	if err != nil {
		t.Fatalf("replayed issue: %v", err)
	}
	if again.ID != created.ID {
		t.Fatalf("replay created a second document: %s want %s", again.ID, created.ID)
	}
	stock, err := store.Get(fx.org, fx.warehouse, fx.sku)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock.OnHand != 6 {
		t.Fatalf("on_hand=%d want=6: the issue was applied twice", stock.OnHand)
	}

	oversized := Document{
		ID:             integrationUUID(t),
		OrganizationID: fx.org,
		Type:           DocumentIssue,
		IdempotencyKey: "issue-2",
		CreatedAt:      time.Now().UTC(),
		Lines:          []DocumentLine{{WarehouseID: fx.warehouse, SKUID: fx.sku, Quantity: 100}},
	}
	if _, err := store.CreateDocumentAtomic(oversized); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("oversized issue: err=%v want ErrInsufficient", err)
	}
	stock, err = store.Get(fx.org, fx.warehouse, fx.sku)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock.OnHand != 6 {
		t.Fatalf("rejected issue changed stock: on_hand=%d want=6", stock.OnHand)
	}
}

// outboxEvent builds a claimable event. The dedup key is unique per event so
// that UNIQUE (organization_id, aggregate_type, aggregate_id, event_type,
// dedup_key) (migrations/000001_initial_schema.up.sql:127) only fires when a
// test wants it to.
func outboxEvent(t *testing.T, org string, now time.Time) outbox.Event {
	t.Helper()
	id := integrationUUID(t)
	return outbox.Event{
		ID:             id,
		OrganizationID: org,
		AggregateType:  "order",
		AggregateID:    id,
		Type:           "order.created",
		DedupKey:       id,
		Payload:        []byte(`{"id":"` + id + `"}`),
		Status:         outbox.StatusPending,
		NextAttemptAt:  now.Add(-time.Minute),
		CreatedAt:      now.Add(-time.Minute),
	}
}

// claimOwn claims one batch and reports an actionable error when Claim returns
// work from another organization. Claim is deliberately global (the worker
// drains every tenant), so this suite must run against a database that has no
// other pending outbox events - CI creates a dedicated one.
func claimOwn(t *testing.T, events *outbox.PostgresStore, org string, limit int, now time.Time) []outbox.Event {
	t.Helper()
	batch := events.Claim(limit, now)
	for _, event := range batch {
		if event.OrganizationID != org {
			t.Errorf("claimed event %s belonging to organization %s: point TEST_DATABASE_URL at a dedicated database with no other pending outbox events", event.ID, event.OrganizationID)
			return nil
		}
	}
	return batch
}

// TestOutboxClaimHandsEachEventToExactlyOneConsumer exercises the worker's
// FOR UPDATE SKIP LOCKED claim query (internal/outbox/postgres_store.go:47):
// with as many concurrent consumers as there are events, no event may be handed
// to two consumers and none may be skipped.
func TestOutboxClaimHandsEachEventToExactlyOneConsumer(t *testing.T) {
	fx := newTenant(t)
	events := outbox.NewPostgresStore(fx.db)
	now := time.Now().UTC()
	const count, consumers = 8, 8
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		event := outboxEvent(t, fx.org, now)
		if err := events.Enqueue(event); err != nil {
			t.Fatalf("enqueue %s: %v", event.ID, err)
		}
		ids = append(ids, event.ID)
	}

	var wg sync.WaitGroup
	claimed := make([][]outbox.Event, consumers)
	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// SELECT ... FOR UPDATE SKIP LOCKED can hand a consumer an empty batch
			// while other consumers still hold the remaining rows, so each consumer
			// keeps asking until it owns exactly one event. The invariant under test
			// is that no event is ever handed to two consumers, not that a single
			// round drains every event.
			for attempt := 0; attempt < 100; attempt++ {
				if batch := claimOwn(t, events, fx.org, 1, now); len(batch) > 0 {
					claimed[i] = batch
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}
	wg.Wait()

	seen := map[string]int{}
	for _, batch := range claimed {
		for _, event := range batch {
			seen[event.ID]++
			if event.Status != outbox.StatusProcessing {
				t.Errorf("claimed event %s status=%s want=%s", event.ID, event.Status, outbox.StatusProcessing)
			}
			if event.Attempts != 1 {
				t.Errorf("claimed event %s attempts=%d want=1", event.ID, event.Attempts)
			}
			if !event.ClaimedUntil.After(now) {
				t.Errorf("claimed event %s lease=%s want after %s", event.ID, event.ClaimedUntil, now)
			}
		}
	}
	if len(seen) != count {
		t.Fatalf("claimed %d distinct events want=%d", len(seen), count)
	}
	for _, id := range ids {
		if seen[id] != 1 {
			t.Fatalf("event %s claimed %d times want=1", id, seen[id])
		}
	}
}

// TestOutboxDedupAndLeaseExpiry pins the two SQL behaviours the worker relies
// on: the dedup key refuses a second event, and a five-minute lease that
// expires without a result becomes claimable again.
func TestOutboxDedupAndLeaseExpiry(t *testing.T) {
	fx := newTenant(t)
	events := outbox.NewPostgresStore(fx.db)
	now := time.Now().UTC()
	event := outboxEvent(t, fx.org, now)
	if err := events.Enqueue(event); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	duplicate := event
	duplicate.ID = integrationUUID(t)
	if err := events.Enqueue(duplicate); !errors.Is(err, outbox.ErrConflict) {
		t.Fatalf("duplicate dedup key: err=%v want ErrConflict", err)
	}

	first := claimOwn(t, events, fx.org, 1, now)
	if len(first) != 1 || first[0].ID != event.ID {
		t.Fatalf("claim=%v want the pending event %s", first, event.ID)
	}
	if held := claimOwn(t, events, fx.org, 1, now); len(held) != 0 {
		t.Fatalf("second claim returned %d events: the lease did not hold", len(held))
	}
	if err := events.MarkPublished(event.ID, now); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	if err := events.MarkPublished(event.ID, now); !errors.Is(err, outbox.ErrInvalidState) {
		t.Fatalf("republish err=%v want ErrInvalidState", err)
	}

	// A consumer that died after claiming leaves the lease behind; once it
	// expires the event must be handed out again with the attempt recorded.
	stale := outboxEvent(t, fx.org, now)
	if err := events.Enqueue(stale); err != nil {
		t.Fatalf("enqueue stale: %v", err)
	}
	if claimedNow := claimOwn(t, events, fx.org, 1, now); len(claimedNow) != 1 || claimedNow[0].ID != stale.ID {
		t.Fatalf("first claim of stale=%v want %s", claimedNow, stale.ID)
	}
	later := now.Add(6 * time.Minute)
	reclaimed := claimOwn(t, events, fx.org, 1, later)
	if len(reclaimed) != 1 || reclaimed[0].ID != stale.ID {
		t.Fatalf("reclaim after lease expiry=%v want %s", reclaimed, stale.ID)
	}
	if reclaimed[0].Attempts != 2 {
		t.Fatalf("reclaimed attempts=%d want=2", reclaimed[0].Attempts)
	}

	if err := events.MarkFailed(stale.ID, later, "boom", false); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	refreshed, err := events.Get(stale.ID)
	if err != nil {
		t.Fatalf("get stale: %v", err)
	}
	if refreshed.Status != outbox.StatusPending || refreshed.LastError != "boom" {
		t.Fatalf("status=%s last_error=%q want %s/boom", refreshed.Status, refreshed.LastError, outbox.StatusPending)
	}
}
