package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nothing-4413/saas/internal/inventory"
)

func TestListOrdersPaginationAndStatusFilter(t *testing.T) {
	inv := inventory.NewService(inventory.NewMemoryStore())
	if _, err := inv.Receive("org", "warehouse", "sku", inventory.StockOperationInput{Quantity: 10, IdempotencyKey: "seed"}); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewMemoryStore(), inv)
	for _, key := range []string{"one", "two"} {
		if _, err := service.Create("org", CreateInput{IdempotencyKey: key, Lines: []Line{{SKUID: "sku", WarehouseID: "warehouse", Quantity: 1, UnitPriceCents: 100}}}); err != nil {
			t.Fatal(err)
		}
	}
	res := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/organizations/org/orders?page_size=1&status=pending", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body struct {
		Items []Order `json:"items"`
		Total int     `json:"total"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Total != 2 || body.Items[0].Status != StatusPending {
		t.Fatalf("body=%+v", body)
	}
}
