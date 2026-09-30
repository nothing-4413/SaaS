# cmd/worker — 后台 Worker

独立后台进程（`main.go`），与 API 服务解耦，负责消费 Outbox 事件并执行周期任务。

## 职责

- 周期性领取 Outbox 事件（默认每 1 秒，`OUTBOX_POLL_INTERVAL_MS`），批量投递（默认 100，`OUTBOX_BATCH_SIZE`）。
- 按事件类型选择投递通道：
  - `auth.password_reset_requested` → `notification.EmailSender`（SMTP 密码重置邮件）；
  - 其余事件 → `webhook.SubscriptionService.Deliver`（按订阅投递）。
- 周期性扫描启用中的库存预警规则（默认每 60 秒，`STOCK_ALERT_INTERVAL_SECONDS`），把低库存写入 Outbox `stock.low` 事件。
- 在 `WORKER_HEALTH_ADDR`（默认 `:9090`）暴露 `/healthz` 与 `/metrics`，输出派发、失败、扫描与错误计数。
- 监听 SIGINT/SIGTERM 优雅退出。

## 投递语义

- 事件发送成功 → `outbox.Service.Succeed` 标记 `published`；失败 → `outbox.Service.Fail` 进入退避重试，达到最大尝试次数后终态 `failed`。
- 邮件发送要求 SMTP 已配置（`SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`）；未配置时密码重置事件被跳过，不标记失败。
- Webhook 投递携带 HMAC-SHA256 签名，按 `(event_id, subscription_id)` 去重。

## 运行

```bash
go run ./cmd/worker
```

Compose 默认同时启动 Worker，配置见 [`.env.example`](../../.env.example) 与 [`docker-compose.yml`](../../docker-compose.yml)。
