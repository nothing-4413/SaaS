-- Tenant and authorization boundary
CREATE TABLE organizations (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (organization_id, name),
    UNIQUE (organization_id, id)
);

CREATE TABLE users (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email text NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, email),
    UNIQUE (organization_id, id)
);

CREATE TABLE user_roles (
	organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (organization_id, user_id, role_id),
    FOREIGN KEY (organization_id, user_id) REFERENCES users(organization_id, id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id, role_id) REFERENCES roles(organization_id, id) ON DELETE CASCADE
);

-- Product catalog and warehouses
CREATE TABLE products (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, id)
);

CREATE TABLE skus (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    code text NOT NULL,
    name text NOT NULL,
    price_cents bigint NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    UNIQUE (organization_id, code),
    UNIQUE (organization_id, id),
    FOREIGN KEY (organization_id, product_id) REFERENCES products(organization_id, id)
);

CREATE TABLE warehouses (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    address text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name),
    UNIQUE (organization_id, id)
);

-- Inventory ledger. available is derived and maintained transactionally.
CREATE TABLE inventory_stocks (
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    warehouse_id uuid NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    sku_id uuid NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    on_hand bigint NOT NULL DEFAULT 0 CHECK (on_hand >= 0),
    reserved bigint NOT NULL DEFAULT 0 CHECK (reserved >= 0 AND reserved <= on_hand),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, warehouse_id, sku_id),
    FOREIGN KEY (organization_id, warehouse_id) REFERENCES warehouses(organization_id, id),
    FOREIGN KEY (organization_id, sku_id) REFERENCES skus(organization_id, id)
);

-- Sales orders and immutable order lines.
CREATE TYPE order_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE orders (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    idempotency_key text NOT NULL,
    status order_status NOT NULL DEFAULT 'pending',
    total_cents bigint NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, idempotency_key),
    UNIQUE (organization_id, id)
);

CREATE TABLE order_lines (
    id bigserial PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    warehouse_id uuid NOT NULL REFERENCES warehouses(id),
    sku_id uuid NOT NULL REFERENCES skus(id),
    FOREIGN KEY (organization_id, order_id) REFERENCES orders(organization_id, id),
    FOREIGN KEY (organization_id, warehouse_id) REFERENCES warehouses(organization_id, id),
    FOREIGN KEY (organization_id, sku_id) REFERENCES skus(organization_id, id),
    quantity bigint NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint NOT NULL CHECK (unit_price_cents >= 0)
);

-- Transactional event publication queue.
CREATE TYPE outbox_status AS ENUM ('pending', 'processing', 'published', 'failed');

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    event_type text NOT NULL,
    dedup_key text NOT NULL,
    payload jsonb NOT NULL,
    status outbox_status NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    claimed_until timestamptz,
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    UNIQUE (organization_id, aggregate_type, aggregate_id, event_type, dedup_key)
);

CREATE TABLE audit_logs (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_org ON users (organization_id);
CREATE INDEX idx_products_org ON products (organization_id);
CREATE INDEX idx_skus_product ON skus (organization_id, product_id);
CREATE INDEX idx_warehouses_org ON warehouses (organization_id);
CREATE INDEX idx_orders_org_created ON orders (organization_id, created_at DESC);
CREATE INDEX idx_order_lines_order ON order_lines (organization_id, order_id);
CREATE INDEX idx_outbox_pending ON outbox_events (status, next_attempt_at) WHERE status = 'pending';
CREATE INDEX idx_audit_org_created ON audit_logs (organization_id, created_at DESC);
