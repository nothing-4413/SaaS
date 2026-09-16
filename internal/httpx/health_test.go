package httpx

import (
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
