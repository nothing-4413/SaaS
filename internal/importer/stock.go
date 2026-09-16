package importer

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/nothing-4413/saas/internal/inventory"
)

var ErrInvalidCSV = errors.New("invalid stock csv")

func Stocks(r io.Reader, organizationID string) ([]inventory.Stock, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, ErrInvalidCSV
	}
	c := csv.NewReader(r)
	header, e := c.Read()
	if e != nil {
		return nil, ErrInvalidCSV
	}
	want := []string{"organization_id", "warehouse_id", "sku_id", "on_hand", "reserved", "available"}
	if len(header) != len(want) {
		return nil, ErrInvalidCSV
	}
	for i, v := range want {
		if strings.TrimSpace(header[i]) != v {
			return nil, ErrInvalidCSV
		}
	}
	out := []inventory.Stock{}
	for {
		row, e := c.Read()
		if e == io.EOF {
			break
		}
		if e != nil || len(row) != 6 {
			return nil, ErrInvalidCSV
		}
		if strings.TrimSpace(row[0]) != organizationID {
			return nil, ErrInvalidCSV
		}
		on, e1 := strconv.ParseInt(row[3], 10, 64)
		res, e2 := strconv.ParseInt(row[4], 10, 64)
		avail, e3 := strconv.ParseInt(row[5], 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || on < 0 || res < 0 || avail < 0 || res > on || avail != on-res || strings.TrimSpace(row[1]) == "" || strings.TrimSpace(row[2]) == "" {
			return nil, ErrInvalidCSV
		}
		out = append(out, inventory.Stock{OrganizationID: organizationID, WarehouseID: row[1], SKUID: row[2], OnHand: on, Reserved: res, Available: avail})
	}
	return out, nil
}
