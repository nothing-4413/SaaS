# cmd/worker

后台 Worker：周期领取 Outbox 事件并投递通知/Webhook，扫描库存预警规则，暴露 `/healthz` 与 `/metrics`。

运行：`go run ./cmd/worker`
