# internal/httpx — HTTP 基础设施

可复用的 HTTP 中间件、健康检查、指标与安全组件。

## 健康检查（`health.go`）

- `HealthHandler` — `GET /healthz` 存活检查，返回 `{"status":"ok"}`。
- `ReadinessHandler(pinger)` — `GET /readyz` 就绪检查，`PingContext` 失败时返回 `503`。

## 指标（`metrics.go`）

- `Metrics` 用原子计数统计请求总数、错误数（5xx）与响应字节数。
- `Wrap` 包装 Handler 记录计数；`ServeHTTP` 以 Prometheus 文本格式暴露 `/metrics`。

## 中间件（`middleware.go`）

- `RequestID` — 读取/校验 `X-Request-ID`（长度 ≤128、拒绝控制字符），无效则生成随机 ID。
- `AccessLog` — 结构化访问日志（方法、路径、状态、字节、耗时、Request ID）。
- `Recover` — panic 恢复，返回 `500`，避免单个异常导致进程退出。
- `Chain` — 依次组合 `RequestID → AccessLog → Recover`。

## 安全（`security.go`）

- `SecurityHeaders` — 设置 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Content-Security-Policy`。
- `MaxBodyBytes` — 限制请求体大小（API 使用 2 MiB）。
- `RateLimiter` — 按直接连接地址（`RemoteAddr`）的固定窗口限流，清理过期桶并限制缓存规模（最多 10000 键），避免无界内存增长；不信任客户端 `X-Forwarded-For`。
