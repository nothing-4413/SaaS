# internal/platform — 平台基础包

跨领域复用的基础设施包，供各业务模块调用：

- [`idgen`](idgen/README.md) — 安全随机 UUID 生成
- [`pagination`](pagination/README.md) — 列表查询参数解析与分页切片
- [`postgres`](postgres/README.md) — PostgreSQL 连接池初始化
- [`sqlctx`](sqlctx/README.md) — 有超时的数据库操作上下文
- [`sqltx`](sqltx/README.md) — 事务封装与库存行锁查询

这些包不依赖任何业务领域，保持与上层模块解耦。
