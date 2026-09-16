package export

import (
	"bytes"
	"github.com/nothing-4413/saas/internal/inventory"
	"strings"
	"testing"
)

func TestStocksCSV(t *testing.T) {
	var b bytes.Buffer
	if e := Stocks(&b, []inventory.Stock{{OrganizationID: "o", WarehouseID: "w", SKUID: "s", OnHand: 3, Reserved: 1, Available: 2}}); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(b.String(), "warehouse_id") || !strings.Contains(b.String(), "w,s,3,1,2") {
		t.Fatalf("csv=%s", b.String())
	}
}
