package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/platform/pagination"
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
			h.list(w, r, org)
			return
		}
		if len(p) == 5 && r.Method == http.MethodPost {
			var v Order
			var e error
			if p[4] == "confirm" {
				v, e = h.service.Confirm(org, p[3])
			} else if p[4] == "pay" {
				v, e = h.service.Pay(org, p[3])
			} else if p[4] == "ship" {
				v, e = h.service.Ship(org, p[3])
			} else if p[4] == "complete" {
				v, e = h.service.Complete(org, p[3])
			} else if p[4] == "refund" {
				v, e = h.service.Refund(org, p[3])
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
func (h *Handler) list(w http.ResponseWriter, r *http.Request, org string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.List(org)
	statusFilter := Status(r.URL.Query().Get("status"))
	if statusFilter != "" {
		valid := map[Status]bool{StatusPending: true, StatusConfirmed: true, StatusPaid: true, StatusShipped: true, StatusCompleted: true, StatusCancelled: true, StatusRefunded: true}
		if !valid[statusFilter] {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		filtered := items[:0]
		for _, v := range items {
			if v.Status == statusFilter {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.ID), q) || strings.Contains(strings.ToLower(v.IdempotencyKey), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		if p.Sort == "total_cents" && items[i].TotalCents != items[j].TotalCents {
			if p.Desc {
				return items[i].TotalCents > items[j].TotalCents
			}
			return items[i].TotalCents < items[j].TotalCents
		}
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
