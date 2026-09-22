# 架构与一致性说明

项目采用模块化单体结构，各领域通过 Store/Provider 接口隔离；测试使用内存实现，生产运行时使用 PostgreSQL 持久化，并由独立 Worker 处理 Outbox 和 Webhook。

## 关键一致性边界

1. 库存操作在单个服务临界区内串行化；PostgreSQL 版本应使用 `SELECT ... FOR UPDATE`。

`internal/platform/sqltx` 提供标准 `database/sql` 事务封装和库存行锁查询，驱动可选 pgx 或 lib/pq。
2. PostgreSQL 订单创建、确认、取消以及入库/出库单会在同一事务内完成库存行锁定、库存更新、业务记录和 Outbox 写入；内存实现保留补偿路径用于单元测试。
3. 订单确认消费已预占库存，普通出库只允许扣减可用库存，所有库存相关行按仓库和 SKU 稳定排序加锁。
4. 订单、库存和 Outbox 事件使用幂等键，重复请求不会重复记账。
5. Outbox 消费失败使用递增退避，达到最大次数后进入 `failed` 终态。
6. 所有业务资源携带 `organization_id`，查询和组合外键均按租户隔离。

## 推荐生产部署

- API 实例无状态运行在负载均衡后。
- PostgreSQL 使用主从或云数据库高可用版本。
- Outbox 消费者独立扩容，Webhook 发送器设置超时和重试。
- 日志、指标和审计记录写入集中式平台。
