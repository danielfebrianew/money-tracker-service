ALTER TABLE accounts
    DROP COLUMN IF EXISTS icon,
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_type_check;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_type_check CHECK (type IN ('bank', 'ewallet', 'cash'));
