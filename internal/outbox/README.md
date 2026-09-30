# internal/outbox — 事务性事件队列

基于数据库的 Outbox 模式实现，保证业务写入与事件发布的最终一致性。

## 模型与状态

`Event` 记录事件的聚合类型、聚合 ID、事件类型、去重键（`DedupKey`）与 JSON 载荷。`Status` 状态机：

- `pending` — 等待领取
- `processing` — 已被消费者领取（带 `claimed_until` 租约）
- `published` — 发送成功
- `failed` — 达到最大尝试次数后的终态

## 服务（`Service`）

- `Enqueue` — 入队；`UNIQUE (organization_id, aggregate_type, aggregate_id, event_type, dedup_key)` 保证幂等去重。
- `Claim` — 批量领取到期事件（PostgreSQL 使用 `FOR UPDATE SKIP LOCKED`，支持多消费者并行）。
- `Succeed` — 标记 `published`。
- `Fail` — 失败写入递增退避时间（`attempts²` 秒）与错误信息；达到最大尝试次数（默认 5）进入终态 `failed`。
- `Get` / `List` — 查询事件。

事件领取带五分钟租约：消费者崩溃后租约过期，事件可被其他消费者重新领取。

## 与业务集成

订单、库存预警等模块在事务内写入 Outbox 事件，`cmd/worker` 通过 [`internal/notification`](../notification) 与 [`internal/webhook`](../webhook) 领取并投递。相关表结构见 [`migrations/000001_initial_schema.up.sql`](../../migrations/000001_initial_schema.up.sql)。
