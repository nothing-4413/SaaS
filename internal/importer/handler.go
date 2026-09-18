package importer

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
)

type Handler struct{ inventory *inventory.Service }

func NewHandler(services ...*inventory.Service) *Handler {
	var service *inventory.Service
	if len(services) > 0 {
		service = services[0]
	}
	return &Handler{inventory: service}
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) == 4 && p[0] == "organizations" && p[2] == "imports" && p[3] == "stocks" && r.Method == http.MethodPost {
		values, err := Stocks(r.Body, p[1])
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		if h.inventory != nil {
			if err := h.inventory.ImportStocks(p[1], values); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"imported": len(values), "stocks": values})
		return
	}
	http.NotFound(w, r)
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
