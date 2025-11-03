
-- Migration: Initialize Inventory domain
-- Mirrors backend/internal/domain/inventory/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS inventories (
  id              TEXT        PRIMARY KEY,
  connected_token TEXT,
  models          JSONB       NOT NULL DEFAULT '[]'::jsonb, -- [{modelNumber, quantity}]
  location        TEXT        NOT NULL,
  status          TEXT        NOT NULL CHECK (status IN ('inspecting','inspected','listed','discarded','deleted')),
  created_by      TEXT        NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_by      TEXT        NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_inventories_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(location)) > 0
    AND char_length(trim(created_by)) > 0
    AND char_length(trim(updated_by)) > 0
  ),

  -- models は配列であることを最低限チェック
  CONSTRAINT chk_inventories_models_is_array CHECK (jsonb_typeof(models) = 'array'),

  -- time order
  CONSTRAINT chk_inventories_time_order CHECK (updated_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_inventories_status           ON inventories(status);
CREATE INDEX IF NOT EXISTS idx_inventories_connected_token  ON inventories(connected_token);
CREATE INDEX IF NOT EXISTS idx_inventories_created_at       ON inventories(created_at);
CREATE INDEX IF NOT EXISTS idx_inventories_updated_at       ON inventories(updated_at);
CREATE INDEX IF NOT EXISTS idx_inventories_location         ON inventories(location);

COMMIT;
