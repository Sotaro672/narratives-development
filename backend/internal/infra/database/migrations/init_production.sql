
-- Migration: Initialize Production domain (productions)
-- Mirrors backend/internal/domain/production/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS productions (
  id                   TEXT        PRIMARY KEY,
  product_blueprint_id TEXT        NOT NULL,
  assignee_id          TEXT        NOT NULL,
  models               JSONB       NOT NULL DEFAULT '[]'::jsonb, -- [{modelId, quantity}]
  status               TEXT        NOT NULL CHECK (status IN ('manufacturing','printed','inspected','planning','deleted','suspended')),
  printed_at           TIMESTAMPTZ,
  inspected_at         TIMESTAMPTZ,
  created_by           TEXT,
  created_at           TIMESTAMPTZ DEFAULT NOW(),
  updated_by           TEXT,
  updated_at           TIMESTAMPTZ,
  deleted_by           TEXT,
  deleted_at           TIMESTAMPTZ,

  -- Non-empty checks
  CONSTRAINT chk_productions_ids_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(product_blueprint_id)) > 0
    AND char_length(trim(assignee_id)) > 0
  ),

  -- models は配列であることだけ最低限チェック（要件に応じて厳格化可）
  CONSTRAINT chk_productions_models_is_array CHECK (jsonb_typeof(models) = 'array'),

  -- Status/time coherence (aligns with entity validation)
  CONSTRAINT chk_productions_status_coherence CHECK (
    (status IN ('manufacturing','planning','suspended'))
    OR (status = 'printed'   AND printed_at IS NOT NULL AND (inspected_at IS NULL OR inspected_at >= printed_at))
    OR (status = 'inspected' AND printed_at IS NOT NULL AND inspected_at IS NOT NULL AND inspected_at >= printed_at)
    OR (status = 'deleted'   AND deleted_at IS NOT NULL)
  ),

  -- updated_at >= created_at if both present; deleted_at >= created_at if both present
  CONSTRAINT chk_productions_time_order CHECK (
    (updated_at IS NULL OR created_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR created_at IS NULL OR deleted_at >= created_at)
  )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_productions_status            ON productions(status);
CREATE INDEX IF NOT EXISTS idx_productions_product_blueprint ON productions(product_blueprint_id);
CREATE INDEX IF NOT EXISTS idx_productions_assignee          ON productions(assignee_id);
CREATE INDEX IF NOT EXISTS idx_productions_created_at        ON productions(created_at);
CREATE INDEX IF NOT EXISTS idx_productions_updated_at        ON productions(updated_at);
CREATE INDEX IF NOT EXISTS idx_productions_deleted_at        ON productions(deleted_at);

COMMIT;
