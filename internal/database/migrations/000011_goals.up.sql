CREATE TABLE IF NOT EXISTS goals (
    id           VARCHAR(36) PRIMARY KEY,
    user_id      VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    target_amount INT NOT NULL CHECK (target_amount > 0),
    current_amount INT NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    deadline     DATE NOT NULL,
    icon         VARCHAR(100) NOT NULL DEFAULT 'target',
    color        VARCHAR(20) NOT NULL DEFAULT '#6366F1',
    status       VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'achieved')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goals_user ON goals(user_id);
