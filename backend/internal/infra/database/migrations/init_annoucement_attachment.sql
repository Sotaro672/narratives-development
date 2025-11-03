
CREATE TABLE IF NOT EXISTS announcement_attachments (
  announcement_id TEXT NOT NULL,
  file_name TEXT NOT NULL,
  file_url TEXT NOT NULL,
  file_size BIGINT NOT NULL CHECK (file_size > 0 AND file_size <= 52428800),
  mime_type TEXT NOT NULL CHECK (
    mime_type IN ('application/pdf','image/jpeg','image/png','image/webp','image/gif','text/plain')
  ),
  -- Use a composite primary key since there is no standalone id in the domain
  CONSTRAINT pk_announcement_attachments PRIMARY KEY (announcement_id, file_name),
  -- Basic URL sanity check (http/https)
  CHECK (file_url ~* '^https?://')
);

CREATE INDEX IF NOT EXISTS idx_announcement_attachments_announcement_id
  ON announcement_attachments (announcement_id);

CREATE INDEX IF NOT EXISTS idx_announcement_attachments_mime_type
  ON announcement_attachments (mime_type);
