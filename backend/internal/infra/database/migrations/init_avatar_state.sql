
CREATE TABLE IF NOT EXISTS avatar_states (
  id TEXT PRIMARY KEY,
  avatar_id TEXT NOT NULL UNIQUE,
  follower_count BIGINT NOT NULL DEFAULT 0 CHECK (follower_count >= 0),
  following_count BIGINT NOT NULL DEFAULT 0 CHECK (following_count >= 0),
  post_count BIGINT NOT NULL DEFAULT 0 CHECK (post_count >= 0),
  last_active_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NULL,
  CHECK (updated_at IS NULL OR updated_at >= last_active_at)
);

CREATE INDEX IF NOT EXISTS idx_avatar_states_last_active_at
  ON avatar_states (last_active_at DESC);

CREATE INDEX IF NOT EXISTS idx_avatar_states_updated_at
  ON avatar_states (updated_at DESC);
