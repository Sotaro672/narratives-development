
-- Migration: Initialize order_items table (mirrors domain/orderItem/entity.go)

BEGIN;

CREATE TABLE IF NOT EXISTS order_items (
  id            TEXT    PRIMARY KEY,
  model_id      TEXT    NOT NULL,
  sale_id       TEXT    NOT NULL,
  inventory_id  TEXT    NOT NULL,
  quantity      INTEGER NOT NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_order_items_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(model_id)) > 0
    AND char_length(trim(sale_id)) > 0
    AND char_length(trim(inventory_id)) > 0
  },

  -- Quantity policy (MinQuantity = 1)
  CONSTRAINT chk_order_items_quantity CHECK (quantity >= 1)
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_order_items_model_id     ON order_items(model_id);
CREATE INDEX IF NOT EXISTS idx_order_items_sale_id      ON order_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_order_items_inventory_id ON order_items(inventory_id);

COMMIT;
