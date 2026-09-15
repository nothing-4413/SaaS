# 数据库迁移说明

执行库存扣减或预占时，应在同一事务中锁定库存行：

```sql
SELECT on_hand, reserved
FROM inventory_stocks
WHERE organization_id = $1 AND warehouse_id = $2 AND sku_id = $3
FOR UPDATE;
```

校验通过后更新 `on_hand` / `reserved`，并在同一事务插入 `outbox_events`。事务提交后由后台消费者领取 `pending` 事件并发布消息；发布成功更新为 `published`，失败则写入退避时间并重试。

所有业务表都带有 `organization_id`，跨租户关联使用组合外键约束，避免仅凭资源 ID 产生越权关联。
