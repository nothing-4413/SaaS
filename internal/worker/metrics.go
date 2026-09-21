package worker

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	processed  uint64
	failed     uint64
	alertScans uint64
	errors     uint64
}

func NewMetrics() *Metrics { return &Metrics{} }
func (m *Metrics) AddDispatch(processed, failed int) {
	atomic.AddUint64(&m.processed, uint64(processed))
	atomic.AddUint64(&m.failed, uint64(failed))
}
func (m *Metrics) AddAlertScan() { atomic.AddUint64(&m.alertScans, 1) }
func (m *Metrics) AddError()     { atomic.AddUint64(&m.errors, 1) }
func (m *Metrics) Snapshot() (processed, failed, scans, errors uint64) {
	return atomic.LoadUint64(&m.processed), atomic.LoadUint64(&m.failed), atomic.LoadUint64(&m.alertScans), atomic.LoadUint64(&m.errors)
}
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("ok\n"))
		case "/metrics":
			processed, failed, scans, errors := m.Snapshot()
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			_, _ = fmt.Fprintf(w, "saas_worker_dispatch_processed_total %d\nsaas_worker_dispatch_failed_total %d\nsaas_worker_alert_scans_total %d\nsaas_worker_errors_total %d\n", processed, failed, scans, errors)
		default:
			http.NotFound(w, r)
		}
	})
}
