
-- Migration: Initialize TokenIcon domain
-- Mirrors backend/internal/domain/tokenIcon/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS token_icons (
  id          TEXT        PRIMARY KEY,
  url         TEXT        NOT NULL,
  file_name   TEXT        NOT NULL,
  size        BIGINT      NOT NULL CHECK (size >= 0),
  created_at  TIMESTAMPTZ NOT NULL,
  created_by  TEXT        NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL,
  updated_by  TEXT        NOT NULL,
  deleted_at  TIMESTAMPTZ,
  deleted_by  TEXT,

  -- Non-empty checks
  CONSTRAINT chk_ti_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(url)) > 0
    AND char_length(trim(file_name)) > 0
    AND char_length(trim(created_by)) > 0
    AND char_length(trim(updated_by)) > 0
  ),

  -- simple URL format
  CONSTRAINT chk_ti_url_format CHECK (url ~* '^(https?)://'),

  -- time order
  CHECK (updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

-- Indexes (optional)
CREATE INDEX IF NOT EXISTS idx_ti_created_by ON token_icons(created_by);
CREATE INDEX IF NOT EXISTS idx_ti_updated_by ON token_icons(updated_by);
CREATE INDEX IF NOT EXISTS idx_ti_created_at ON token_icons(created_at);
CREATE INDEX IF NOT EXISTS idx_ti_updated_at ON token_icons(updated_at);
CREATE INDEX IF NOT EXISTS idx_ti_deleted_at ON token_icons(deleted_at);

COMMIT;
