# internal/platform/sqlctx — 数据库操作上下文

`Context()` 返回带 5 秒超时的 `context.Context`，并用 `time.AfterFunc` 在到期时调用 cancel，避免取消函数被丢弃。

用于约束单个数据库操作或事务，使阻塞的数据库不会无限期拖住 HTTP 请求或 Worker 循环。
