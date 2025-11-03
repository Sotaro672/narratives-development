
-- Migration: Initialize Tracking domain
-- Mirrors backend/internal/domain/tracking/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS trackings (
  id                     TEXT        PRIMARY KEY,
  order_id               TEXT        NOT NULL,
  tracking_number        TEXT        NOT NULL,
  carrier                TEXT        NOT NULL,
  special_instructions   TEXT,
  created_at             TIMESTAMPTZ NOT NULL,
  updated_at             TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_trackings_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(order_id)) > 0
    AND char_length(trim(tracking_number)) > 0
    AND char_length(trim(carrier)) > 0
  ),

  -- tracking_number format (matches ^[A-Za-z0-9\-_.]+$)
  CONSTRAINT chk_trackings_tracking_number_format CHECK (tracking_number ~ '^[A-Za-z0-9\\-_.]+$'),

  -- time order
  CHECK (updated_at >= created_at)
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_trackings_order_id         ON trackings(order_id);
CREATE INDEX IF NOT EXISTS idx_trackings_created_at       ON trackings(created_at);
CREATE INDEX IF NOT EXISTS idx_trackings_updated_at       ON trackings(updated_at);
CREATE INDEX IF NOT EXISTS idx_trackings_tracking_number  ON trackings(tracking_number);
CREATE INDEX IF NOT EXISTS idx_trackings_carrier          ON trackings(carrier);

COMMIT;
