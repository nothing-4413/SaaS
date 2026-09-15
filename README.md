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
