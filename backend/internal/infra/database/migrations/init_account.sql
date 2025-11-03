
-- Accounts DDL generated from domain/account entity.
CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  member_id VARCHAR(100) NOT NULL,
  account_holder_name VARCHAR(100) GENERATED ALWAYS AS (member_id) STORED,
  bank_name VARCHAR(50) NOT NULL,
  branch_name VARCHAR(50) NOT NULL,
  account_number INTEGER NOT NULL CHECK (account_number >= 0 AND account_number <= 99999999),
  account_type TEXT NOT NULL CHECK (account_type IN ('普通','当座')),
  currency TEXT NOT NULL DEFAULT '円',
  status TEXT NOT NULL CHECK (status IN ('active','inactive','suspended','deleted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by TEXT NULL,
  CHECK (updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at),
  CHECK (id LIKE 'account_%')
);

CREATE INDEX IF NOT EXISTS idx_accounts_member_id ON accounts(member_id);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_account_number ON accounts(account_number);
CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at);
