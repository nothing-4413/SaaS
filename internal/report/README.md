# internal/report — 经营报表

汇总组织订单与库存的只读报表模块。

## 模型

- `Summary` — 报表结果，含 `Orders`（各状态数量与已确认销售额）与 `Inventory`（SKU 数、总量、预占、可用与低库存数）。
- `OrderSummary` / `StockSummary` — 明细汇总。

## 服务（`Service`）

- `Summary(org, lowStockThreshold)` 遍历组织订单与库存，统计：
  - 订单各状态数量；`confirmed_cents` 累加 `confirmed`、`paid`、`shipped`、`completed` 状态的订单金额。
  - 库存总量、预占、可用，以及可用量 ≤ 阈值的低库存项数量。
- 通过 `OrderProvider` / `StockProvider` 接口注入订单与库存服务，始终按组织隔离。

## HTTP 接口

- `GET /organizations/{id}/reports/summary?low_stock_threshold=<n>`

`low_stock_threshold` 可选，默认统计可用库存小于等于该阈值的库存项。
