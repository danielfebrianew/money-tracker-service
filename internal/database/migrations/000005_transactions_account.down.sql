ALTER TABLE transactions DROP COLUMN IF EXISTS account_id;

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_tipe_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_tipe_check CHECK (tipe IN ('IN', 'OUT'));

ALTER TABLE transactions ALTER COLUMN tipe TYPE VARCHAR(3);
