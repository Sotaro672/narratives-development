
-- Migration: Initialize Fulfillment domain
-- Mirrors backend/internal/domain/fulfillment/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS fulfillments (
  id          UUID        PRIMARY KEY,
  order_id    TEXT        NOT NULL,
  payment_id  TEXT        NOT NULL,
  status      TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_fulfillments_non_empty CHECK (
    char_length(trim(order_id)) > 0 AND
    char_length(trim(payment_id)) > 0 AND
    char_length(trim(status)) > 0
  ),

  -- time order
  CONSTRAINT chk_fulfillments_time_order CHECK (updated_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_fulfillments_order_id   ON fulfillments(order_id);
CREATE INDEX IF NOT EXISTS idx_fulfillments_payment_id ON fulfillments(payment_id);
CREATE INDEX IF NOT EXISTS idx_fulfillments_status     ON fulfillments(status);
CREATE INDEX IF NOT EXISTS idx_fulfillments_created_at ON fulfillments(created_at);

COMMIT;
