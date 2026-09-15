package inventory

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) >= 7 && p[0] == "organizations" && p[2] == "warehouses" && p[4] == "skus" && p[6] == "stock" {
		org, warehouse, sku := p[1], p[3], p[5]
		if len(p) == 7 && r.Method == http.MethodGet {
			v, e := h.service.Get(org, warehouse, sku)
			if e != nil {
				writeError(w, 404, e.Error())
				return
			}
			writeJSON(w, 200, v)
			return
		}
		if len(p) == 8 && r.Method == http.MethodPost {
			h.mutate(w, r, org, warehouse, sku, p[7])
			return
		}
	}
	if len(p) == 3 && p[0] == "organizations" && p[2] == "stocks" && r.Method == http.MethodGet {
		writeJSON(w, 200, h.service.List(p[1]))
		return
	}
	http.NotFound(w, r)
}
func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, org, warehouse, sku, action string) {
	var in StockOperationInput
	if e := json.NewDecoder(r.Body).Decode(&in); e != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	var v Stock
	var e error
	switch action {
	case "receive":
		v, e = h.service.Receive(org, warehouse, sku, in)
	case "reserve":
		v, e = h.service.Reserve(org, warehouse, sku, in)
	case "release":
		v, e = h.service.Release(org, warehouse, sku, in)
	case "deduct":
		v, e = h.service.Deduct(org, warehouse, sku, in)
	default:
		writeError(w, 404, "unknown stock action")
		return
	}
	if e != nil {
		writeError(w, status(e), e.Error())
		return
	}
	writeJSON(w, 200, v)
}
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, m string) {
	writeJSON(w, status, map[string]string{"error": m})
}
func status(e error) int {
	if errors.Is(e, ErrNotFound) {
		return 404
	}
	if errors.Is(e, ErrConflict) {
		return 409
	}
	if errors.Is(e, ErrInsufficient) {
		return 409
	}
	return 400
}
