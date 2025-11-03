
-- Migration: Initialize Token domain
-- Based on web-app/src/shared/types/token.ts

BEGIN;

CREATE TABLE IF NOT EXISTS tokens (
  mint_address    TEXT        PRIMARY KEY,                      -- TS: Token.mintAddress
  mint_request_id TEXT        NOT NULL,                         -- TS: Token.mintRequestId
  owner           TEXT        NOT NULL,                         -- TS: Token.owner (wallet address)
  minted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_transferred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_tokens_mint_address_non_empty CHECK (char_length(trim(mint_address)) > 0),
  CONSTRAINT chk_tokens_owner_non_empty        CHECK (char_length(trim(owner)) > 0),

  -- Wallet must exist
  CONSTRAINT fk_tokens_owner_wallet
    FOREIGN KEY (owner) REFERENCES wallets(wallet_address) ON DELETE RESTRICT
);

-- Optional FK to mint_requests(id) if the table exists
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'mint_requests'
  ) THEN
    BEGIN
      ALTER TABLE tokens
        ADD CONSTRAINT fk_tokens_mint_request
        FOREIGN KEY (mint_request_id) REFERENCES mint_requests(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;
END$$;

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tokens_owner            ON tokens(owner);
CREATE INDEX IF NOT EXISTS idx_tokens_mint_request_id  ON tokens(mint_request_id);
CREATE INDEX IF NOT EXISTS idx_tokens_minted_at       ON tokens(minted_at);
CREATE INDEX IF NOT EXISTS idx_tokens_last_transferred_at       ON tokens(last_transferred_at);

COMMIT;
