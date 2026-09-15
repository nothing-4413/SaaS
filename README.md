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
