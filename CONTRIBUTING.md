# Contributing

1. Fork 或创建功能分支。
2. 保持模块边界清晰，新增业务逻辑优先放入对应 `internal/<module>`。
3. 为并发、幂等、租户隔离和状态流转补充测试。
4. 提交前执行 `gofmt`、`go test ./...` 和 `go build ./...`。
5. Pull Request 中说明数据迁移、API 变更和兼容性影响。

提交信息建议使用 Conventional Commits，例如 `feat: add stock warning`、`fix: prevent duplicate reservation`。
