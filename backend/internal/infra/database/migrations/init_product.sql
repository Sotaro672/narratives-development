
-- Migration: Initialize/Update Product domain
-- Mirrors backend/internal/domain/product/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS products (
  id                TEXT        PRIMARY KEY,
  model_id          TEXT        NOT NULL,
  production_id     TEXT        NOT NULL,
  inspection_result TEXT        NOT NULL CHECK (inspection_result IN ('notYet','passed','failed','notManufactured')),
  connected_token   TEXT        NULL,

  printed_at        TIMESTAMPTZ NULL,
  printed_by        TEXT        NULL,

  inspected_at      TIMESTAMPTZ NULL,
  inspected_by      TEXT        NULL,

  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by        TEXT        NOT NULL,

  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Non-empty checks
  CONSTRAINT chk_products_ids_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(model_id)) > 0
    AND char_length(trim(production_id)) > 0
  ),
  CONSTRAINT chk_products_connected_token_non_empty CHECK (
    connected_token IS NULL OR char_length(trim(connected_token)) > 0
  ),
  CONSTRAINT chk_products_printed_by_non_empty CHECK (
    printed_by IS NULL OR char_length(trim(printed_by)) > 0
  ),
  CONSTRAINT chk_products_inspected_by_non_empty CHECK (
    inspected_by IS NULL OR char_length(trim(inspected_by)) > 0
  ),
  CONSTRAINT chk_products_updated_by_non_empty CHECK (
    char_length(trim(updated_by)) > 0
  ),

  -- Printed coherence: both NULL or both present
  CONSTRAINT chk_products_printed_coherence CHECK (
    (printed_by IS NULL AND printed_at IS NULL)
    OR
    (printed_by IS NOT NULL AND char_length(trim(printed_by)) > 0 AND printed_at IS NOT NULL)
  ),

  -- Coherence with inspection_result:
  -- passed/failed: inspected_by, inspected_at required
  -- notYet/notManufactured: inspected_by, inspected_at must be NULL
  CONSTRAINT chk_products_inspection_coherence CHECK (
    (inspection_result IN ('notYet','notManufactured') AND inspected_by IS NULL AND inspected_at IS NULL)
    OR
    (inspection_result IN ('passed','failed') AND inspected_by IS NOT NULL AND char_length(trim(inspected_by)) > 0 AND inspected_at IS NOT NULL)
  ),

  -- Optional FK: disconnect token automatically when deleted
  CONSTRAINT fk_products_connected_token
    FOREIGN KEY (connected_token) REFERENCES tokens(mint_address) ON DELETE SET NULL
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_products_model_id           ON products(model_id);
CREATE INDEX IF NOT EXISTS idx_products_production_id      ON products(production_id);
CREATE INDEX IF NOT EXISTS idx_products_inspection_result  ON products(inspection_result);
CREATE INDEX IF NOT EXISTS idx_products_printed_at         ON products(printed_at);
CREATE INDEX IF NOT EXISTS idx_products_inspected_at       ON products(inspected_at);
CREATE INDEX IF NOT EXISTS idx_products_updated_at         ON products(updated_at);

COMMIT;
