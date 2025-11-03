
-- Migration: Initialize TokenContents domain
-- Mirrors backend/internal/domain/tokenContents/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS token_contents (
  id          TEXT        PRIMARY KEY,
  name        TEXT        NOT NULL,
  type        TEXT        NOT NULL CHECK (type IN ('image','video','pdf','document')),
  url         TEXT        NOT NULL,
  size        BIGINT      NOT NULL CHECK (size >= 0),
  created_at  TIMESTAMPTZ NOT NULL,
  created_by  TEXT        NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL,
  updated_by  TEXT        NOT NULL,
  deleted_at  TIMESTAMPTZ,
  deleted_by  TEXT,

  -- Non-empty checks
  CONSTRAINT chk_tc_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(name)) > 0
    AND char_length(trim(url)) > 0
    AND char_length(trim(created_by)) > 0
    AND char_length(trim(updated_by)) > 0
  ),

  -- simple URL format
  CONSTRAINT chk_tc_url_format CHECK (url ~* '^(https?)://'),

  -- time order
  CHECK (updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tc_type        ON token_contents(type);
CREATE INDEX IF NOT EXISTS idx_tc_created_at  ON token_contents(created_at);
CREATE INDEX IF NOT EXISTS idx_tc_updated_at  ON token_contents(updated_at);
CREATE INDEX IF NOT EXISTS idx_tc_deleted_at  ON token_contents(deleted_at);

COMMIT;
