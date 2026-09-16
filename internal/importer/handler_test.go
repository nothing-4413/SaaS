package importer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
