
-- Migration: Initialize User domain
-- Mirrors backend/internal/domain/user/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS users (
  id               TEXT        PRIMARY KEY,
  first_name       TEXT,
  first_name_kana  TEXT,
  last_name_kana   TEXT,
  last_name        TEXT,
  email            TEXT,
  phone_number     TEXT,
  created_at       TIMESTAMPTZ NOT NULL,
  updated_at       TIMESTAMPTZ NOT NULL,
  deleted_at       TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_users_non_empty CHECK (char_length(trim(id)) > 0),

  -- time order
  CONSTRAINT chk_users_time_order CHECK (
    updated_at >= created_at AND deleted_at >= created_at
  )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email       ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_created_at  ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_updated_at  ON users(updated_at);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at  ON users(deleted_at);

COMMIT;
