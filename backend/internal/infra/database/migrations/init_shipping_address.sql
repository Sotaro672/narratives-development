
-- Migration: Initialize ShippingAddress domain
-- Mirrors backend/internal/domain/shippingAddress/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS shipping_addresses (
  id              TEXT        PRIMARY KEY,
  user_id         TEXT        NOT NULL,
  street          TEXT        NOT NULL,
  city            TEXT        NOT NULL,
  state           TEXT        NOT NULL,
  zip_code        TEXT        NOT NULL,
  country         TEXT        NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_shipping_addresses_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(user_id)) > 0
    AND char_length(trim(street)) > 0
    AND char_length(trim(city)) > 0
    AND char_length(trim(state)) > 0
    AND char_length(trim(zip_code)) > 0
    AND char_length(trim(country)) > 0
  ),

  -- time order
  CONSTRAINT chk_shipping_addresses_time_order CHECK (updated_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_shipping_addresses_user_id     ON shipping_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_shipping_addresses_updated_at  ON shipping_addresses(updated_at);

COMMIT;
