
-- Migration: Initialize BillingAddress domain
-- Mirrors backend/internal/domain/billingAddress/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS billing_addresses (
  id               UUID        PRIMARY KEY,
  user_id          TEXT        NOT NULL,
  name_on_account  TEXT,
  billing_type     TEXT        NOT NULL,
  card_brand       TEXT,
  card_last4       TEXT,
  card_exp_month   INTEGER,
  card_exp_year    INTEGER,
  card_token       TEXT,
  postal_code      INTEGER,
  state            TEXT,
  city             TEXT,
  street           TEXT,
  country          TEXT,
  is_default       BOOLEAN     NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL,
  updated_at       TIMESTAMPTZ NOT NULL,

  -- Basic checks
  CONSTRAINT chk_ba_billing_type_non_empty CHECK (char_length(trim(billing_type)) > 0),

  -- card_last4 must be 4 digits if provided
  CONSTRAINT chk_ba_card_last4 CHECK (card_last4 IS NULL OR card_last4 ~ '^[0-9]{4}$'),

  -- month/year ranges if provided
  CONSTRAINT chk_ba_card_month CHECK (card_exp_month IS NULL OR (card_exp_month BETWEEN 1 AND 12)),
  CONSTRAINT chk_ba_card_year  CHECK (card_exp_year  IS NULL OR (card_exp_year BETWEEN 2000 AND 2100)),

  -- time order
  CONSTRAINT chk_ba_time_order CHECK (updated_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_ba_user_id     ON billing_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_ba_is_default  ON billing_addresses(is_default);
CREATE INDEX IF NOT EXISTS idx_ba_created_at  ON billing_addresses(created_at);
CREATE INDEX IF NOT EXISTS idx_ba_updated_at  ON billing_addresses(updated_at);

COMMIT;
