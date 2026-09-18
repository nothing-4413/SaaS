CREATE TABLE webhook_subscriptions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url text NOT NULL,
    secret text NOT NULL,
    event_types jsonb NOT NULL DEFAULT '[]'::jsonb,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, url)
);

CREATE INDEX idx_webhook_subscriptions_org_active
    ON webhook_subscriptions (organization_id, active);
