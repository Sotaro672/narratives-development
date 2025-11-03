
-- Migration: Initialize/Update Model domain (model_sets, model_variations)
-- Mirrors backend/internal/domain/model/entity.go

BEGIN;

-- A set of variations for a product (tracks UpdatedAt at product scope)
CREATE TABLE IF NOT EXISTS model_sets (
  product_id           TEXT        PRIMARY KEY,
  product_blueprint_id TEXT        NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_model_sets_non_empty CHECK (
    char_length(trim(product_id)) > 0
    AND char_length(trim(product_blueprint_id)) > 0
  )
);

-- Ensure blueprint is unique to reference from variations
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_sets_product_blueprint_id
  ON model_sets(product_blueprint_id);

-- Each concrete variation (measurements embedded as JSONB)
CREATE TABLE IF NOT EXISTS model_variations (
  id                   TEXT NOT NULL PRIMARY KEY,
  product_blueprint_id TEXT NOT NULL, -- TS: ModelVariation.productBlueprintId
  model_number         TEXT NOT NULL,
  size                 TEXT NOT NULL,
  color                TEXT NOT NULL,
  measurements         JSONB NOT NULL DEFAULT '{}'::jsonb, -- TS: Record<string, number>

  -- Audit (optional in TS)
  created_at TIMESTAMPTZ NULL,
  created_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,
  updated_at TIMESTAMPTZ NULL,
  updated_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,

  CONSTRAINT chk_model_variations_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(product_blueprint_id)) > 0
    AND char_length(trim(model_number)) > 0
    AND char_length(trim(size)) > 0
    AND char_length(trim(color)) > 0
  ),
  -- measurements must be a JSON object
  CONSTRAINT chk_model_variations_measurements_object CHECK (jsonb_typeof(measurements) = 'object'),

  -- Audit coherence (when provided)
  CONSTRAINT chk_model_variations_time_order CHECK (
    updated_at IS NULL OR created_at IS NULL OR updated_at >= created_at
  )
);

-- Relationships
ALTER TABLE model_sets
  ADD CONSTRAINT fk_model_sets_product
  FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

-- variations -> sets by product_blueprint_id
ALTER TABLE model_variations
  ADD CONSTRAINT fk_model_variations_set_by_blueprint
  FOREIGN KEY (product_blueprint_id) REFERENCES model_sets(product_blueprint_id) ON DELETE CASCADE;

-- Uniqueness to avoid duplicates per blueprint
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_variations_blueprint_modelnumber_size_color
  ON model_variations(product_blueprint_id, model_number, size, color);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_model_variations_product_blueprint_id  ON model_variations(product_blueprint_id);
CREATE INDEX IF NOT EXISTS idx_model_variations_model_number          ON model_variations(model_number);
CREATE INDEX IF NOT EXISTS idx_model_variations_size                  ON model_variations(size);
CREATE INDEX IF NOT EXISTS idx_model_variations_color                 ON model_variations(color);
CREATE INDEX IF NOT EXISTS idx_model_variations_measurements_gin      ON model_variations USING GIN (measurements);
CREATE INDEX IF NOT EXISTS idx_model_variations_created_at            ON model_variations (created_at);
CREATE INDEX IF NOT EXISTS idx_model_variations_updated_at            ON model_variations (updated_at);
CREATE INDEX IF NOT EXISTS idx_model_variations_deleted_at            ON model_variations (deleted_at);

COMMIT;
