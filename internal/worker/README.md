# internal/worker — Worker 指标

`cmd/worker` 使用的健康检查与 Prometheus 指标组件。

## 指标（`Metrics`）

- `AddDispatch(processed, failed)` — 记录 Outbox 派发成功/失败数。
- `AddAlertScan` / `AddError` — 记录预警扫描次数与内部错误数。
- `Snapshot` — 原子读取全部计数。
- `Handler` — 暴露 `/healthz`（存活）与 `/metrics`（Prometheus 文本）：
  - `saas_worker_dispatch_processed_total`
  - `saas_worker_dispatch_failed_total`
  - `saas_worker_alert_scans_total`
  - `saas_worker_errors_total`

## 使用

由 [`cmd/worker`](../../cmd/worker) 在 `WORKER_HEALTH_ADDR`（默认 `:9090`）启动健康服务器，Compose 中该端口用于健康检查。
