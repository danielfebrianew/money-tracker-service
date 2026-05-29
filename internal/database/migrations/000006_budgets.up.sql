CREATE TABLE IF NOT EXISTS budgets (
    id             VARCHAR(36) PRIMARY KEY,
    user_id        VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kategori       VARCHAR(50) NOT NULL,
    kategori_label VARCHAR(100) NOT NULL,
    "limit"        INTEGER NOT NULL CHECK ("limit" > 0),
    month          CHAR(7) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kategori, month)
);

CREATE INDEX IF NOT EXISTS idx_budgets_user_month ON budgets(user_id, month);
