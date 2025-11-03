
-- Migration: Initialize List domain
-- Mirrors backend/internal/domain/list/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS lists (
  id            TEXT        PRIMARY KEY,
  inventory_id  TEXT        NOT NULL,
  status        TEXT        NOT NULL CHECK (status IN ('listing','suspended')),
  assignee_id   TEXT        NOT NULL,
  image_url     TEXT        NOT NULL,
  description   TEXT        NOT NULL,
  created_by    TEXT        NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL,
  updated_by    TEXT        NULL,
  updated_at    TIMESTAMPTZ NULL,
  deleted_at    TIMESTAMPTZ NULL,
  deleted_by    TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_lists_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(inventory_id)) > 0
    AND char_length(trim(assignee_id)) > 0
    AND char_length(trim(image_url)) > 0
    AND char_length(trim(description)) > 0
    AND char_length(trim(created_by)) > 0
  ),

  -- Description length policy (aligns with MaxDescriptionLength)
  CONSTRAINT chk_lists_description_len CHECK (char_length(description) <= 2000),

  -- Time order
  CONSTRAINT chk_lists_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Normalized prices per modelNumber
CREATE TABLE IF NOT EXISTS list_prices (
  list_id      TEXT    NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  model_number TEXT    NOT NULL,
  price        INTEGER NOT NULL CHECK (price >= 0 AND price <= 10000000),
  PRIMARY KEY (list_id, model_number),
  CONSTRAINT chk_list_prices_model_non_empty CHECK (char_length(trim(model_number)) > 0)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_lists_inventory_id ON lists (inventory_id);
CREATE INDEX IF NOT EXISTS idx_lists_status       ON lists (status);
CREATE INDEX IF NOT EXISTS idx_lists_assignee_id  ON lists (assignee_id);
CREATE INDEX IF NOT EXISTS idx_lists_created_at   ON lists (created_at);
CREATE INDEX IF NOT EXISTS idx_lists_updated_at   ON lists (updated_at);

CREATE INDEX IF NOT EXISTS idx_list_prices_model_number ON list_prices (model_number);

COMMIT;
