ALTER TABLE wallets RENAME TO accounts;
ALTER INDEX IF EXISTS idx_wallets_user RENAME TO idx_accounts_user;
ALTER TABLE transactions RENAME COLUMN wallet_id TO account_id;
ALTER INDEX IF EXISTS idx_transactions_wallet RENAME TO idx_transactions_account;
