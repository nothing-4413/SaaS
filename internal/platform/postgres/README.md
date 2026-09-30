# internal/platform/postgres — PostgreSQL 连接

`Open(ctx, url)` 使用 `jackc/pgx/v5` 标准库驱动打开 PostgreSQL 连接池：

- 设置连接池上限（20 最大、5 空闲）、连接生命周期（30 分钟）与空闲时间（5 分钟）。
- 打开后在 5 秒超时内 `PingContext` 校验连通性，失败时关闭并返回错误。

由 `cmd/api` 与 `cmd/worker` 在启动时调用，连接串来自 `DATABASE_URL`。
