package inventory

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/nothing-4413/saas/internal/platform/pagination"
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
		h.listStocks(w, r, p[1])
		return
	}
	if len(p) >= 3 && p[0] == "organizations" && (p[2] == "receipts" || p[2] == "issues") {
		org, typ := p[1], DocumentReceipt
		if p[2] == "issues" {
			typ = DocumentIssue
		}
		if len(p) == 3 && r.Method == http.MethodPost {
			var in DocumentInput
			if e := json.NewDecoder(r.Body).Decode(&in); e != nil {
				writeError(w, 400, "invalid JSON")
				return
			}
			var v Document
			var e error
			if typ == DocumentReceipt {
				v, e = h.service.CreateReceipt(org, in)
			} else {
				v, e = h.service.CreateIssue(org, in)
			}
			if e != nil {
				writeError(w, status(e), e.Error())
				return
			}
			writeJSON(w, 201, v)
			return
		}
		if len(p) == 3 && r.Method == http.MethodGet {
			h.listDocuments(w, r, org, typ)
			return
		}
		if len(p) == 4 && r.Method == http.MethodGet {
			v, e := h.service.GetDocument(org, p[3])
			if e != nil {
				writeError(w, 404, e.Error())
				return
			}
			writeJSON(w, 200, v)
			return
		}
	}
	http.NotFound(w, r)
}

func (h *Handler) listStocks(w http.ResponseWriter, r *http.Request, org string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.List(org)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.WarehouseID), q) || strings.Contains(strings.ToLower(v.SKUID), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i].WarehouseID+"\x00"+items[i].SKUID, items[j].WarehouseID+"\x00"+items[j].SKUID
		if p.Sort == "available" {
			if items[i].Available == items[j].Available {
				return items[i].SKUID < items[j].SKUID
			}
			if p.Desc {
				return items[i].Available > items[j].Available
			}
			return items[i].Available < items[j].Available
		}
		if left == right {
			return false
		}
		if p.Desc {
			return left > right
		}
		return left < right
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request, org string, typ DocumentType) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListDocuments(org, typ)
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
