# internal/audit — 审计日志

组织级审计记录，自动记录所有已认证写请求。

## 模型

`Entry` 记录组织、操作者（`ActorUserID`）、动作（HTTP 方法）、资源类型/ID、元数据（路径、响应状态、Request ID）与时间。

## 服务（`Service`）

- `Record` — 追加一条审计记录（PostgreSQL 持久化）。
- `List` — 按组织查询。

## 中间件（`Middleware`）

包裹在受保护的 Handler 外，仅记录非 GET/HEAD/OPTIONS 的已认证请求：从 context 读取 `Claims`，执行后把动作、资源路径、响应状态和 Request ID 写入审计。`resourceFromPath` 从 URL 提取资源类型与 ID。

## HTTP 接口

- `GET /organizations/{id}/audit-logs`（要求 `audit:read`），支持 `action`、`resource_type` 过滤、`q` 搜索与分页，默认返回最近记录。

相关表见 [`migrations/000001_initial_schema.up.sql`](../../migrations/000001_initial_schema.up.sql) 的 `audit_logs`。
