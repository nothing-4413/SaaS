# internal/platform/sqltx — 事务与行锁辅助

标准 `database/sql` 事务封装与库存行锁查询，驱动可选 pgx、lib/pq 或 database/sql 代理。

## `WithTx`

`WithTx(ctx, db, fn)` 在事务中执行 `fn`：

- 出错或 panic 自动回滚；
- 成功提交；panic 会先回滚再重新抛出。

## `LockStock`

`LockStock(ctx, tx, organizationID, warehouseID, skuID)` 使用 `SELECT ... FOR UPDATE` 锁定库存行并返回 `StockRow{OnHand, Reserved}`，供 reserve/deduct 等库存流程在事务内串行化。

各 PostgreSQL Repository（库存、订单、单据）复用此处的提交/回滚与行锁逻辑，保证多 API 实例下的一致性。
