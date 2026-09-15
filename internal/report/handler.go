package report

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) == 4 && p[0] == "organizations" && p[2] == "reports" && p[3] == "summary" && r.Method == http.MethodGet {
		threshold := int64(0)
		if raw := r.URL.Query().Get("low_stock_threshold"); raw != "" {
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || v < 0 {
				writeError(w, 400, "invalid low_stock_threshold")
				return
			}
			threshold = v
		}
		v, err := h.service.Summary(p[1], threshold)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, v)
		return
	}
	http.NotFound(w, r)
}
func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
