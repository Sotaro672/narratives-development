
-- Migration: Initialize ListImage domain
-- Mirrors backend/internal/domain/listImage/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS list_images (
  id             TEXT        PRIMARY KEY,
  list_id        TEXT        NOT NULL,
  url            TEXT        NOT NULL,
  file_name      TEXT        NOT NULL,
  size           BIGINT      NOT NULL CHECK (size >= 0),
  display_order  INT         NOT NULL CHECK (display_order >= 0),

  created_at     TIMESTAMPTZ NOT NULL,
  created_by     TEXT        NOT NULL,
  updated_at     TIMESTAMPTZ NULL,
  updated_by     TEXT        NULL,
  deleted_at     TIMESTAMPTZ NULL,
  deleted_by     TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_list_images_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(list_id)) > 0
    AND char_length(trim(url)) > 0
    AND char_length(trim(file_name)) > 0
    AND char_length(trim(created_by)) > 0
  ),

  -- Time order
  CONSTRAINT chk_list_images_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Prevent duplicate file names per list (optional but useful)
CREATE UNIQUE INDEX IF NOT EXISTS ux_list_images_list_file
  ON list_images (list_id, file_name);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_list_images_list_id        ON list_images (list_id);
CREATE INDEX IF NOT EXISTS idx_list_images_display_order  ON list_images (list_id, display_order);
CREATE INDEX IF NOT EXISTS idx_list_images_created_at     ON list_images (created_at);
CREATE INDEX IF NOT EXISTS idx_list_images_updated_at     ON list_images (updated_at);
CREATE INDEX IF NOT EXISTS idx_list_images_deleted_at     ON list_images (deleted_at);

COMMIT;
