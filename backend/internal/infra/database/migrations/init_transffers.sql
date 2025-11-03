
-- Migration: Initialize Transffer domain
-- Mirrors backend/internal/domain/transffer/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS transffers (
  id              TEXT        PRIMARY KEY,
  mint_address    TEXT        NOT NULL,
  from_address    TEXT        NOT NULL,
  to_address      TEXT        NOT NULL,
  requested_at    TIMESTAMPTZ NOT NULL,
  transffered_at  TIMESTAMPTZ,
  status          TEXT        NOT NULL CHECK (status IN ('fulfilled','requested','error')),
  error_type      TEXT,

  -- Non-empty checks
  CONSTRAINT chk_transffers_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(mint_address)) > 0
    AND char_length(trim(from_address)) > 0
    AND char_length(trim(to_address)) > 0
  ),

  -- error_type whitelist (nullable)
  CONSTRAINT chk_transffers_error_type CHECK (
    error_type IS NULL OR error_type IN ('insufficient_balance','invalid_address','network_error','timeout','unknown')
  ),

  -- time order
  CONSTRAINT chk_transffers_time_order CHECK (
    transffered_at IS NULL OR transffered_at >= requested_at
  ),

  -- state coherence
  CONSTRAINT chk_transffers_state_coherence CHECK (
    (status = 'requested' AND transffered_at IS NULL AND error_type IS NULL)
    OR (status = 'fulfilled' AND transffered_at IS NOT NULL AND error_type IS NULL)
    OR (status = 'error'     AND transffered_at IS NULL AND error_type IS NOT NULL)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_transffers_mint_address    ON transffers(mint_address);
CREATE INDEX IF NOT EXISTS idx_transffers_from_address    ON transffers(from_address);
CREATE INDEX IF NOT EXISTS idx_transffers_to_address      ON transffers(to_address);
CREATE INDEX IF NOT EXISTS idx_transffers_status          ON transffers(status);
CREATE INDEX IF NOT EXISTS idx_transffers_requested_at    ON transffers(requested_at);
CREATE INDEX IF NOT EXISTS idx_transffers_transffered_at  ON transffers(transffered_at);

COMMIT;
