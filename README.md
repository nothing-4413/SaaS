# 多租户库存与订单协同 SaaS

当前已完成基础身份与组织模块（内存存储版本），为后续 PostgreSQL 持久化保留 `Store` 接口。

## 运行

```bash
go run ./cmd/api
```

## API（第一阶段）

- `POST /organizations` 创建组织：`{"name":"Acme"}`
- `POST /organizations/{id}/roles` 创建角色
- `GET /organizations/{id}/roles` 查询角色
- `POST /organizations/{id}/users` 创建用户（角色必须属于同一组织）
- `GET /organizations/{id}/users` 查询用户

组织 ID 由服务端生成，所有用户和角色操作都显式绑定组织 ID，避免跨租户关联。

## 库存 API

- `GET /organizations/{id}/stocks` 查询组织库存
- `GET /organizations/{id}/warehouses/{warehouseId}/skus/{skuId}/stock` 查询单个库存
- `POST .../stock/receive` 入库，参数：`{"quantity":10,"idempotency_key":"receipt-1"}`
- `POST .../stock/reserve` 预占库存
- `POST .../stock/release` 释放预占
- `POST .../stock/deduct` 扣减库存

库存变更在服务层使用互斥锁保证并发安全；相同组织下重复使用同一幂等键不会重复变更库存，参数不一致会返回冲突。

## 销售订单 API

- `POST /organizations/{id}/orders` 创建订单并批量预占库存
- `GET /organizations/{id}/orders` 查询订单
- `POST /organizations/{id}/orders/{orderId}/confirm` 确认订单并扣减已预占库存
- `POST /organizations/{id}/orders/{orderId}/cancel` 取消订单并释放预占库存

订单创建失败时会补偿释放此前已经成功预占的明细；相同订单幂等键重复提交会返回原订单。

## 经营报表 API

- `GET /organizations/{id}/reports/summary`
- 可选参数 `low_stock_threshold`，统计可用库存小于等于该阈值的库存项

报表返回订单状态数量、已确认销售额，以及库存总量、预占量、可用量和低库存数量，并始终按组织隔离。

## Outbox 与审计基础

`internal/outbox` 提供事件入队、幂等去重、批量领取、成功确认和失败重试。事件达到最大尝试次数后进入终态 `failed`，不会再次被领取；非终态失败使用递增退避时间。

`internal/audit` 提供组织级审计记录模型和存储接口，记录操作者、动作、资源和扩展元数据，查询时按组织隔离。

## PostgreSQL 迁移

`migrations/000001_initial_schema.up.sql` 定义第一版持久化结构，覆盖租户、权限、商品、仓库、库存、订单、Outbox 和审计日志。库存表通过 `CHECK (reserved <= on_hand)` 保证预占量不会超过现有库存；订单和 Outbox 使用组织范围内的幂等唯一约束。

迁移文件兼容 golang-migrate、goose 等常见工具。
