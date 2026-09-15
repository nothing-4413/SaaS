package order

import (
	"encoding/json"
	"errors"
	"github.com/nothing-4413/saas/internal/inventory"
	"net/http"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) >= 3 && p[0] == "organizations" && p[2] == "orders" {
		org := p[1]
		if len(p) == 3 && r.Method == http.MethodPost {
			var in CreateInput
			if !decode(w, r, &in) {
				return
			}
			v, e := h.service.Create(org, in)
			if e != nil {
				writeError(w, stat(e), e.Error())
				return
			}
			writeJSON(w, 201, v)
			return
		}
		if len(p) == 3 && r.Method == http.MethodGet {
			writeJSON(w, 200, h.service.List(org))
			return
		}
		if len(p) == 5 && r.Method == http.MethodPost {
			var v Order
			var e error
			if p[4] == "confirm" {
				v, e = h.service.Confirm(org, p[3])
			} else if p[4] == "cancel" {
				v, e = h.service.Cancel(org, p[3])
			} else {
				writeError(w, 404, "unknown order action")
				return
			}
			if e != nil {
				writeError(w, stat(e), e.Error())
				return
			}
			writeJSON(w, 200, v)
			return
		}
	}
	http.NotFound(w, r)
}
func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if e := json.NewDecoder(r.Body).Decode(v); e != nil {
		writeError(w, 400, "invalid JSON")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, code int, m string) {
	writeJSON(w, code, map[string]string{"error": m})
}
func stat(e error) int {
	if errors.Is(e, ErrNotFound) {
		return 404
	}
	if errors.Is(e, ErrConflict) {
		return 409
	}
	if errors.Is(e, inventory.ErrInsufficient) {
		return 409
	}
	return 400
}
