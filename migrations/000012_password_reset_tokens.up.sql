CREATE TABLE password_reset_tokens (
    token_hash text PRIMARY KEY,
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    used_at timestamptz,
    FOREIGN KEY (organization_id, user_id) REFERENCES users(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_password_reset_tokens_expiry
    ON password_reset_tokens (organization_id, user_id, expires_at)
    WHERE used_at IS NULL;
