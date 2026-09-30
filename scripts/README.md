# scripts — 辅助脚本

## `smoke.sh`

端到端冒烟测试脚本，验证 Compose 栈的完整业务闭环（需要 `curl` 与 `jq`）：

1. 检查 `/healthz`、`/readyz` 与管理台静态资源。
2. 创建组织并登录取得访问令牌。
3. 创建仓库、商品、SKU，执行库存入库。
4. 创建订单并完成 `confirm → pay → ship → complete` 全流程。
5. 校验报表、CSV 导出、库存预警规则。
6. 请求密码重置并轮询 Mailpit，确认邮件送达。

运行：

```bash
BASE_URL=http://localhost:8080 MAILPIT_URL=http://localhost:8025 bash scripts/smoke.sh
```

GitHub Actions 在 `compose-smoke` 工作流中自动执行同一脚本。
