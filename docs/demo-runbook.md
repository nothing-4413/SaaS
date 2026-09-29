# 面试演示与验收手册

## 一键启动

需要 Docker Desktop 和 Compose：

```bash
docker compose up --build
```

打开 `http://localhost:8080/` 进入管理台。开发环境邮件会进入 Mailpit：`http://localhost:8025/`。

自动验收：在服务健康后运行 `bash scripts/smoke.sh`（需要 curl、jq），检查建租户、库存预占与扣减、订单流转、报表、CSV 和 Mailpit 密码重置邮件。CI 会自动执行相同脚本；本地 Docker 无法启动时，可先运行 `go test ./...`、`go vet ./...`、`go build ./...` 验证 Go 代码。

## 业务闭环

1. 在管理台创建 workspace，记录返回的 organization ID，然后登录。
2. 在 Catalog 中创建仓库和商品，再添加一个 SKU。
3. 在 Inventory 中录入库存，观察 Overview 的可用量和低库存指标。
4. 在 Orders 中创建订单。创建事务会按仓库/SKU 预占库存，Inventory 表中的 reserved 增加。
5. 依次执行 confirm、pay、ship、complete，观察订单状态、收入和库存变化。
6. 将低库存阈值设为较高值，等待 Worker 扫描；`stock.low` 事件进入 Outbox，可由 Webhook 消费。
7. 用登录页密码重置功能验证事件链路；开启 SMTP 后邮件会由 Worker 投递，开发 Compose 中可在 Mailpit 查看。

## 面试重点

- **多租户隔离**：每个资源都携带 organization ID，Token 租户与 URL 租户不一致会被拒绝。
- **库存一致性**：PostgreSQL 使用行锁，业务幂等键和数据库唯一约束防止重复记账；订单创建失败有补偿路径。
- **可靠通知**：Outbox 事件先落库，Worker 使用租约、退避和终态失败；Webhook 订阅按事件去重。
- **可运营性**：API 和 Worker 都有健康检查、Prometheus 指标、审计日志和结构化的业务流程。

## 生产上线检查

- 使用 `.env.production.example` 派生密钥，禁止使用 Compose 中的开发数据库密码。
- 将 `DATABASE_URL` 指向启用 TLS 的托管 PostgreSQL，并配置备份、恢复演练和连接池上限。
- 将 `APP_PUBLIC_URL` 设置为 HTTPS 地址；SMTP 使用 587/STARTTLS 或云邮件服务。
- 将 `/healthz`、`/readyz`、`/metrics` 和 Worker `:9090/metrics` 接入监控与告警。
- 迁移由独立发布步骤执行，API/Worker 只在迁移成功后启动。
