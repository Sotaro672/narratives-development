
-- Migration: Initialize messages table

BEGIN;

CREATE TABLE IF NOT EXISTS messages (
  id           UUID        PRIMARY KEY,
  sender_id    UUID        NOT NULL,
  receiver_id  UUID        NOT NULL,
  content      TEXT        NOT NULL,
  status       TEXT        NOT NULL CHECK (status IN ('draft','sent','delivered','read')),
  created_at   TIMESTAMPTZ NOT NULL,
  updated_at   TIMESTAMPTZ NULL,
  deleted_at   TIMESTAMPTZ NULL,
  read_at      TIMESTAMPTZ NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_messages_non_empty CHECK (char_length(trim(content)) > 0),

  -- Time order
  CONSTRAINT chk_messages_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
    AND (read_at IS NULL OR read_at >= created_at)
  ),

  -- Foreign keys to members
  CONSTRAINT fk_messages_sender   FOREIGN KEY (sender_id)   REFERENCES members (id) ON DELETE RESTRICT,
  CONSTRAINT fk_messages_receiver FOREIGN KEY (receiver_id) REFERENCES members (id) ON DELETE RESTRICT
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_messages_sender_id    ON messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id  ON messages (receiver_id);
CREATE INDEX IF NOT EXISTS idx_messages_status       ON messages (status);
CREATE INDEX IF NOT EXISTS idx_messages_created_at   ON messages (created_at);
CREATE INDEX IF NOT EXISTS idx_messages_updated_at   ON messages (updated_at);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at   ON messages (deleted_at);
CREATE INDEX IF NOT EXISTS idx_messages_read_at      ON messages (read_at);

COMMIT;
