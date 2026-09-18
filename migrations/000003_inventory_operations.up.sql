CREATE TABLE inventory_operations (
    id bigserial PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    warehouse_id uuid NOT NULL,
    sku_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('receive', 'reserve', 'release', 'deduct', 'consume_reserved')),
    quantity bigint NOT NULL CHECK (quantity > 0),
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, idempotency_key),
    FOREIGN KEY (organization_id, warehouse_id) REFERENCES warehouses(organization_id, id),
    FOREIGN KEY (organization_id, sku_id) REFERENCES skus(organization_id, id)
);

CREATE INDEX idx_inventory_operations_resource
    ON inventory_operations (organization_id, warehouse_id, sku_id, created_at DESC);
