package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing security header")
	}
}
func TestRateLimiter(t *testing.T) {
	l := NewRateLimiter(1, time.Minute)
	if !l.Allow("x") || l.Allow("x") {
		t.Fatal("unexpected limiter result")
	}
}

func TestRateLimiterUsesSafeDefaults(t *testing.T) {
	l := NewRateLimiter(0, 0)
	if !l.Allow("x") || l.Allow("x") {
		t.Fatal("invalid limits should normalize to one request per window")
	}
}

func TestRateLimiterCleansExpiredClients(t *testing.T) {
	l := NewRateLimiter(1, time.Millisecond)
	l.maxKeys = 1
	if !l.Allow("expired") {
		t.Fatal("first request rejected")
	}
	time.Sleep(2 * time.Millisecond)
	if !l.Allow("fresh") {
		t.Fatal("fresh client rejected")
	}
	if _, ok := l.seen["expired"]; ok {
		t.Fatal("expired client was not cleaned")
	}
}
