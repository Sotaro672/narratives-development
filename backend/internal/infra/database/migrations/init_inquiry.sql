
CREATE TABLE IF NOT EXISTS inquiries (
  id                   UUID        PRIMARY KEY,
  avatar_id            TEXT        NOT NULL,
  subject              TEXT        NOT NULL,
  content              TEXT        NOT NULL,
  status               TEXT        NOT NULL,
  inquiry_type         TEXT        NOT NULL,
  product_blueprint_id TEXT        NULL,
  token_blueprint_id   TEXT        NULL,
  assignee_id          TEXT        NULL,
  image                TEXT        NULL,
  created_at           TIMESTAMPTZ NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL,
  updated_by           TEXT        NULL,
  deleted_at           TIMESTAMPTZ NULL,
  deleted_by           TEXT        NULL,

  -- Non-empty checks
  CONSTRAINT chk_inquiries_non_empty CHECK (
    char_length(trim(subject)) > 0 AND
    char_length(trim(content)) > 0 AND
    char_length(trim(status)) > 0 AND
    char_length(trim(inquiry_type)) > 0
  ),

  -- time order
  CONSTRAINT chk_inquiries_time_order CHECK (updated_at >= created_at),
  CONSTRAINT chk_inquiries_deleted_time CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_inquiries_avatar_id       ON inquiries(avatar_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_assignee_id     ON inquiries(assignee_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_status          ON inquiries(status);
CREATE INDEX IF NOT EXISTS idx_inquiries_inquiry_type    ON inquiries(inquiry_type);
CREATE INDEX IF NOT EXISTS idx_inquiries_created_at      ON inquiries(created_at);
CREATE INDEX IF NOT EXISTS idx_inquiries_updated_at      ON inquiries(updated_at);
CREATE INDEX IF NOT EXISTS idx_inquiries_deleted_at      ON inquiries(deleted_at);
