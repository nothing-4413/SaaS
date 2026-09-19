CREATE TABLE webhook_deliveries (
    event_id uuid NOT NULL REFERENCES outbox_events(id) ON DELETE CASCADE,
    subscription_id uuid NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    delivered_at timestamptz NOT NULL,
    PRIMARY KEY (event_id, subscription_id)
);

CREATE INDEX idx_webhook_deliveries_subscription
    ON webhook_deliveries (subscription_id, delivered_at DESC);
