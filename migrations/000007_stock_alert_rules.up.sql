CREATE TABLE stock_alert_rules (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    threshold bigint NOT NULL CHECK (threshold >= 0),
    enabled boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_alert_rules_enabled
    ON stock_alert_rules (enabled) WHERE enabled = true;
