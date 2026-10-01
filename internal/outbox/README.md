# internal/outbox

事务性事件队列（Outbox 模式）：幂等入队、`FOR UPDATE SKIP LOCKED` 领取、五分钟租约恢复与递增退避重试，终态 `failed`。
