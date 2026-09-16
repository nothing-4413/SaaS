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
