package httpx

import (
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
