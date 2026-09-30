# 数据库迁移说明

迁移文件位于本目录，采用向上（`.up.sql`）/ 向下（`.down.sql`）成对结构，兼容 golang-migrate、goose 等常见工具。Compose 栈使用 `migrate/migrate` 镜像在 API 和 Worker 启动前执行 `up`，因此业务容器只在迁移成功后启动。

## 迁移清单

| 迁移 | 内容 |
| --- | --- |
| `000001_initial_schema` | 租户（organizations）、角色/用户/用户角色关联、商品/SKU/仓库、库存、订单与明细、Outbox、审计日志的初始结构 |
| `000002_inventory_documents` | 入库单、出库单及其明细表，组织范围幂等约束与组合外键 |
| `000003_inventory_operations` | 库存操作流水表，记录 receive/reserve/release/deduct/consume_reserved 的幂等键 |
| `000004_outbox_lease_index` | 为 `processing` 状态添加 `claimed_until` 租约索引 |
| `000005_user_passwords` | 用户表增加 `password_hash` 列 |
| `000006_webhook_subscriptions` | Webhook 订阅表，按组织 + URL 唯一 |
| `000007_stock_alert_rules` | 库存预警规则表，每组织一条，含阈值与启用状态 |
| `000008_webhook_deliveries` | Webhook 投递记录表，`(event_id, subscription_id)` 唯一防止重复投递 |
| `000009_auth_sessions` | 用户启用标记与会话表，支持撤销 |
| `000010_order_statuses` | 扩展 `order_status` 枚举：`paid`/`shipped`/`completed`/`refunded` |
| `000011_auth_login_attempts` | 登录失败计数与锁定时长表 |
| `000012_password_reset_tokens` | 一次性密码重置令牌表，按 SHA-256 哈希存储 |

## 关键约束

执行库存扣减或预占时，应在同一事务中锁定库存行：

```sql
SELECT on_hand, reserved
FROM inventory_stocks
WHERE organization_id = $1 AND warehouse_id = $2 AND sku_id = $3
FOR UPDATE;
```

校验通过后更新 `on_hand` / `reserved`，并在同一事务插入 `outbox_events`。事务提交后由后台消费者领取 `pending` 事件并发布消息；发布成功更新为 `published`，失败则写入退避时间并重试。

- 库存表通过 `CHECK (reserved <= on_hand)` 保证预占量不超过现有库存。
- 订单、Outbox 事件、库存操作与单据均使用组织范围内的幂等唯一约束。
- 所有业务表都带有 `organization_id`，跨租户关联使用组合外键约束，避免仅凭资源 ID 产生越权关联。

事务辅助代码见 [`internal/platform/sqltx`](../internal/platform/sqltx)，封装了事务提交/回滚与库存行锁查询，供各 PostgreSQL Repository 复用。
