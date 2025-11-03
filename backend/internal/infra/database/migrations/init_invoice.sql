
-- Migration: Initialize Invoice domain
-- Mirrors backend/internal/domain/invoice/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS invoices (
  order_id           TEXT        PRIMARY KEY,
  subtotal           INTEGER     NOT NULL CHECK (subtotal >= 0),
  discount_amount    INTEGER     NOT NULL CHECK (discount_amount >= 0),
  tax_amount         INTEGER     NOT NULL CHECK (tax_amount >= 0),
  shipping_cost      INTEGER     NOT NULL CHECK (shipping_cost >= 0),
  total_amount       INTEGER     NOT NULL CHECK (total_amount >= 0),
  currency           TEXT        NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL,
  updated_at         TIMESTAMPTZ NOT NULL,
  billing_address_id TEXT        NOT NULL,

  -- Basic validations
  CONSTRAINT chk_invoices_currency_len CHECK (char_length(currency) = 3),
  CONSTRAINT chk_invoices_time_order CHECK (updated_at >= created_at),

  -- Keep total consistency with domain rule:
  CONSTRAINT chk_invoices_total_consistency
    CHECK (total_amount = subtotal - discount_amount + tax_amount + shipping_cost)
);

-- Order item level invoices
CREATE TABLE IF NOT EXISTS order_item_invoices (
  id            TEXT        PRIMARY KEY,
  order_item_id TEXT        NOT NULL,
  unit_price    INTEGER     NOT NULL CHECK (unit_price >= 0),
  total_price   INTEGER     NOT NULL CHECK (total_price >= 0),
  created_at    TIMESTAMPTZ NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL,

  -- Optional linkage to invoices (nullable to avoid strict domain coupling)
  order_id      TEXT        NULL REFERENCES invoices(order_id) ON DELETE CASCADE,

  CONSTRAINT chk_order_item_invoices_time_order CHECK (updated_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_invoices_currency            ON invoices(currency);
CREATE INDEX IF NOT EXISTS idx_invoices_created_at          ON invoices(created_at);
CREATE INDEX IF NOT EXISTS idx_invoices_updated_at          ON invoices(updated_at);
CREATE INDEX IF NOT EXISTS idx_invoices_billing_address_id  ON invoices(billing_address_id);

CREATE INDEX IF NOT EXISTS idx_order_item_invoices_item_id  ON order_item_invoices(order_item_id);
CREATE INDEX IF NOT EXISTS idx_order_item_invoices_order_id ON order_item_invoices(order_id);

COMMIT;
