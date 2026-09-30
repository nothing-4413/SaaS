# internal/config — 环境配置

集中读取与校验服务运行配置。

## `Config`

字段：`HTTPAddr`、`PostgresURL`、`AuthTokenSecret`、`AuthTokenPreviousSecrets`、`Environment`、`SMTPHost/Port/Username/Password/From`、`AppPublicURL`。

- `Load()` — 从环境变量读取，提供开发默认值。
- `ValidateAPI()` — 校验 `DATABASE_URL` 非空、`AUTH_TOKEN_SECRET` ≥32 字符、生产环境不得使用默认密钥、旧密钥列表每条 ≥32 字符。
- `IntEnv(key, fallback)` — 读取整数环境变量，解析失败回退默认值。

## 环境变量

完整清单见 [`.env.example`](../../.env.example) 与 [`.env.production.example`](../../.env.production.example)，包括 `HTTP_ADDR`、`DATABASE_URL`、`AUTH_TOKEN_SECRET`、`AUTH_TOKEN_PREVIOUS_SECRETS`、`APP_ENV`、SMTP 相关变量，以及 Worker 的 `OUTBOX_POLL_INTERVAL_MS`、`OUTBOX_BATCH_SIZE`、`STOCK_ALERT_INTERVAL_SECONDS`、`WORKER_HEALTH_ADDR`。
