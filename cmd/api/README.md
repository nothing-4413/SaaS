# cmd/api — HTTP API 服务

API 服务的入口程序（`main.go`），把各领域模块组装成一个无状态的 HTTP 服务。

## 职责

- 加载并校验配置（`config.Load` / `ValidateAPI`），连接 PostgreSQL。
- 组装各模块的 Store、Service 和 Handler，注入权限中间件与审计中间件。
- 依据 URL 结构分发到对应 Handler（`apiHandler.ServeHTTP`）。
- 叠加安全响应头、2 MiB 请求体限制、按直接连接地址的限流（每分钟 120 次）和 Prometheus 指标包装。
- 启动 HTTP 服务，监听 SIGINT/SIGTERM 后在 10 秒超时内优雅退出。

## 路由分发

路由按前缀匹配到领域 Handler，业务 API 均要求 Bearer Token 并由权限中间件保护：

| 前缀 / 路径 | Handler | 权限 |
| --- | --- | --- |
| `/organizations`（创建、`/sessions` 登录） | `auth` 公开 | 无 |
| `/organizations/{id}` | `auth` | `role:manage`（写）/ `user:read`（读） |
| `/organizations/{id}/users` | `auth` | `user:read` / `user:write` |
| `/organizations/{id}/roles` | `auth` | `role:read` / `role:manage` |
| `/organizations/{id}/products`、`/warehouses` | `product` | `product:read` / `product:write` |
| `/organizations/{id}/stocks`、`/stock`、`/receipts`、`/issues` | `inventory` | `inventory:read` / `inventory:write` |
| `/organizations/{id}/orders` | `order` | `order:read` / `order:write` / `order:approve` |
| `/organizations/{id}/reports` | `report` | `report:read` |
| `/organizations/{id}/exports` | `export` | `report:export` |
| `/organizations/{id}/imports` | `importer` | `inventory:import` |
| `/organizations/{id}/audit-logs` | `audit` | `audit:read` |
| `/organizations/{id}/webhooks` | `webhook` | `webhook:manage` |
| `/organizations/{id}/alerts/stock` | `alert` | `inventory:read` / `inventory:write` |
| `/healthz`、`/readyz`、`/metrics`、管理台静态资源 | 内部 | 无 |

## 中间件链

`httpx.Chain` 依次叠加 `RequestID → AccessLog → Recover`，外层再包 `SecurityHeaders → MaxBodyBytes → RateLimiter → Metrics`。每个受保护路由还会经过 `auth.RequireTokenPermission*` 与 `audit.Middleware`，使所有已认证写请求自动落审计记录。

## 运行

```bash
go run ./cmd/api
```

配置项见 [`.env.example`](../../.env.example)，完整容器栈使用 `docker compose up --build`。健康检查：`GET /healthz`（存活）与 `GET /readyz`（含 PostgreSQL 连通性）。
