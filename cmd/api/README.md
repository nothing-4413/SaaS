# cmd/api

API 服务入口：加载配置、连接 PostgreSQL，按 URL 前缀路由到各领域 Handler，叠加权限/审计中间件、安全头、限流与指标，支持优雅退出。

运行：`go run ./cmd/api`
