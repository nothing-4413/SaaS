package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetrics(t *testing.T) {
	m := NewMetrics()
	h := m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("err")) }))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	out := httptest.NewRecorder()
	m.ServeHTTP(out, r)
	if !strings.Contains(out.Body.String(), "saas_http_requests_total 1") || !strings.Contains(out.Body.String(), "saas_http_errors_total 1") {
		t.Fatalf("metrics=%s", out.Body.String())
	}
}
