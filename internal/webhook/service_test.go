package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendSignsAndAddsIdempotencyHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Idempotency-Key") != "k1" {
			t.Fatal("missing idempotency")
		}
		if r.Header.Get("X-Webhook-Signature") == "" {
			t.Fatal("missing signature")
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()
	e := Sender{}.Send(Delivery{ID: "d1", URL: srv.URL, Secret: "secret", EventType: "order.created", Payload: []byte(`{"ok":true}`), IdempotencyKey: "k1"})
	if e != nil {
		t.Fatal(e)
	}
}
func TestSendRejectsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	if e := (Sender{}).Send(Delivery{ID: "d", URL: srv.URL, Secret: "s", EventType: "x", Payload: []byte("{}"), IdempotencyKey: "k"}); e == nil {
		t.Fatal("expected error")
	}
}
