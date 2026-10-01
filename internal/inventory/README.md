# internal/inventory

库存与单据：receive/reserve/release/deduct/consume_reserved 五种幂等操作，入库/出库单；生产用 `FOR UPDATE` 行锁保证并发安全。
