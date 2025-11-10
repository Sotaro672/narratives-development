
-- Migration: Initialize Transfer domain
-- Mirrors backend/internal/domain/transfer/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS transfers (
  id              TEXT        PRIMARY KEY,
  mint_address    TEXT        NOT NULL,
  from_address    TEXT        NOT NULL,
  to_address      TEXT        NOT NULL,
  requested_at    TIMESTAMPTZ NOT NULL,
  transferred_at  TIMESTAMPTZ,
  status          TEXT        NOT NULL CHECK (status IN ('fulfilled','requested','error')),
  error_type      TEXT,

  -- Non-empty checks
  CONSTRAINT chk_transfers_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(mint_address)) > 0
    AND char_length(trim(from_address)) > 0
    AND char_length(trim(to_address)) > 0
  ),

  -- error_type whitelist (nullable)
  CONSTRAINT chk_transfers_error_type CHECK (
    error_type IS NULL OR error_type IN ('insufficient_balance','invalid_address','network_error','timeout','unknown')
  ),

  -- time order
  CONSTRAINT chk_transfers_time_order CHECK (
    transferred_at IS NULL OR transferred_at >= requested_at
  ),

  -- state coherence
  CONSTRAINT chk_transfers_state_coherence CHECK (
    (status = 'requested' AND transferred_at IS NULL AND error_type IS NULL)
    OR (status = 'fulfilled' AND transferred_at IS NOT NULL AND error_type IS NULL)
    OR (status = 'error'     AND transferred_at IS NULL AND error_type IS NOT NULL)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_transfers_mint_address    ON transfers(mint_address);
CREATE INDEX IF NOT EXISTS idx_transfers_from_address    ON transfers(from_address);
CREATE INDEX IF NOT EXISTS idx_transfers_to_address      ON transfers(to_address);
CREATE INDEX IF NOT EXISTS idx_transfers_status          ON transfers(status);
CREATE INDEX IF NOT EXISTS idx_transfers_requested_at    ON transfers(requested_at);
CREATE INDEX IF NOT EXISTS idx_transfers_transferred_at  ON transfers(transferred_at);

COMMIT;
