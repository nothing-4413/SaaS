package alert

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct{ service *RuleService }

func NewHandler(service *RuleService) *Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 4 && parts[0] == "organizations" && parts[2] == "alerts" && parts[3] == "stock" {
		if r.Method == http.MethodGet {
			value, err := h.service.Get(parts[1])
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, ErrNotFound) {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, value)
			return
		}
		if r.Method == http.MethodPut {
			var input UpdateRuleInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
				return
			}
			value, err := h.service.Update(parts[1], input)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, value)
			return
		}
	}
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
