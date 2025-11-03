
-- Migration: Initialize orders table (mirrors domain/order/entity.go)

BEGIN;

CREATE TABLE IF NOT EXISTS orders (
  id                   TEXT        PRIMARY KEY,
  order_number         TEXT        NOT NULL,
  status               TEXT        NOT NULL CHECK (status IN ('paid','transferred')),
  user_id              TEXT        NOT NULL,
  shipping_address_id  TEXT        NOT NULL,
  billing_address_id   TEXT        NOT NULL,
  list_id              TEXT        NOT NULL,
  items                JSONB       NOT NULL DEFAULT '[]'::jsonb,  -- array of item ids
  invoice_id           TEXT        NOT NULL,
  payment_id           TEXT        NOT NULL,
  fulfillment_id       TEXT        NOT NULL,
  tracking_id          TEXT        NULL,
  transffered_date     TIMESTAMPTZ NULL,                          -- note: TS field uses this spelling
  last_update          TIMESTAMPTZ NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL,
  updated_by           TEXT        NULL,
  deleted_at           TIMESTAMPTZ NULL,
  deleted_by           TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_orders_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(order_number)) > 0
    AND char_length(trim(user_id)) > 0
    AND char_length(trim(shipping_address_id)) > 0
    AND char_length(trim(billing_address_id)) > 0
    AND char_length(trim(list_id)) > 0
    AND char_length(trim(invoice_id)) > 0
    AND char_length(trim(payment_id)) > 0
    AND char_length(trim(fulfillment_id)) > 0
  ),

  -- items must be a JSON array with at least one element
  CONSTRAINT chk_orders_items_array CHECK (
    jsonb_typeof(items) = 'array' AND jsonb_array_length(items) >= 1
  ),

  -- Time order coherence
  CONSTRAINT chk_orders_time_order CHECK (
    updated_at >= created_at
    AND last_update >= created_at
    AND last_update >= updated_at
    AND (transffered_date IS NULL OR transffered_date >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  ),

  -- UpdatedBy/DeletedBy coherence
  CONSTRAINT chk_orders_updated_by_non_empty CHECK (
    updated_by IS NULL OR char_length(trim(updated_by)) > 0
  ),
  CONSTRAINT chk_orders_deleted_pair CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL)
    OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
  )
);

-- Useful indexes
CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_order_number ON orders(order_number);
CREATE INDEX IF NOT EXISTS idx_orders_status            ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_user_id           ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_transffered_date  ON orders(transffered_date);
CREATE INDEX IF NOT EXISTS idx_orders_last_update       ON orders(last_update);
CREATE INDEX IF NOT EXISTS idx_orders_created_at        ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_orders_updated_at        ON orders(updated_at);
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at        ON orders(deleted_at);

COMMIT;
