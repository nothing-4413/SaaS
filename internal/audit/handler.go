package audit

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/nothing-4413/saas/internal/platform/pagination"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "organizations" && parts[2] == "audit-logs" && r.Method == http.MethodGet {
		p, err := pagination.Parse(r.URL.Query())
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		items := h.service.List(parts[1])
		if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
			filtered := items[:0]
			for _, v := range items {
				if v.Action == action {
					filtered = append(filtered, v)
				}
			}
			items = filtered
		}
		if resource := strings.TrimSpace(r.URL.Query().Get("resource_type")); resource != "" {
			filtered := items[:0]
			for _, v := range items {
				if v.ResourceType == resource {
					filtered = append(filtered, v)
				}
			}
			items = filtered
		}
		if p.Query != "" {
			q := strings.ToLower(p.Query)
			filtered := items[:0]
			for _, v := range items {
				if strings.Contains(strings.ToLower(v.ResourceID), q) || strings.Contains(strings.ToLower(v.Action), q) {
					filtered = append(filtered, v)
				}
			}
			items = filtered
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].CreatedAt.Equal(items[j].CreatedAt) {
				if p.Desc {
					return items[i].ID > items[j].ID
				}
				return items[i].ID < items[j].ID
			}
			if p.Desc {
				return items[i].CreatedAt.After(items[j].CreatedAt)
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})
		writeJSON(w, http.StatusOK, pagination.Slice(items, p))
		return
	}
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
