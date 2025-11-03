
-- Migration: Initialize Announcement domain
-- Mirrors backend/internal/domain/annoucement/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS announcements (
  id               TEXT        PRIMARY KEY,
  title            TEXT        NOT NULL,
  content          TEXT        NOT NULL,
  category         TEXT        NOT NULL,
  target_audience  TEXT        NOT NULL,
  target_token     TEXT,
  target_products  TEXT[]      NOT NULL DEFAULT '{}',
  target_avatars   TEXT[]      NOT NULL DEFAULT '{}',
  is_published     BOOLEAN     NOT NULL DEFAULT false,
  published_at     TIMESTAMPTZ,
  attachments      TEXT[]      NOT NULL DEFAULT '{}',
  status           TEXT        NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL,
  created_by       TEXT        NOT NULL,
  updated_at       TIMESTAMPTZ,
  updated_by       TEXT,
  deleted_at       TIMESTAMPTZ,
  deleted_by       TEXT,

  -- Non-empty checks
  CONSTRAINT chk_ann_title_non_empty       CHECK (char_length(trim(title)) > 0),
  CONSTRAINT chk_ann_content_non_empty     CHECK (char_length(trim(content)) > 0),
  CONSTRAINT chk_ann_category_non_empty    CHECK (char_length(trim(category)) > 0),
  CONSTRAINT chk_ann_audience_non_empty    CHECK (char_length(trim(target_audience)) > 0),
  CONSTRAINT chk_ann_status_non_empty      CHECK (char_length(trim(status)) > 0),
  CONSTRAINT chk_ann_created_by_non_empty  CHECK (char_length(trim(created_by)) > 0),

  -- Time order
  CONSTRAINT chk_ann_time_updated   CHECK (updated_at   IS NULL OR updated_at   >= created_at),
  CONSTRAINT chk_ann_time_deleted   CHECK (deleted_at   IS NULL OR deleted_at   >= created_at),
  CONSTRAINT chk_ann_time_published CHECK (published_at IS NULL OR published_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_ann_is_published ON announcements(is_published);
CREATE INDEX IF NOT EXISTS idx_ann_status       ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_ann_category     ON announcements(category);
CREATE INDEX IF NOT EXISTS idx_ann_created_at   ON announcements(created_at);
CREATE INDEX IF NOT EXISTS idx_ann_published_at ON announcements(published_at);

COMMIT;
