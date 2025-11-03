
CREATE TABLE members (
  id UUID PRIMARY KEY,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  first_name_kana VARCHAR(100),
  last_name_kana VARCHAR(100),
  email VARCHAR(255) UNIQUE,
  role VARCHAR(50) NOT NULL,
  authorizations TEXT[] NOT NULL,
  assigned_brands TEXT[],

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  updated_by TEXT,
  deleted_at TIMESTAMPTZ,
  deleted_by TEXT
);
