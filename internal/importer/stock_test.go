package importer

import (
	"strings"
	"testing"
)

func TestStocksImportValidatesTenantAndDerivedAvailable(t *testing.T) {
	csv := `organization_id,warehouse_id,sku_id,on_hand,reserved,available
o,w,s,10,3,7
`
	v, e := Stocks(strings.NewReader(csv), "o")
	if e != nil || len(v) != 1 || v[0].Available != 7 {
		t.Fatalf("v=%+v e=%v", v, e)
	}
	bad := strings.Replace(csv, "7\n", "8\n", 1)
	if _, e = Stocks(strings.NewReader(bad), "o"); e == nil {
		t.Fatal("expected derived field validation")
	}
}
