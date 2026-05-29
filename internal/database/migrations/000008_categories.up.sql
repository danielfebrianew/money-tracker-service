CREATE TABLE IF NOT EXISTS categories (
    id         VARCHAR(36) PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(50) NOT NULL,
    label      VARCHAR(100) NOT NULL,
    icon       VARCHAR(50) NOT NULL DEFAULT '🏷️',
    color      VARCHAR(7)  NOT NULL DEFAULT '#607D8B',
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_categories_user ON categories(user_id);

ALTER TABLE budgets DROP COLUMN IF EXISTS kategori_label;
