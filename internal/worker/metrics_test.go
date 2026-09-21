package worker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	m := NewMetrics()
	m.AddDispatch(3, 1)
	m.AddAlertScan()
	m.AddError()
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "saas_worker_dispatch_processed_total 3") || !strings.Contains(rr.Body.String(), "saas_worker_errors_total 1") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("health status=%d body=%s", rr.Code, rr.Body.String())
	}
}
