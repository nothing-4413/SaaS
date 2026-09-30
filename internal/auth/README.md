# internal/auth — 多租户与身份认证

组织、用户、角色与权限的核心领域模块，同时提供无状态令牌、会话与密码重置能力。

## 模型

- `Organization` — 租户，所有业务资源都归属某个组织。
- `Role` — 组织内角色，携带 `[]Permission`；`owner` 角色在创建组织时自动建立并授予全部权限。
- `User` — 组织内用户，通过 `RoleIDs` 关联角色，密码以 bcrypt 哈希存储（`PasswordHash` 不对外序列化）。
- `Permission` — 细粒度权限常量，如 `user:read`、`inventory:import`、`order:approve` 等，并保留 `*:manage` 兼容旧权限（`permissionIncludes` 实现包含语义）。
- `Session` — 服务端会话，支持撤销与过期校验。

## 核心能力

- 组织创建：`CreateOrganization` 在同一事务内创建组织、`owner` 角色和首位所有者用户。
- 认证：`Authenticate` 校验邮箱/密码；连续失败 5 次锁定 15 分钟，成功登录清除失败计数。
- 令牌：`IssueToken` / `ParseTokenWithSecrets` 签发与校验 HMAC-SHA256 无状态令牌（`header.payload.signature`），支持通过 `AUTH_TOKEN_PREVIOUS_SECRETS` 轮换旧密钥。
- 会话：`POST /organizations/{id}/sessions` 登录签发 15 分钟访问令牌与 30 天刷新令牌；`PUT` 刷新、`DELETE` 撤销。
- 密码重置：`RequestPasswordReset` 生成 15 分钟一次性令牌（存 SHA-256 哈希），通过 Outbox 事件 `auth.password_reset_requested` 异步投递；`ResetPassword` 校验边界、更新密码并撤销全部既有会话。请求接口对存在/不存在的邮箱统一返回 `202` 防枚举。
- 权限判断：`HasPermission` 按组织校验用户角色权限。

## 中间件

- `RequireTokenPermissions` / `RequireTokenPermissionResolver` — 解析 `Authorization: Bearer <token>`，校验签名、会话有效性与组织边界（令牌租户与 URL 租户不一致返回 403），把 `Claims` 注入 context 供下游（如审计）使用。
- `RequirePermission` — 基于 `X-Organization-ID` / `X-User-ID` 头的兼容实现。

## 接口

- `POST /organizations` 创建组织与首位所有者
- `GET/PUT /organizations/{id}` 查询/更新组织
- `POST/GET /organizations/{id}/roles`、`PUT /organizations/{id}/roles/{roleId}`
- `POST/GET /organizations/{id}/users`、`PUT /organizations/{id}/users/{userId}`
- `POST/PUT/DELETE /organizations/{id}/sessions`
- `POST/PUT /organizations/{id}/sessions/password-reset`

相关存储：`store.go`（接口）、`postgres_store.go`（PostgreSQL 实现）；测试使用内存 Store。
