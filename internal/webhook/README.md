# internal/webhook — Webhook 订阅与投递

带 HMAC-SHA256 签名的 Webhook 推送，支持订阅管理与按事件去重投递。

## 订阅管理（`SubscriptionService`）

- `Create` — 校验 URL（http/https）、签名密钥（至少 16 字符）与事件类型列表；`*` 表示订阅全部事件。
- `List` / `Delete` — 按组织查询/删除订阅。
- `Deliver` — 对匹配事件类型的启用订阅逐个投递，按 `(event_id, subscription_id)` 去重（`MarkDelivered`）。

## 发送器（`Sender`）

- `Send` 发送 POST 请求，携带 `Content-Type: application/json` 与签名头：
  - `X-Webhook-Event`、`X-Webhook-Id`、`X-Idempotency-Key`、`X-Webhook-Signature`（`sha256=...`）。
- 默认超时 10 秒，非 2xx 响应返回错误交给 Outbox 重试。
- `validateTarget` 默认阻止 localhost、私有/环回地址的 SSRF 攻击；`AllowPrivate` 仅供测试打开。

## 安全细节

- 密码重置事件（`auth.password_reset_requested`）含一次性密钥，`*` 通配订阅不会收到它，需显式订阅该事件类型。
- 签名使用 `Sign(secret, payload)` 计算 HMAC-SHA256 十六进制摘要。

## HTTP 接口

- `POST/GET /organizations/{id}/webhooks`
- `DELETE /organizations/{id}/webhooks/{webhookId}`

相关表见 [`migrations/000006_webhook_subscriptions.up.sql`](../../migrations/000006_webhook_subscriptions.up.sql) 与 `000008_webhook_deliveries.up.sql`。
