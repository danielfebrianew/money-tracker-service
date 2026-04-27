CREATE TABLE IF NOT EXISTS transactions (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id    VARCHAR(36) REFERENCES budget_groups(id) ON DELETE SET NULL,
    jumlah      INTEGER NOT NULL,
    deskripsi   VARCHAR(255) NOT NULL,
    kategori    VARCHAR(50) NOT NULL,
    tipe        VARCHAR(3) NOT NULL CHECK (tipe IN ('IN', 'OUT')),
    source      VARCHAR(20) NOT NULL DEFAULT 'whatsapp',
    recorded_by VARCHAR(20),
    confidence  DECIMAL(3,2),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    IF to_regclass('public.sheet_transactions') IS NOT NULL THEN
        INSERT INTO transactions (id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, created_at)
        SELECT st.id,
               CASE
                   WHEN st.owner_type = 'user' THEN st.owner_id
                   ELSE bg.owner_id
               END AS user_id,
               CASE WHEN st.owner_type = 'group' THEN st.owner_id ELSE NULL END AS group_id,
               st.jumlah,
               LEFT(st.deskripsi, 255),
               LEFT(st.kategori, 50),
               st.tipe,
               st.source,
               st.recorded_by,
               st.tanggal
        FROM sheet_transactions st
        LEFT JOIN budget_groups bg ON bg.id = st.owner_id AND st.owner_type = 'group'
        WHERE EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = CASE WHEN st.owner_type = 'user' THEN st.owner_id ELSE bg.owner_id END
        )
        ON CONFLICT (id) DO NOTHING;

        DROP TABLE sheet_transactions;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_created ON transactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_tipe ON transactions(user_id, tipe);
CREATE INDEX IF NOT EXISTS idx_transactions_user_kategori ON transactions(user_id, kategori);
CREATE INDEX IF NOT EXISTS idx_transactions_group ON transactions(group_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at DESC);

ALTER TABLE users DROP COLUMN IF EXISTS sheets_id;
ALTER TABLE users DROP COLUMN IF EXISTS setup_complete;
ALTER TABLE budget_groups DROP COLUMN IF EXISTS sheets_id;
