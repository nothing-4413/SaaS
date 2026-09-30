# internal/platform/idgen — UUID 生成

`New()` 使用 `crypto/rand` 生成 RFC 4122 version 4 UUID 字符串。

所有业务资源的 ID（组织、用户、角色、商品、SKU、仓库、订单、事件、单据等）都由它生成，避免自增 ID 泄露规模，也便于分布式环境无冲突。
