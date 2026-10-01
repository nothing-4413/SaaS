# internal/alert

库存预警：`RuleService` 管理每组织阈值规则，`Service.Scan` 把低于阈值的库存写入 Outbox `stock.low` 事件（按更新时间+可用量去重）。
