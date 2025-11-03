
CREATE TABLE IF NOT EXISTS campaign_images (
  id UUID PRIMARY KEY,
  campaign_id UUID NOT NULL,
  image_url TEXT NOT NULL,
  width INT NULL,
  height INT NULL,
  file_size BIGINT NULL,
  mime_type TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT NULL,
  updated_at TIMESTAMPTZ NULL,
  updated_by TEXT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by TEXT NULL,
  CONSTRAINT fk_campaign_images_campaign
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,
  CHECK (width IS NULL OR width > 0),
  CHECK (height IS NULL OR height > 0),
  CHECK (file_size IS NULL OR (file_size >= 1 AND file_size <= 20971520)),
  CHECK (mime_type IS NULL OR mime_type ~* '^[A-Za-z0-9.+-]+/[A-Za
'-z0-9.+-]+$'),
  CHECK (created_at IS NOT NULL),
  CHECK (updated_at IS NULL OR updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at),
  CHECK (deleted_at IS NULL OR updated_at IS NULL OR deleted_at >= updated_at)
);
CREATE INDEX IF NOT EXISTS idx_campaign_images_campaign_id ON campaign_images(campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_images_deleted_at ON campaign_images(deleted_at);
