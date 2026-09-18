package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	HealthHandler(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Fatal("empty health response")
	}
}

type testPinger struct{ err error }

func (p testPinger) PingContext(context.Context) error { return p.err }
func TestReadinessHandler(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	ok := httptest.NewRecorder()
	ReadinessHandler(testPinger{}).ServeHTTP(ok, r)
	if ok.Code != 200 {
		t.Fatalf("ready=%d", ok.Code)
	}
	bad := httptest.NewRecorder()
	ReadinessHandler(testPinger{err: errors.New("down")}).ServeHTTP(bad, r)
	if bad.Code != 503 {
		t.Fatalf("unready=%d", bad.Code)
	}
}
