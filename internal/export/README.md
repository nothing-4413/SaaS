# internal/export — CSV 导出

订单与库存的 CSV 导出模块，可接入后台任务或 HTTP 下载接口。

## 函数

- `Orders(w, values)` — 导出订单，列：`id, organization_id, status, total_cents, created_at`。
- `Stocks(w, values)` — 导出库存，列：`organization_id, warehouse_id, sku_id, on_hand, reserved, available`。

使用标准库 `encoding/csv`，写完自动 `Flush` 并返回错误。

## HTTP 接口

- `GET /organizations/{id}/exports/orders/csv`
- `GET /organizations/{id}/exports/stocks/csv`

响应设置 `Content-Type: text/csv` 与 `Content-Disposition: attachment`，数据来自订单/库存服务的 `List`，始终按组织隔离。

库存 CSV 的对称导入见 [`internal/importer`](../importer)。
