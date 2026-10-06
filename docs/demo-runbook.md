# 面试演示与验收手册

这个项目的目的是**面试演示，不是生产系统**：重点是把业务闭环真的跑起来、把设计取舍讲清楚。

## 1. 五分钟演示路径（建议顺序）

| 时间 | 做什么 | 讲什么 |
| --- | --- | --- |
| 0:00 | 打开 `http://localhost:8080/` 登录，或直接展示 Overview 截图 | 一个 workspace 里同时有 Catalog / Inventory / Orders / 低库存策略 |
| 0:30 | Catalog：建仓库、商品、SKU | 组织内编码与名称唯一性由数据库约束保证 |
| 1:30 | Inventory：入库 10 件，看 Available 变化 | 幂等键（`idempotency_key`）重复提交不会重复记账 |
| 2:00 | Orders：建单 2 件 | 创建时**预占**：reserved +2、available −2；单价取服务端 SKU 售价，不信任客户端 |
| 3:00 | confirm → pay → ship → complete | 状态机严格单向；confirm 才真正扣减已预占库存 |
| 4:00 | Overview 看报表，下载库存 CSV | 报表与导出都按组织隔离 |
| 4:30 | 低库存策略设阈值，观察 Worker 产出 `stock.low` 事件 | Outbox 模式：事件与业务同事务落库，Worker 带租约/退避/终态失败 |

`scripts/smoke.ps1`（Windows）或 `scripts/smoke.sh`（Linux）会把上面这条链路自动跑一遍，可以作为"确实跑通"的证据。

## 2. 启动方式

Docker 可用时：

```bash
docker compose up --build
```

本机 Docker/WSL 起不来时（Windows，便携 PostgreSQL）：

```powershell
# 便携 PostgreSQL + 迁移 + API（Ctrl+C 停 API，-Action down 停数据库）
powershell -File scripts\dev.ps1 -PgRoot <解压目录>\pgsql -DataDir <数据目录>
# 端到端业务冒烟（不需要 bash/jq）
powershell -File scripts\smoke.ps1
```

管理台 `http://localhost:8080/`；开发 Compose 的邮件进 Mailpit `http://localhost:8025/`。

> 若 `127.0.0.1:8080` 已被别的程序占用（例如 Steam 的 `steamwebhelper`），请用 `localhost` 访问，或用 `scripts\dev.ps1 -HttpAddr :8090`。Docker 起不来时先查 WSL：`wsl -l -v` 若长时间不返回，用管理员 PowerShell 执行 `Restart-Service WSLService -Force`（或重启机器）。

## 3. 自动验收

- `scripts/smoke.ps1` / `scripts/smoke.sh`：建租户、登录、目录、入库、订单全生命周期、报表、CSV、低库存策略、密码重置（Mailpit 可达时校验邮件真的投递）。
- `go test ./...`：内存 Store 的单元测试。
- `TEST_DATABASE_URL=... go test -tags=integration ./...`：真实 PostgreSQL 上的并发扣减/预占、幂等重放、出入库单原子性、Outbox 领取与租约。
- CI（[.github/workflows/ci.yml](../.github/workflows/ci.yml)）每次提交都会跑这三类检查。

## 4. 面试重点（被追问时的答案）

- **多租户隔离**：每个资源都携带 organization ID；Token 租户与 URL 租户不一致会被直接拒绝。
- **库存一致性**：PostgreSQL 用 `SELECT ... FOR UPDATE` 行锁，配合 `CHECK (reserved <= on_hand)` 与组织范围幂等唯一约束；订单创建失败会补偿释放之前成功的预占。
- **为什么有两套 Store**：内存实现让单元测试快且无外部依赖，PostgreSQL 实现是真实运行路径——代价是 SQL 层必须靠 `-tags=integration` 的测试覆盖（这正是后来补集成测试的原因）。
- **可靠通知**：Outbox 事件与业务写在同一事务；Worker 用 `FOR UPDATE SKIP LOCKED` 领取，带五分钟租约、递增退避、达到次数进入终态 `failed`；Webhook 用 HMAC-SHA256 签名并携带幂等键。
- **踩过的坑（可以主动讲）**：安全中间件曾对所有响应下发 `Content-Security-Policy: default-src 'none'`，浏览器把 `/app.js` 和 `/styles.css` 全拦掉，登录退化成原生 GET 表单（密码进了 URL）；而 CI 的 curl 冒烟不看 CSP，所以一直是绿的。后来用无头浏览器截图时才发现，改成"只允许同源资源 + 禁内联脚本"，并补了回归断言。

## 5. 有意简化 / 未覆盖（如实说）

- 限流是单进程内存实现，不保证多实例下的全局配额。
- 没有压测、长稳测试，也没有 Kubernetes、计费、SSO。
- 管理台没有浏览器端自动化测试（截图与登录流程是人工 + 无头浏览器验证的）。

## 6. 如果真上线，还缺什么

- 用 `.env.production.example` 派生密钥，不要复用 Compose 里的开发密码。
- `DATABASE_URL` 指向启用 TLS 的托管 PostgreSQL，并配置备份、恢复演练与连接池上限。
- `APP_PUBLIC_URL` 走 HTTPS；SMTP 用 587/STARTTLS 或云邮件服务。
- 将 `/healthz`、`/readyz`、`/metrics`（含 Worker `:9090/metrics`）接入监控与告警。
- 迁移作为独立发布步骤执行，API/Worker 只在迁移成功后启动。
