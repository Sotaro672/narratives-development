
-- Avatars DDL generated from domain/avatar entity.

CREATE TABLE IF NOT EXISTS avatars (
  id             TEXT PRIMARY KEY,
  user_id        TEXT        NOT NULL,               -- ユーザーID（型はアプリ都合でTEXT）
  avatar_name    TEXT        NOT NULL,               -- 最大50文字（CHECKで制約）
  avatar_icon_id TEXT,                               -- 画像ID（任意）
  wallet_address TEXT,                               -- wallets.wallet_address への外部キー（任意）
  bio            TEXT,                               -- 最大1000文字
  website        TEXT,                               -- URL（形式はアプリ層で検証）
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ,

  -- 文字数制約
  CONSTRAINT ck_avatar_name_len CHECK (char_length(avatar_name) <= 50),
  CONSTRAINT ck_bio_len         CHECK (bio IS NULL OR char_length(bio) <= 1000),

  -- 時系列整合
  CONSTRAINT ck_avatars_time_order
    CHECK (updated_at >= created_at AND (deleted_at IS NULL OR deleted_at >= created_at)),

  -- 外部キー（walletsはwallet_address TEXT PRIMARY KEY）
  CONSTRAINT fk_avatar_wallet_address
    FOREIGN KEY (wallet_address) REFERENCES wallets(wallet_address) ON DELETE SET NULL
);

-- よく使う検索向けインデックス
CREATE INDEX IF NOT EXISTS idx_avatars_user_id        ON avatars(user_id);
CREATE INDEX IF NOT EXISTS idx_avatars_wallet_address ON avatars(wallet_address);
CREATE INDEX IF NOT EXISTS idx_avatars_deleted_at     ON avatars(deleted_at);
