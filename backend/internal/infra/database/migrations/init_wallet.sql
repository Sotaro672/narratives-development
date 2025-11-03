
-- Wallets DDL generated from domain/wallet entity.

-- メイン
CREATE TABLE IF NOT EXISTS wallets (
  wallet_address  TEXT PRIMARY KEY,                    -- Solana等のウォレットアドレス
  tokens          TEXT[]      NOT NULL DEFAULT '{}',   -- 所有ミント（重複はアプリ層で排除）
  status          TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- 形式バリデーション（必要に応じて緩め/外してください）
  CONSTRAINT ck_wallet_address_format
    CHECK (wallet_address ~ '^[1-9A-HJ-NP-Za-km-z]{32,44}$'),

  -- 空文字トークンの禁止（mint未設定の混入防止）
  CONSTRAINT ck_tokens_no_empty
    CHECK (NOT EXISTS (SELECT 1 FROM unnest(tokens) t(x) WHERE x = '')),

  -- 時系列整合
  CONSTRAINT ck_wallets_time_order
    CHECK (updated_at >= created_at AND last_updated_at >= created_at)
);

-- 配列検索最適化（tokens @> ARRAY['mint'] などに有効）
CREATE INDEX IF NOT EXISTS idx_wallets_tokens_gin       ON wallets USING GIN (tokens);
CREATE INDEX IF NOT EXISTS idx_wallets_last_updated_at  ON wallets(last_updated_at);
CREATE INDEX IF NOT EXISTS idx_wallets_created_at       ON wallets(created_at);
CREATE INDEX IF NOT EXISTS idx_wallets_updated_at       ON wallets(updated_at);
CREATE INDEX IF NOT EXISTS idx_wallets_status           ON wallets(status);

-- ログ
CREATE TABLE IF NOT EXISTS wallet_update_logs (
  log_id BIGSERIAL PRIMARY KEY,
  wallet_address TEXT NOT NULL REFERENCES wallets(wallet_address) ON DELETE CASCADE,
  changed_fields JSONB NOT NULL,                       -- {"tokens":{"old":[...],"new":[...]}, ...}
  updated_by UUID,                                     -- 操作者（任意）
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('CREATE','UPDATE','DELETE'))
);

-- トリガー関数：INSERT/UPDATE時に差分をログへ
CREATE OR REPLACE FUNCTION trg_wallets_update()
RETURNS TRIGGER AS $$
DECLARE
  diff JSONB := '{}'::jsonb;
  tokens_changed BOOLEAN := FALSE;
BEGIN
  IF TG_OP = 'UPDATE' THEN
    -- 差分検出
    tokens_changed := (NEW.tokens IS DISTINCT FROM OLD.tokens);

    IF tokens_changed THEN
      diff := jsonb_set(
               diff, '{tokens}',
               jsonb_build_object('old', COALESCE(to_jsonb(OLD.tokens), '[]'::jsonb),
                                  'new', COALESCE(to_jsonb(NEW.tokens), '[]'::jsonb))
             );
      -- トークン変更時は last_updated_at を進める
      NEW.last_updated_at := NOW();
    END IF;

    -- いずれの更新でも updated_at を進める
    NEW.updated_at := NOW();

    IF diff <> '{}'::jsonb THEN
      INSERT INTO wallet_update_logs(wallet_address, changed_fields, updated_by, updated_at, operation_type)
      VALUES (OLD.wallet_address, diff, current_setting('app.user_id', true)::uuid, NOW(), 'UPDATE');
    END IF;

    RETURN NEW;

  ELSIF TG_OP = 'INSERT' THEN
    -- 監査時刻補完
    IF NEW.created_at IS NULL THEN
      NEW.created_at := NOW();
    END IF;
    IF NEW.updated_at IS NULL THEN
      NEW.updated_at := NEW.created_at;
    END IF;
    IF NEW.last_updated_at IS NULL THEN
      NEW.last_updated_at := NEW.created_at;
    END IF;

    INSERT INTO wallet_update_logs(wallet_address, changed_fields, updated_by, updated_at, operation_type)
    VALUES (NEW.wallet_address, '{}'::jsonb, current_setting('app.user_id', true)::uuid, NEW.created_at, 'CREATE');

    RETURN NEW;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- UPDATEトリガー
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'wallets_update_trg'
  ) THEN
    CREATE TRIGGER wallets_update_trg
    BEFORE UPDATE ON wallets
    FOR EACH ROW
    WHEN (OLD IS DISTINCT FROM NEW)
    EXECUTE FUNCTION trg_wallets_update();
  END IF;
END$$;

-- INSERTトリガー
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'wallets_insert_trg'
  ) THEN
    CREATE TRIGGER wallets_insert_trg
    BEFORE INSERT ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION trg_wallets_update();
  END IF;
END$$;
