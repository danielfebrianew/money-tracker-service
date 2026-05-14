ALTER TABLE transactions ALTER COLUMN tipe TYPE VARCHAR(10);

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_tipe_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_tipe_check CHECK (tipe IN ('IN', 'OUT', 'TRANSFER'));

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS account_id VARCHAR(36) REFERENCES accounts(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions(account_id);
