ALTER TABLE users ADD COLUMN active boolean NOT NULL DEFAULT true;

CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    FOREIGN KEY (organization_id, user_id) REFERENCES users(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_auth_sessions_user_active
    ON auth_sessions (organization_id, user_id, expires_at) WHERE revoked_at IS NULL;
