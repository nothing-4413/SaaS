package httpx

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	requests uint64
	errors   uint64
	bytes    uint64
}

func NewMetrics() *Metrics { return &Metrics{} }
func (m *Metrics) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		atomic.AddUint64(&m.requests, 1)
		atomic.AddUint64(&m.bytes, uint64(rw.bytes))
		if rw.status >= 500 {
			atomic.AddUint64(&m.errors, 1)
		}
	})
}
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "saas_http_requests_total %d\nsaas_http_errors_total %d\nsaas_http_response_bytes_total %d\n", atomic.LoadUint64(&m.requests), atomic.LoadUint64(&m.errors), atomic.LoadUint64(&m.bytes))
}
