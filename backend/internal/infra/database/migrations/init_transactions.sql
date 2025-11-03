
-- Migration: Initialize Transaction domain
-- Mirrors backend/internal/domain/transaction/entity.go and web-app/src/shared/types/transaction.ts

BEGIN;

CREATE TABLE IF NOT EXISTS transactions (
  id            TEXT        PRIMARY KEY,
  account_id    TEXT        NOT NULL,
  brand_name    TEXT        NOT NULL,
  type          TEXT        NOT NULL CHECK (type IN ('receive','send')),
  amount        BIGINT      NOT NULL CHECK (amount >= 0),
  currency      TEXT        NOT NULL,
  from_account  TEXT        NOT NULL,
  to_account    TEXT        NOT NULL,
  timestamp     TIMESTAMPTZ NOT NULL,
  description   TEXT        NOT NULL DEFAULT '',

  -- Non-empty checks
  CONSTRAINT chk_tx_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(account_id)) > 0
    AND char_length(trim(brand_name)) > 0
    AND char_length(trim(currency)) > 0
    AND char_length(trim(from_account)) > 0
    AND char_length(trim(to_account)) > 0
  ),

  -- currency format (ISO 4217-like: 3 uppercase letters)
  CONSTRAINT chk_tx_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tx_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_tx_brand_name ON transactions(brand_name);
CREATE INDEX IF NOT EXISTS idx_tx_type       ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_tx_currency   ON transactions(currency);
CREATE INDEX IF NOT EXISTS idx_tx_timestamp  ON transactions(timestamp);

COMMIT;
