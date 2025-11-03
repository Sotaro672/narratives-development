
-- Migration: Initialize payments table (mirrors domain/payment/entity.go)

BEGIN;

CREATE TABLE IF NOT EXISTS payments (
  id                 TEXT        PRIMARY KEY,
  invoice_id         TEXT        NOT NULL,
  billing_address_id TEXT        NOT NULL,
  amount             INTEGER     NOT NULL,
  status             TEXT        NOT NULL,
  error_type         TEXT        NULL,
  created_at         TIMESTAMPTZ NOT NULL,
  updated_at         TIMESTAMPTZ NOT NULL,
  deleted_at         TIMESTAMPTZ NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_payments_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(invoice_id)) > 0
    AND char_length(trim(billing_address_id)) > 0
  ),

  -- Amount policy (MinAmount = 0)
  CONSTRAINT chk_payments_amount CHECK (amount >= 0),

  -- Status must be non-empty (enum is open in domain)
  CONSTRAINT chk_payments_status_non_empty CHECK (char_length(trim(status)) > 0),

  -- error_type optional but if present must be non-empty
  CONSTRAINT chk_payments_error_type_non_empty CHECK (
    error_type IS NULL OR char_length(trim(error_type)) > 0
  ),

  -- Time order coherence
  CONSTRAINT chk_payments_time_order CHECK (
    updated_at >= created_at
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id   ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_status       ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_billing_id   ON payments(billing_address_id);
CREATE INDEX IF NOT EXISTS idx_payments_amount       ON payments(amount);
CREATE INDEX IF NOT EXISTS idx_payments_created_at   ON payments(created_at);
CREATE INDEX IF NOT EXISTS idx_payments_updated_at   ON payments(updated_at);
CREATE INDEX IF NOT EXISTS idx_payments_deleted_at   ON payments(deleted_at);

COMMIT;
