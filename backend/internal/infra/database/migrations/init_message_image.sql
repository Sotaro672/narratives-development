
-- Migration: Initialize message_images table

BEGIN;

CREATE TABLE IF NOT EXISTS message_images (
  message_id     UUID        NOT NULL,
  file_name      TEXT        NOT NULL,
  file_url       TEXT        NOT NULL,
  file_size      BIGINT      NOT NULL CHECK (file_size >= 0),
  mime_type      TEXT        NOT NULL,
  width          INT         NULL CHECK (width IS NULL OR (width >= 1 AND width <= 10000)),
  height         INT         NULL CHECK (height IS NULL OR (height IS NULL) OR (height >= 1 AND height <= 10000)),
  created_at     TIMESTAMPTZ NOT NULL,
  updated_at     TIMESTAMPTZ NULL,
  deleted_at     TIMESTAMPTZ NULL,

  CONSTRAINT pk_message_images PRIMARY KEY (message_id, file_name),

  -- Basic non-empty checks
  CONSTRAINT chk_message_images_non_empty CHECK (
    char_length(trim(file_name)) > 0
    AND char_length(trim(file_url)) > 0
    AND char_length(trim(mime_type)) > 0
  ),

  -- Time order
  CONSTRAINT chk_message_images_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_message_images_message_id  ON message_images (message_id);
CREATE INDEX IF NOT EXISTS idx_message_images_created_at  ON message_images (created_at);
CREATE INDEX IF NOT EXISTS idx_message_images_deleted_at  ON message_images (deleted_at);

COMMIT;
