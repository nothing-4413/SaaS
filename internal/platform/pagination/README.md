# internal/platform/pagination — 列表分页

通用列表查询参数与分页切片。

## `Params`

解析 `page`（从 1 开始）、`page_size`（默认 20，最大 100）、`q`（关键词）、`sort`（排序字段）、`order`（`asc`/`desc`）。

- `Parse(values)` — 从 `url.Values` 解析，非法参数返回 `invalid pagination parameters`（`Invalid` 判断）。
- `DefaultPageSize = 20`、`MaxPageSize = 100`。

## `Result[T]` 与 `Slice`

- `Slice(items, p)` 返回 `{items, page, page_size, total}`，`items` 为空时归一化为 `[]` 而非 `null`。
- 切片前先校验页数，避免客户端 `page` 触发整数溢出。

各列表接口（商品、订单、审计日志等）在内存中过滤/排序后，通过 `Slice` 完成分页。
