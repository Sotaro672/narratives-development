
-- Migration: Initialize Sale domain
-- Mirrors backend/internal/domain/sale/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS sales (
  id          TEXT  PRIMARY KEY,
  list_id     TEXT  NOT NULL,
  discount_id TEXT,
  prices      JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- Non-empty checks
  CONSTRAINT chk_sales_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(list_id)) > 0
  ),

  -- prices must be a JSON array
  CONSTRAINT chk_sales_prices_array CHECK (jsonb_typeof(prices) = 'array')
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sales_list_id ON sales(list_id);

COMMIT;
