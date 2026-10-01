# internal/platform/sqltx

`WithTx` 事务封装（出错/panic 回滚），`LockStock` 用 `FOR UPDATE` 锁定库存行。
