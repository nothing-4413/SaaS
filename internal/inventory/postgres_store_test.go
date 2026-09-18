package inventory

import "testing"

func TestDocumentLineOrderIsStableByInventoryKey(t *testing.T) {
	lines := []DocumentLine{
		{WarehouseID: "w2", SKUID: "s2"},
		{WarehouseID: "w1", SKUID: "s3"},
		{WarehouseID: "w1", SKUID: "s1"},
	}
	indices := documentLineOrder(lines)
	if got := indices; len(got) != 3 || got[0] != 2 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("indices=%v", got)
	}
}
