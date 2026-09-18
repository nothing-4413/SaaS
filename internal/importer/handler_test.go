package importer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nothing-4413/saas/internal/inventory"
)

func TestStockImportHandler(t *testing.T) {
	body := `organization_id,warehouse_id,sku_id,on_hand,reserved,available
o,w,s,2,0,2
`
	req := httptest.NewRequest(http.MethodPost, "/organizations/o/imports/stocks", strings.NewReader(body))
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStockImportHandlerPersistsWholeBatch(t *testing.T) {
	service := inventory.NewService(inventory.NewMemoryStore())
	body := `organization_id,warehouse_id,sku_id,on_hand,reserved,available
o,w,s1,10,2,8
o,w,s2,4,0,4
`
	req := httptest.NewRequest(http.MethodPost, "/organizations/o/imports/stocks", strings.NewReader(body))
	w := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	values := service.List("o")
	if len(values) != 2 {
		t.Fatalf("stocks=%+v", values)
	}
}

func TestStockImportHandlerRejectsDuplicatesWithoutWriting(t *testing.T) {
	service := inventory.NewService(inventory.NewMemoryStore())
	body := `organization_id,warehouse_id,sku_id,on_hand,reserved,available
o,w,s,10,2,8
o,w,s,4,0,4
`
	req := httptest.NewRequest(http.MethodPost, "/organizations/o/imports/stocks", strings.NewReader(body))
	w := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if values := service.List("o"); len(values) != 0 {
		t.Fatalf("partial import: %+v", values)
	}
}
