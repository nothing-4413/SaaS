CREATE TABLE inventory_documents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    document_type text NOT NULL CHECK (document_type IN ('receipt', 'issue')),
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, document_type, idempotency_key),
    UNIQUE (organization_id, id)
);

CREATE TABLE inventory_document_lines (
    id bigserial PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    document_id uuid NOT NULL,
    warehouse_id uuid NOT NULL,
    sku_id uuid NOT NULL,
    quantity bigint NOT NULL CHECK (quantity > 0),
    FOREIGN KEY (organization_id, document_id) REFERENCES inventory_documents(organization_id, id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id, warehouse_id) REFERENCES warehouses(organization_id, id),
    FOREIGN KEY (organization_id, sku_id) REFERENCES skus(organization_id, id)
);

CREATE INDEX idx_inventory_documents_org_created
    ON inventory_documents (organization_id, document_type, created_at DESC);
CREATE INDEX idx_inventory_document_lines_document
    ON inventory_document_lines (organization_id, document_id);
