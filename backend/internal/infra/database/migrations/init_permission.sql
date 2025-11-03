
-- ==========================================
-- Permissions Table Initialization
-- ==========================================

CREATE TABLE IF NOT EXISTS permissions (
  id TEXT PRIMARY KEY,             -- 例: 'perm_001'
  name TEXT NOT NULL UNIQUE,       -- 例: 'brand.create'
  description TEXT NOT NULL,       -- 権限の説明
  category TEXT NOT NULL CHECK (
    category IN (
      'wallet',
      'inquiry',
      'organization',
      'brand',
      'member',
      'order',
      'product',
      'campaign',
      'token',
      'inventory',
      'production',
      'analytics',
      'system'
    )
  ),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ==========================================
-- Indexes
-- ==========================================

-- カテゴリでの絞り込み用
CREATE INDEX IF NOT EXISTS idx_permissions_category ON permissions(category);

-- 名前での検索用
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions(name);

-- ==========================================
-- Comments
-- ==========================================

COMMENT ON TABLE permissions IS 'Permission master table defining system permissions';
COMMENT ON COLUMN permissions.id IS 'Primary key, e.g. perm_001';
COMMENT ON COLUMN permissions.name IS 'Unique permission key, e.g. brand.create';
COMMENT ON COLUMN permissions.category IS 'Permission category (wallet, brand, token, etc.)';
