ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS icon       VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color      VARCHAR(7)  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_type_check;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_type_check CHECK (type IN ('bank', 'ewallet', 'cash', 'credit_card'));
