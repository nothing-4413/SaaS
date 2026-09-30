# internal/importer — 库存 CSV 导入

与 [`internal/export`](../export) 同格式的库存 CSV 导入模块。

## 解析（`Stocks`）

- 校验表头与列：`organization_id, warehouse_id, sku_id, on_hand, reserved, available`。
- 逐行校验：
  - `organization_id` 必须等于目标组织；
  - 数量非负，且 `reserved <= on_hand`、`available = on_hand - reserved`；
  - 仓库/SKU 字段非空。

## 落库（`inventory.Service.ImportStocks`）

- 再次校验组织归属、重复库存项、非负数量与派生关系。
- 通过 `ImportStore` 在单个 PostgreSQL 事务内批量写入，任一仓库/SKU 外键或数据非法时整批回滚。

## HTTP 接口

- `POST /organizations/{id}/imports/stocks`（要求 `inventory:import`），请求体为库存 CSV，先完成整批校验和原子落库，再返回导入结果 `{"imported": N, "stocks": [...]}`。
