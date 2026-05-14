CREATE TABLE IF NOT EXISTS admin_refresh_tokens (
    id TEXT PRIMARY KEY,
    admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_admin ON admin_refresh_tokens(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_expires ON admin_refresh_tokens(expires_at);
