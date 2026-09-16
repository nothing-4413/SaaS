package sqltx

import (
	"context"
	"database/sql"
	"errors"
)

var ErrInvalidInput = errors.New("invalid sql transaction input")

// WithTx executes fn in a transaction and rolls back on every error or panic.
// The caller chooses the PostgreSQL driver (pgx, lib/pq, or database/sql proxy).
func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	if db == nil || fn == nil {
		return ErrInvalidInput
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	err = fn(tx)
	return err
}

// StockRow is the lockable inventory balance used by reserve/deduct flows.
type StockRow struct{ OnHand, Reserved int64 }

func LockStock(ctx context.Context, tx *sql.Tx, organizationID, warehouseID, skuID string) (StockRow, error) {
	if tx == nil || organizationID == "" || warehouseID == "" || skuID == "" {
		return StockRow{}, ErrInvalidInput
	}
	var row StockRow
	err := tx.QueryRowContext(ctx, `
		SELECT on_hand, reserved
		FROM inventory_stocks
		WHERE organization_id = $1 AND warehouse_id = $2 AND sku_id = $3
		FOR UPDATE`, organizationID, warehouseID, skuID).Scan(&row.OnHand, &row.Reserved)
	return row, err
}
