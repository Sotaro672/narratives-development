// backend\internal\domain\wallet\entity.go
package wallet

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Domain errors
var (
	ErrInvalidWalletAddress = errors.New("wallet: invalid walletAddress")
	ErrInvalidMintAddress   = errors.New("wallet: invalid mintAddress")
	ErrInvalidCreatedAt     = errors.New("wallet: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("wallet: invalid updatedAt")
	ErrInvalidLastUpdatedAt = errors.New("wallet: invalid lastUpdatedAt")
	ErrInvalidStatus        = errors.New("wallet: invalid status")
)

// WalletStatus mirrors TS: 'active' | 'inactive'
type WalletStatus string

const (
	StatusActive   WalletStatus = "active"
	StatusInactive WalletStatus = "inactive"
)

func isValidStatus(s WalletStatus) bool {
	return s == StatusActive || s == StatusInactive
}

// Solana-like base58 address/mint format (approximation).
var base58Re = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

func isValidWallet(s string) bool {
	return base58Re.MatchString(s)
}

func isValidMint(s string) bool {
	return base58Re.MatchString(s)
}

// Wallet mirrors web-app/src/shared/types/wallet.ts
//
//	interface Wallet {
//	  walletAddress: string;
//	  tokens: string[];
//	  lastUpdatedAt: string;
//	  status: 'active' | 'inactive';
//	  createdAt: string;
//	  updatedAt: string;
//	}
type Wallet struct {
	WalletAddress string
	Tokens        []string
	LastUpdatedAt time.Time
	Status        WalletStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// New constructs a Wallet (backward-compatible constructor).
// It sets CreatedAt, UpdatedAt, and LastUpdatedAt to updatedAt, and Status to 'active'.
func New(addr string, tokens []string, updatedAt time.Time) (Wallet, error) {
	w := Wallet{
		WalletAddress: addr,
		Tokens:        nil,
		LastUpdatedAt: updatedAt.UTC(),
		Status:        StatusActive,
		CreatedAt:     updatedAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}
	if err := w.setTokens(tokens); err != nil {
		return Wallet{}, err
	}
	if err := w.validate(); err != nil {
		return Wallet{}, err
	}
	return w, nil
}

// NewFull constructs a Wallet with all fields explicitly provided.
func NewFull(addr string, tokens []string, lastUpdatedAt, createdAt, updatedAt time.Time, status WalletStatus) (Wallet, error) {
	w := Wallet{
		WalletAddress: addr,
		Tokens:        nil,
		LastUpdatedAt: lastUpdatedAt.UTC(),
		Status:        status,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}
	if err := w.setTokens(tokens); err != nil {
		return Wallet{}, err
	}
	if err := w.validate(); err != nil {
		return Wallet{}, err
	}
	return w, nil
}

// NewNow constructs Wallet using current time for CreatedAt/UpdatedAt/LastUpdatedAt.
func NewNow(addr string, tokens []string, status WalletStatus) (Wallet, error) {
	now := time.Now().UTC()
	return NewFull(addr, tokens, now, now, now, status)
}

// NewFromStringTime accepts lastUpdatedAt as string (ISO8601). Status becomes 'active' and created/updated are set to lastUpdatedAt.
func NewFromStringTime(addr string, tokens []string, lastUpdatedAt string) (Wallet, error) {
	t, err := parseTime(lastUpdatedAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidLastUpdatedAt, err)
	}
	return New(addr, tokens, t)
}

// NewFromStringTimes accepts ISO8601 strings for created/updated/lastUpdated and status.
func NewFromStringTimes(addr string, tokens []string, lastUpdatedAt, createdAt, updatedAt, status string) (Wallet, error) {
	lut, err := parseTime(lastUpdatedAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidLastUpdatedAt, err)
	}
	ct, err := parseTime(createdAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	ws := WalletStatus(status)
	if !isValidStatus(ws) {
		return Wallet{}, ErrInvalidStatus
	}
	return NewFull(addr, tokens, lut, ct, ut, ws)
}

// Behavior

// AddToken appends a mint if not present and updates UpdatedAt/LastUpdatedAt.
func (w *Wallet) AddToken(mint string, now time.Time) error {
	if !isValidMint(mint) {
		return ErrInvalidMintAddress
	}
	if !contains(w.Tokens, mint) {
		w.Tokens = append(w.Tokens, mint)
		w.touch(now)
	}
	return nil
}

// RemoveToken removes a mint if present and updates UpdatedAt/LastUpdatedAt when changed.
func (w *Wallet) RemoveToken(mint string, now time.Time) bool {
	if mint == "" {
		return false
	}
	before := len(w.Tokens)
	w.Tokens = remove(w.Tokens, mint)
	changed := len(w.Tokens) != before
	if changed {
		w.touch(now)
	}
	return changed
}

// ReplaceTokens replaces and validates the token set, deduplicated.
func (w *Wallet) ReplaceTokens(tokens []string, now time.Time) error {
	if err := w.setTokens(tokens); err != nil {
		return err
	}
	w.touch(now)
	return nil
}

func (w *Wallet) HasToken(mint string) bool {
	return contains(w.Tokens, mint)
}

// SetStatus updates status and UpdatedAt.
func (w *Wallet) SetStatus(s WalletStatus, now time.Time) error {
	if !isValidStatus(s) {
		return ErrInvalidStatus
	}
	w.Status = s
	if now.IsZero() {
		now = time.Now().UTC()
	}
	w.UpdatedAt = now.UTC()
	return nil
}

// Validation and helpers

func (w Wallet) validate() error {
	if !isValidWallet(w.WalletAddress) {
		return ErrInvalidWalletAddress
	}
	if w.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if w.UpdatedAt.IsZero() || w.UpdatedAt.Before(w.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if w.LastUpdatedAt.IsZero() || w.LastUpdatedAt.Before(w.CreatedAt) {
		return ErrInvalidLastUpdatedAt
	}
	if !isValidStatus(w.Status) {
		return ErrInvalidStatus
	}
	for _, t := range w.Tokens {
		if !isValidMint(t) {
			return ErrInvalidMintAddress
		}
	}
	return nil
}

func (w *Wallet) setTokens(tokens []string) error {
	d := dedup(tokens)
	for _, t := range d {
		if !isValidMint(t) {
			return ErrInvalidMintAddress
		}
	}
	w.Tokens = d
	return nil
}

func (w *Wallet) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	w.LastUpdatedAt = now
	w.UpdatedAt = now
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func remove(xs []string, v string) []string {
	out := xs[:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func dedup(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, ErrInvalidUpdatedAt
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// WalletsTableDDL defines the SQL for the wallets tables migration.
const WalletsTableDDL = `
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
`
