
-- Migration: Initialize TokenBlueprint domain
-- Mirrors backend/internal/domain/tokenBlueprint/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS token_blueprints (
  id               TEXT        PRIMARY KEY,
  name             TEXT        NOT NULL,
  symbol           TEXT        NOT NULL,
  brand_id         TEXT        NOT NULL,
  description      TEXT        NOT NULL,
  icon_url         TEXT,
  content_files    TEXT[]      NOT NULL DEFAULT '{}',
  assignee_id      TEXT        NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by       TEXT        NOT NULL,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Non-empty checks
  CONSTRAINT chk_tb_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(name)) > 0
    AND char_length(trim(symbol)) > 0
    AND char_length(trim(brand_id)) > 0
    AND char_length(trim(description)) > 0
    AND char_length(trim(assignee_id)) > 0
    AND char_length(trim(created_by)) > 0
  ),

  -- Symbol format (matches ^[A-Z0-9]{1,10}$)
  CONSTRAINT chk_tb_symbol_format CHECK (symbol ~ '^[A-Z0-9]{1,10}$'),

  -- icon_url must be http/https if present
  CONSTRAINT chk_tb_icon_url_format CHECK (
    icon_url IS NULL OR icon_url ~* '^(https?)://'
  ),

  -- content_files: no empty items
  CONSTRAINT chk_tb_content_files_no_empty CHECK (
    NOT EXISTS (SELECT 1 FROM unnest(content_files) t(x) WHERE x = '')
  ),

  CHECK (updated_at >= created_at)
);

-- Optional FKs (add only if referenced tables exist)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'brands'
  ) THEN
    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_brand
        FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'members'
  ) THEN
    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_assignee
        FOREIGN KEY (assignee_id) REFERENCES members(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;
END$$;

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tb_brand_id    ON token_blueprints(brand_id);
CREATE INDEX IF NOT EXISTS idx_tb_symbol      ON token_blueprints(symbol);
CREATE INDEX IF NOT EXISTS idx_tb_created_at  ON token_blueprints(created_at);
CREATE INDEX IF NOT EXISTS idx_tb_updated_at  ON token_blueprints(updated_at);

COMMIT;
