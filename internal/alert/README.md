# internal/alert — 库存预警

库存预警规则与扫描模块，把低于阈值的库存写入 Outbox `stock.low` 事件。

## 规则（`RuleService`）

- `Rule` — 每组织一条规则：`Threshold`（阈值）、`Enabled`（启用）、`UpdatedAt`。
- `Update` / `Get` / `ListEnabled` — 查询与更新规则。

## 扫描（`Service`）

- `Scan(org, threshold)` 遍历组织库存，把可用量 ≤ 阈值的库存项入队为 `stock.low` 事件。
- 去重键由库存更新时间与可用量拼成，重复扫描不会重复产生告警。

## 运行

- 规则通过 `GET/PUT /organizations/{id}/alerts/stock` 管理。
- `cmd/worker` 默认每 60 秒（`STOCK_ALERT_INTERVAL_SECONDS`）对 `ListEnabled` 的规则执行一次扫描。

相关表见 [`migrations/000007_stock_alert_rules.up.sql`](../../migrations/000007_stock_alert_rules.up.sql)。
