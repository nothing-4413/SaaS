# internal/adminui — 内嵌管理台

把演示用管理台（StockPilot）以 `//go:embed static/*` 方式嵌入 API 二进制，使产品可用单一服务端点启动。

## 文件

- `static/index.html`、`static/app.js`、`static/styles.css` — 前端资源。
- `ui.go` — 从嵌入文件系统提供静态资源。

## 行为

- `Handler()` 服务 `/`、`/reset-password`、`/app.js`、`/styles.css`，其余路径返回 404。
- `index.html` 使用 `Cache-Control: no-store`，静态资源缓存 1 小时。
- 由 `cmd/api` 在 URL 为空、`reset-password` 或静态资源路径时接管。

启动后访问 `http://localhost:8080/` 即可完成登录、库存、订单和目录操作。
