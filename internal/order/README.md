# internal/order — 销售订单

销售订单模块，管理订单创建、预占库存与状态流转。

## 模型与状态机

`Status` 枚举：`pending → confirmed → paid → shipped → completed`，以及 `cancelled` / `refunded` 终态。`CanTransition` 显式描述合法迁移：

| 当前状态 | 允许迁移 |
| --- | --- |
| `pending` | `confirmed`、`cancelled` |
| `confirmed` | `paid` |
| `paid` | `shipped`、`refunded` |
| `shipped` | `completed`、`refunded` |
| `completed` | `refunded` |

- `Order` 含 `IdempotencyKey`、`Lines` 与 `TotalCents`；`Line` 含 SKU、仓库、数量与单价。
- 创建时预占库存（`pending`），确认时消费预占（`consume_reserved`），取消时释放预占，退款时回补库存。

## 关键行为

- **权威定价**：`SetCatalog` 注入商品目录后，订单单价取自组织内 SKU 当前售价（`UnitPriceCents` 被覆盖），服务端不信任客户端提交的单价。
- **幂等**：组织范围内 `IdempotencyKey` 唯一；重复提交返回原订单。
- **补偿**：创建失败会释放此前已成功预占的明细；状态迁移失败会回滚库存。
- **事件**：状态变化写入 Outbox 事件（`order.created`、`order.confirmed`、`order.refunded` 等）。
- PostgreSQL 下 `CreateAtomic` / `TransitionAtomic` 在同一事务内完成行锁、库存变更、订单写入与 Outbox 事件。

## HTTP 接口

- `POST/GET /organizations/{id}/orders`
- `POST /organizations/{id}/orders/{orderId}/confirm|pay|ship|complete|refund|cancel`

列表支持 `status` 过滤、`q` 搜索、`sort=total_cents` 排序与分页。
