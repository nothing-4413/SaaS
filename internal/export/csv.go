package export

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/nothing-4413/saas/internal/inventory"
	"github.com/nothing-4413/saas/internal/order"
)

func Orders(w io.Writer, values []order.Order) error {
	c := csv.NewWriter(w)
	if err := c.Write([]string{"id", "organization_id", "status", "total_cents", "created_at"}); err != nil {
		return err
	}
	for _, v := range values {
		if err := c.Write([]string{v.ID, v.OrganizationID, string(v.Status), strconv.FormatInt(v.TotalCents, 10), v.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
func Stocks(w io.Writer, values []inventory.Stock) error {
	c := csv.NewWriter(w)
	if err := c.Write([]string{"organization_id", "warehouse_id", "sku_id", "on_hand", "reserved", "available"}); err != nil {
		return err
	}
	for _, v := range values {
		if err := c.Write([]string{v.OrganizationID, v.WarehouseID, v.SKUID, strconv.FormatInt(v.OnHand, 10), strconv.FormatInt(v.Reserved, 10), strconv.FormatInt(v.Available, 10)}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
