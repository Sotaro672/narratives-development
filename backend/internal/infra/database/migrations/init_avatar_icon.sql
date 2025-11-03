
CREATE TABLE IF NOT EXISTS avatar_icons (
  id TEXT PRIMARY KEY,
  avatar_id TEXT NULL,
  url TEXT NOT NULL,
  file_name TEXT NULL,
  size BIGINT NULL CHECK (size >= 0 AND size <= 10485760),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by TEXT NULL,
  -- Basic URL sanity check (http/https)
  CHECK (url ~* '^https?://'),
  -- Allow only known image extensions when file_name is provided
  CHECK (file_name IS NULL OR file_name ~* '\.(png|jpg|jpeg|webp|gif)$'),
  CHECK (updated_at IS NULL OR updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

CREATE INDEX IF NOT EXISTS idx_avatar_icons_avatar_id ON avatar_icons (avatar_id);
CREATE INDEX IF NOT EXISTS idx_avatar_icons_deleted_at ON avatar_icons (deleted_at);
