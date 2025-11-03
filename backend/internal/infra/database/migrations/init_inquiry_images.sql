
-- Aggregate header (optional)
CREATE TABLE IF NOT EXISTS inquiry_images (
  id TEXT PRIMARY KEY  -- inquiryId
);

-- Image files table (per-image rows)
CREATE TABLE IF NOT EXISTS inquiry_image_files (
  inquiry_id TEXT NOT NULL REFERENCES inquiry_images(id) ON DELETE CASCADE,
  file_name  TEXT NOT NULL,
  file_url   TEXT NOT NULL,
  file_size  BIGINT NOT NULL CHECK (file_size >= 0),
  mime_type  TEXT NOT NULL,
  width      INT NULL CHECK (width > 0),
  height     INT NULL CHECK (height > 0),

  created_at TIMESTAMPTZ NOT NULL,
  created_by TEXT NOT NULL,
  updated_at TIMESTAMPTZ NULL,
  updated_by TEXT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by TEXT NULL,

  PRIMARY KEY (inquiry_id, file_name)
);

-- Prevent duplicate URLs per inquiry (domain enforces URL uniqueness)
CREATE UNIQUE INDEX IF NOT EXISTS ux_inquiry_image_files_inquiry_url
  ON inquiry_image_files (inquiry_id, file_url);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_inquiry_image_files_inquiry
  ON inquiry_image_files (inquiry_id);
CREATE INDEX IF NOT EXISTS idx_inquiry_image_files_created_at
  ON inquiry_image_files (created_at);
CREATE INDEX IF NOT EXISTS idx_inquiry_image_files_mime_type
  ON inquiry_image_files (mime_type);
CREATE INDEX IF NOT EXISTS idx_inquiry_image_files_deleted_at
  ON inquiry_image_files (deleted_at);
