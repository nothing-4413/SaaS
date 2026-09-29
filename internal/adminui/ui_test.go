package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesConsoleAndRejectsUnknownAssets(t *testing.T) {
	h := Handler()
	for _, path := range []string{"/", "/app.js", "/styles.css", "/reset-password"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) == "" {
			t.Fatalf("path=%s status=%d body=%q", path, rr.Code, rr.Body.String())
		}
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown asset status=%d", rr.Code)
	}
}
