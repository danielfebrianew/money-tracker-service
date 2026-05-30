ALTER TABLE accounts RENAME TO wallets;
ALTER INDEX IF EXISTS idx_accounts_user RENAME TO idx_wallets_user;
ALTER TABLE transactions RENAME COLUMN account_id TO wallet_id;
ALTER INDEX IF EXISTS idx_transactions_account RENAME TO idx_transactions_wallet;
