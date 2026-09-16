package export

import (
	"net/http"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
)

type Handler struct {
	orders interface{ List(string) []order.Order }
	stocks interface {
		List(string) []inventory.Stock
	}
}

func NewHandler(orders interface{ List(string) []order.Order }, stocks interface {
	List(string) []inventory.Stock
}) *Handler { return &Handler{orders: orders, stocks: stocks} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) == 5 && p[0] == "organizations" && p[2] == "exports" && r.Method == http.MethodGet {
		org := p[1]
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+p[3]+".csv")
		if p[3] == "orders" && p[4] == "csv" {
			_ = Orders(w, h.orders.List(org))
			return
		}
		if p[3] == "stocks" && p[4] == "csv" {
			_ = Stocks(w, h.stocks.List(org))
			return
		}
	}
	http.NotFound(w, r)
}
