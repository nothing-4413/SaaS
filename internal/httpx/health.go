package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Health struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Health{Status: "ok", Time: time.Now().UTC()})
}

type Pinger interface{ PingContext(context.Context) error }

func ReadinessHandler(pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if pinger == nil || pinger.PingContext(ctx) != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(Health{Status: "unavailable", Time: time.Now().UTC()})
			return
		}
		HealthHandler(w, r)
	}
}
