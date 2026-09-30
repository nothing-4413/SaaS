# internal/product — 商品目录

商品、SKU 与仓库的目录模块，提供创建、查询与业务维护能力。

## 模型

- `Product` — 商品，含名称与描述。
- `SKU` — 商品下的最小销售单元，含编码、名称与售价（`PriceCents`，分为单位）。
- `Warehouse` — 仓库，含名称与地址。

所有资源都携带 `OrganizationID`，组织内编码/名称唯一性由数据库约束保证（`UNIQUE (organization_id, code)` 等）。

## 服务（`Service`）

- `CreateProduct` / `UpdateProduct`
- `CreateSKU` / `UpdateSKU`（校验 `PriceCents >= 0`）
- `CreateWarehouse` / `UpdateWarehouse`
- `ListProducts` / `ListSKUs` / `ListWarehouses`

所有写操作都显式校验组织归属，跨租户访问返回错误。

## HTTP 接口

- `POST/GET /organizations/{id}/products`
- `PUT /organizations/{id}/products/{productId}`
- `POST/GET /organizations/{id}/products/{productId}/skus`
- `PUT /organizations/{id}/products/{productId}/skus/{skuId}`
- `POST/GET /organizations/{id}/warehouses`
- `PUT /organizations/{id}/warehouses/{warehouseId}`

列表接口支持 `q`（关键词）、`sort`、`order`（asc/desc）与分页参数，见 [`internal/platform/pagination`](../platform/pagination)。

`GetSKU` 被订单模块用于权威定价，确保订单单价取自组织内 SKU 当前售价。
