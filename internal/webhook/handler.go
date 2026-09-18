package webhook

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct{ service *SubscriptionService }

func NewHandler(service *SubscriptionService) *Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "organizations" && parts[2] == "webhooks" {
		org := parts[1]
		if len(parts) == 3 && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, h.service.List(org))
			return
		}
		if len(parts) == 3 && r.Method == http.MethodPost {
			var input CreateSubscriptionInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
				return
			}
			value, err := h.service.Create(org, input)
			if err != nil {
				writeJSON(w, statusForError(err), map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, value)
			return
		}
		if len(parts) == 4 && r.Method == http.MethodDelete {
			if err := h.service.Delete(org, parts[3]); err != nil {
				writeJSON(w, statusForError(err), map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusNoContent)
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
