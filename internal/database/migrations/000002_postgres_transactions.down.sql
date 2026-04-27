CREATE TABLE IF NOT EXISTS sheet_transactions (
    id TEXT PRIMARY KEY,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('user', 'group')),
    owner_id TEXT NOT NULL,
    sheets_id TEXT NOT NULL DEFAULT '',
    tanggal TIMESTAMPTZ NOT NULL,
    jumlah INTEGER NOT NULL CHECK (jumlah > 0),
    deskripsi TEXT NOT NULL,
    kategori TEXT NOT NULL,
    tipe TEXT NOT NULL CHECK (tipe IN ('IN', 'OUT')),
    source TEXT NOT NULL,
    recorded_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS sheets_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS setup_complete BOOLEAN DEFAULT FALSE;
ALTER TABLE budget_groups ADD COLUMN IF NOT EXISTS sheets_id VARCHAR(255);

DROP TABLE IF EXISTS transactions;
