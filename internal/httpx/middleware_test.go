package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDAndRecovery(t *testing.T) {
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("X-Request-ID") == "" {
			t.Fatal("request id missing")
		}
		panic("boom")
	}))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAccessLogDefaultsStatusToOK(t *testing.T) {
	h := AccessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestResponseWriterKeepsFirstStatus(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}
	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusInternalServerError)
	if w.Code != http.StatusCreated || rw.status != http.StatusCreated {
		t.Fatalf("status=%d recorded=%d", w.Code, rw.status)
	}
}

func TestRequestIDRejectsControlCharacters(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Request-ID", "bad\nvalue")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("X-Request-ID") == "bad\nvalue" || w.Header().Get("X-Request-ID") == "" {
		t.Fatal("invalid request id accepted")
	}
}
