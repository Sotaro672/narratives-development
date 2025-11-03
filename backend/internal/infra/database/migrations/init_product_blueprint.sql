
-- Migration: Initialize ProductBlueprint domain
-- Mirrors backend/internal/domain/productBlueprint/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS product_blueprints (
  id                     TEXT        PRIMARY KEY,
  product_name           TEXT        NOT NULL,
  brand_id               TEXT        NOT NULL,
  item_type              TEXT        NOT NULL CHECK (item_type IN ('tops','bottoms','other')),
  fit                    TEXT        NOT NULL DEFAULT '',
  material               TEXT        NOT NULL DEFAULT '',
  weight                 DOUBLE PRECISION NOT NULL CHECK (weight >= 0),
  quality_assurance      TEXT[]      NOT NULL DEFAULT '{}',
  product_id_tag_type    TEXT        NOT NULL CHECK (product_id_tag_type IN ('qr','nfc')),
  model_variations       JSONB       NOT NULL DEFAULT '[]'::jsonb, -- TS: ModelVariation[]
  assignee_id            TEXT        NOT NULL,
  created_by             TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Non-empty checks
  CONSTRAINT chk_pb_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(product_name)) > 0
    AND char_length(trim(brand_id)) > 0
    AND char_length(trim(assignee_id)) > 0
  ),

  -- quality_assurance: no empty items
  CONSTRAINT chk_pb_qa_no_empty CHECK (
    NOT EXISTS (SELECT 1 FROM unnest(quality_assurance) t(x) WHERE x = '')
  ),

  -- model_variations must be a JSON array
  CONSTRAINT chk_pb_model_variations_array CHECK (jsonb_typeof(model_variations) = 'array'),

  CHECK (updated_at >= created_at)
);

-- Optional FKs (adjust to your schema)
ALTER TABLE product_blueprints
  ADD CONSTRAINT fk_pb_brand
  FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE RESTRICT;

ALTER TABLE product_blueprints
  ADD CONSTRAINT fk_pb_assignee
  FOREIGN KEY (assignee_id) REFERENCES members(id) ON DELETE RESTRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_pb_brand_id   ON product_blueprints(brand_id);
CREATE INDEX IF NOT EXISTS idx_pb_created_at ON product_blueprints(created_at);

COMMIT;
