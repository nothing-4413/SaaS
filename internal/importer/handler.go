package importer

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) == 4 && p[0] == "organizations" && p[2] == "imports" && p[3] == "stocks" && r.Method == http.MethodPost {
		values, err := Stocks(r.Body, p[1])
		if err != nil {
			writeError(w, 400, err.Error())
			return
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
