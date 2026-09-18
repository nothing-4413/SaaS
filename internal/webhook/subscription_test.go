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
	service := NewSubscriptionService(NewMemoryStore(), Sender{})
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
	service := NewSubscriptionService(NewMemoryStore(), Sender{})
	_, _ = service.Create("org", CreateSubscriptionInput{URL: server.URL, Secret: "0123456789abcdef", EventTypes: []string{"stock.low"}})
	if err := service.Deliver(outbox.Event{ID: "event", OrganizationID: "org", Type: "order.created", Payload: []byte(`{}`)}); err != nil || requests != 0 {
		t.Fatalf("requests=%d err=%v", requests, err)
	}
}
