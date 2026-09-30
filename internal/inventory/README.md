# internal/inventory — 库存与单据

库存变动与入库/出库单据模块，保证并发安全、幂等与可补偿。

## 模型

- `Stock` — 某个仓库/某 SKU 的库存余额：`OnHand`（现有）、`Reserved`（预占）、`Available`（可用，派生为 `on_hand - reserved`）。
- `StockOperationInput` — 变动请求，含数量与幂等键。
- `Document` / `DocumentLine` / `DocumentInput` — 入库单（`receipt`）与出库单（`issue`）及其明细。

## 库存操作

`Service` 提供五类原子操作，均要求幂等键：

- `Receive` 入库（增加 `on_hand`）
- `Reserve` 预占（校验可用量后增加 `reserved`）
- `Release` 释放预占（减少 `reserved`）
- `Deduct` 扣减可用库存（减少 `on_hand`）
- `ConsumeReserved` 消费预占（同时减少 `on_hand` 与 `reserved`，用于订单确认）

PostgreSQL 实现（`AtomicStore`）在事务内使用 `FOR UPDATE` 行锁与 `inventory_operations` 幂等表；内存实现使用互斥锁与内存幂等映射。相同幂等键重复提交不会重复变更库存，参数不一致返回冲突。

## 单据

- `CreateReceipt` / `CreateIssue` 创建入库/出库单，使用 `idempotency_key` 防重复记账。
- PostgreSQL 下 `CreateDocumentAtomic` 在单个事务内完成库存变更、单据写入与明细落库；多明细中途失败时整批回滚。
- 内存实现保留补偿路径，逐行应用并在失败时回滚已完成的行。
- `ListDocuments` / `GetDocument` 按组织与单据类型查询。

## HTTP 接口

- `GET /organizations/{id}/stocks` 查询组织库存
- `GET /organizations/{id}/warehouses/{warehouseId}/skus/{skuId}/stock` 查询单个库存
- `POST .../stock/receive|reserve|release|deduct`
- `POST/GET /organizations/{id}/receipts`、`GET /organizations/{id}/receipts/{documentId}`
- `POST/GET /organizations/{id}/issues`、`GET /organizations/{id}/issues/{documentId}`

库存行锁辅助函数见 [`internal/platform/sqltx`](../platform/sqltx) 的 `LockStock`。
