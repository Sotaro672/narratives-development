
CREATE TABLE IF NOT EXISTS discounts (
  id TEXT PRIMARY KEY,                 -- 例: 'discount_xxx'
  list_id TEXT NOT NULL,               -- 出品ID（型はシステム都合に合わせてUUID等へ変更可）
  description TEXT NULL,               -- 割引の説明
  discounted_by TEXT NOT NULL,         -- 設定者のメンバーID（型はシステム都合に合わせてUUID等へ変更可）
  discounted_at TIMESTAMPTZ NOT NULL,  -- 設定日時
  updated_by TEXT NOT NULL,            -- 最終更新者ID
  updated_at TIMESTAMPTZ NOT NULL      -- 最終更新日時
);

-- modelNumberごとの割引率（正規化）
CREATE TABLE IF NOT EXISTS discount_items (
  discount_id TEXT NOT NULL REFERENCES discounts(id) ON DELETE CASCADE,
  model_number TEXT NOT NULL,
  percent INT NOT NULL CHECK (percent >= 0 AND percent <= 100),
  PRIMARY KEY (discount_id, model_number)
);

-- Search/Sort helpers
CREATE INDEX IF NOT EXISTS idx_discounts_list_id ON discounts (list_id);
CREATE INDEX IF NOT EXISTS idx_discounts_discounted_at ON discounts (discounted_at);
CREATE INDEX IF NOT EXISTS idx_discounts_updated_at ON discounts (updated_at);
CREATE INDEX IF NOT EXISTS idx_discount_items_model_number ON discount_items (model_number);
