package webhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nothing-4413/saas/internal/outbox"
)

func TestSubscriptionCRUDAndDelivery(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if !strings.HasPrefix(r.Header.Get("X-Webhook-Signature"), "sha256=") {
			t.Fatal("missing signature")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	service := NewSubscriptionService(NewMemoryStore(), Sender{AllowPrivate: true})
	subscription, err := service.Create("org", CreateSubscriptionInput{URL: server.URL, Secret: "0123456789abcdef", EventTypes: []string{"order.confirmed"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(service.List("org")) != 1 {
		t.Fatal("subscription not listed")
	}
	err = service.Deliver(outbox.Event{ID: "event", OrganizationID: "org", Type: "order.confirmed", Payload: []byte(`{"ok":true}`)})
	if err != nil || requests != 1 {
		t.Fatalf("requests=%d err=%v", requests, err)
	}
	if err := service.Delete("org", subscription.ID); err != nil || len(service.List("org")) != 0 {
		t.Fatalf("delete err=%v", err)
	}
}

func TestSubscriptionFiltersEvents(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	service := NewSubscriptionService(NewMemoryStore(), Sender{AllowPrivate: true})
	_, _ = service.Create("org", CreateSubscriptionInput{URL: server.URL, Secret: "0123456789abcdef", EventTypes: []string{"stock.low"}})
	if err := service.Deliver(outbox.Event{ID: "event", OrganizationID: "org", Type: "order.created", Payload: []byte(`{}`)}); err != nil || requests != 0 {
		t.Fatalf("requests=%d err=%v", requests, err)
	}
}

func TestSubscriptionDeliveryIsolatesFailuresAndSkipsSuccessesOnRetry(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Idempotency-Key")
		requests[key]++
		if strings.HasSuffix(r.URL.Path, "/bad") && requests[key] == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	store := NewMemoryStore()
	service := NewSubscriptionService(store, Sender{AllowPrivate: true})
	good, _ := service.Create("org", CreateSubscriptionInput{URL: server.URL + "/good", Secret: "0123456789abcdef", EventTypes: []string{"order.created"}})
	bad, _ := service.Create("org", CreateSubscriptionInput{URL: server.URL + "/bad", Secret: "0123456789abcdef", EventTypes: []string{"order.created"}})
	event := outbox.Event{ID: "event", OrganizationID: "org", Type: "order.created", Payload: []byte(`{}`)}
	if err := service.Deliver(event); err == nil {
		t.Fatal("expected failed subscription")
	}
	if err := service.Deliver(event); err != nil {
		t.Fatal(err)
	}
	if requests["event:"+good.ID] != 1 || requests["event:"+bad.ID] != 2 {
		t.Fatalf("requests=%v", requests)
	}
}

func TestPasswordResetEventsRequireExplicitSubscription(t *testing.T) {
	if accepts([]string{"*"}, "auth.password_reset_requested") {
		t.Fatal("wildcard subscription must not receive password reset tokens")
	}
	if !accepts([]string{"auth.password_reset_requested"}, "auth.password_reset_requested") {
		t.Fatal("explicit password reset subscription should receive event")
	}
}
