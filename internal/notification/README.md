# internal/notification — 通知分发

基于 Outbox 的异步通知分发器，以及 SMTP 邮件发送器。

## 分发器（`Service`）

- `Enqueue` — 以 `notification` 为聚合类型入队 Outbox 事件。
- `DispatchOnce(limit, sender)` — 批量领取事件并交给注入的 `Sender` 处理；发送成功标记 `published`，失败进入 Outbox 退避重试。
- `RunUntilEmpty(sender)` — 循环派发直至队列为空（主要用于测试）。

`Sender` 由业务侧注入（`func(outbox.Event) error`），可接邮件、短信、Webhook 或消息队列。`cmd/worker` 中按事件类型路由到邮件或 Webhook 投递。

## 邮件发送器（`EmailSender`）

- 处理 `auth.password_reset_requested` 事件，构造包含重置链接（`APP_PUBLIC_URL` + `/reset-password`）的纯文本邮件。
- 通过 SMTP 投递，可选 STARTTLS 与 SMTP AUTH；`SendMail` 可注入便于测试。
- 校验收件人与发件人地址、拒绝换行注入；配置缺失时 `Enabled()` 为 false，事件被跳过而非失败。

## 配置

SMTP 相关环境变量见 [`.env.example`](../../.env.example)：`SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`、`APP_PUBLIC_URL`。
