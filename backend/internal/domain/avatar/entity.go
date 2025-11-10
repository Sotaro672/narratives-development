// backend\internal\domain\avatar\entity.go
package avatar

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	avatarstate "narratives/internal/domain/avatarState"
	userdom "narratives/internal/domain/user"
	walletdom "narratives/internal/domain/wallet"
)

// Avatar - ドメインエンティティ
// web 側の Avatar に準拠しつつ、avatarState は別ドメインのエンティティ型を参照します。
type Avatar struct {
	ID            string                  `json:"id"`
	UserID        string                  `json:"userId"`
	AvatarName    string                  `json:"avatarName"`
	AvatarIconID  *string                 `json:"avatarIconId,omitempty"`
	AvatarState   avatarstate.AvatarState `json:"avatarState"`             // avatarState パッケージの型を使用
	WalletAddress *string                 `json:"walletAddress,omitempty"` // wallet ドメインへの外部キー
	Bio           *string                 `json:"bio,omitempty"`
	Website       *string                 `json:"website,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
	DeletedAt     *time.Time              `json:"deletedAt,omitempty"` // null 許容
}

// Policy
var (
	MaxAvatarNameLength = 50
	MaxBioLength        = 1000
)

// Errors
var (
	ErrInvalidID         = errors.New("avatar: invalid id")
	ErrInvalidUserID     = errors.New("avatar: invalid userId")
	ErrInvalidAvatarName = errors.New("avatar: invalid avatarName")
	ErrInvalidBio        = errors.New("avatar: invalid bio")
	ErrInvalidWebsite    = errors.New("avatar: invalid website")
	ErrInvalidCreatedAt  = errors.New("avatar: invalid createdAt")
	ErrInvalidUpdatedAt  = errors.New("avatar: invalid updatedAt")
	ErrInvalidDeletedAt  = errors.New("avatar: invalid deletedAt")
	// Link errors
	ErrInvalidWalletAddressLink = errors.New("avatar: invalid walletAddress link")
)

// Constructors

// NewWithState は AvatarState(別ドメイン型) を含む新しいコンストラクタです。
func NewWithState(
	id, userID, avatarName string,
	state avatarstate.AvatarState,
	iconID, walletAddr, bio, website *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Avatar, error) {
	a := Avatar{
		ID:           strings.TrimSpace(id),
		UserID:       strings.TrimSpace(userID),
		AvatarName:   strings.TrimSpace(avatarName),
		AvatarIconID: normalizePtr(iconID),
		Bio:          normalizePtr(bio),
		Website:      normalizePtr(website),
		CreatedAt:    createdAt.UTC(),
		UpdatedAt:    updatedAt.UTC(),
		AvatarState:  state,
	}
	// walletAddress はオプショナル
	a.WalletAddress = normalizePtr(walletAddr)

	// DeletedAt はオプショナル（nil 可）
	if deletedAt != nil && !deletedAt.IsZero() {
		t := deletedAt.UTC()
		a.DeletedAt = &t
	}

	// Validation
	if err := a.validate(); err != nil {
		return Avatar{}, err
	}
	return a, nil
}

// 既存互換: 旧 New は AvatarState を与えず NewWithState を呼びます（ゼロ値のまま）。
func New(
	id, userID, avatarName string,
	iconID, walletAddr, bio, website *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Avatar, error) {
	return NewWithState(
		id, userID, avatarName,
		avatarstate.AvatarState{}, // フィールドはゼロ値のまま（必要に応じて後で別ユースケースで設定）
		iconID, walletAddr, bio, website,
		createdAt, updatedAt, deletedAt,
	)
}

// NewForCreateWithState は作成用（now を使い回す）コンストラクタです。
func NewForCreateWithState(
	id string,
	state avatarstate.AvatarState,
	input struct {
		UserID       string
		AvatarName   string
		AvatarIconID *string
		WalletAddr   *string
		Bio          *string
		Website      *string
	},
	now time.Time,
) (Avatar, error) {
	now = now.UTC()
	return NewWithState(
		id,
		input.UserID,
		input.AvatarName,
		state,
		input.AvatarIconID,
		input.WalletAddr,
		input.Bio,
		input.Website,
		now,
		now,
		nil,
	)
}

// 既存互換: 旧 NewForCreate は AvatarState を与えずゼロ値のまま
func NewForCreate(
	id string,
	input struct {
		UserID       string
		AvatarName   string
		AvatarIconID *string
		WalletAddr   *string
		Bio          *string
		Website      *string
	},
	now time.Time,
) (Avatar, error) {
	return NewForCreateWithState(id, avatarstate.AvatarState{}, input, now)
}

// NewFromStringTimesWithState parses times and delegates to NewWithState.
func NewFromStringTimesWithState(
	id, userID, avatarName string,
	state avatarstate.AvatarState,
	iconID, walletAddr, bio, website *string,
	createdAt, updatedAt string,
	deletedAt *string,
) (Avatar, error) {
	ct, err := parseTime(createdAt)
	if err != nil {
		return Avatar{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return Avatar{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}

	var dtPtr *time.Time
	if deletedAt != nil && strings.TrimSpace(*deletedAt) != "" {
		dt, err := parseTime(*deletedAt)
		if err != nil {
			return Avatar{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, err)
		}
		dtPtr = &dt
	}

	return NewWithState(id, userID, avatarName, state, iconID, walletAddr, bio, website, ct, ut, dtPtr)
}

// 既存互換: 旧 NewFromStringTimes は AvatarState を与えずゼロ値のまま
func NewFromStringTimes(
	id, userID, avatarName string,
	iconID, walletAddr, bio, website *string,
	createdAt, updatedAt string,
	deletedAt *string,
) (Avatar, error) {
	return NewFromStringTimesWithState(
		id, userID, avatarName,
		avatarstate.AvatarState{},
		iconID, walletAddr, bio, website,
		createdAt, updatedAt, deletedAt,
	)
}

// Mutators

func (a *Avatar) SetIconID(v *string) error {
	a.AvatarIconID = normalizePtr(v)
	return nil
}

func (a *Avatar) SetWalletAddress(v *string) error {
	a.WalletAddress = normalizePtr(v)
	return nil
}

func (a *Avatar) SetBio(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxBioLength {
		return ErrInvalidBio
	}
	a.Bio = v
	return nil
}

func (a *Avatar) SetWebsite(v *string) error {
	v = normalizePtr(v)
	if v != nil {
		if err := validateWebsite(*v); err != nil {
			return err
		}
	}
	a.Website = v
	return nil
}

// ステート設定（avatarState ドメインの値をそのまま受け取る）
func (a *Avatar) SetState(state avatarstate.AvatarState) {
	a.AvatarState = state
}

// Mutators (name)
func (a *Avatar) UpdateAvatarName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > MaxAvatarNameLength {
		return ErrInvalidAvatarName
	}
	a.AvatarName = name
	return nil
}

// SetUser は与えられた User を所有者として設定します（User.ID を UserID に反映）
func (a *Avatar) SetUser(u userdom.User) error {
	if a == nil {
		return nil
	}
	id := strings.TrimSpace(u.ID)
	if id == "" {
		return ErrInvalidUserID
	}
	a.UserID = id
	return nil
}

// SetWallet sets the wallet link from a Wallet domain object (uses Wallet.WalletAddress).
func (a *Avatar) SetWallet(w walletdom.Wallet) error {
	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return ErrInvalidWalletAddressLink
	}
	a.WalletAddress = &addr
	return nil
}

// ClearWallet removes the wallet link.
func (a *Avatar) ClearWallet() {
	a.WalletAddress = nil
}

// ValidateUserLink は UserID が有効か（空でないか）を確認します。
// 実在性の確認は上位レイヤー（リポジトリやユースケース）で行ってください。
func (a Avatar) ValidateUserLink() error {
	if strings.TrimSpace(a.UserID) == "" {
		return ErrInvalidUserID
	}
	return nil
}

// ValidateWalletLink ensures WalletAddress is present (existence check is upper layer’s responsibility).
func (a Avatar) ValidateWalletLink() error {
	if a.WalletAddress == nil || strings.TrimSpace(*a.WalletAddress) == "" {
		return ErrInvalidWalletAddressLink
	}
	return nil
}

// Helpers

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// Validation
func (a Avatar) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.UserID == "" {
		return ErrInvalidUserID
	}
	if a.AvatarName == "" || len([]rune(a.AvatarName)) > MaxAvatarNameLength {
		return ErrInvalidAvatarName
	}
	if a.Bio != nil && len([]rune(*a.Bio)) > MaxBioLength {
		return ErrInvalidBio
	}
	if a.Website != nil {
		if err := validateWebsite(*a.Website); err != nil {
			return err
		}
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt.IsZero() || a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.DeletedAt != nil && a.DeletedAt.Before(a.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	return nil
}

// Time parsing helper
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
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
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// Website validator
func validateWebsite(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return ErrInvalidWebsite
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidWebsite
	}
	if u.Host == "" {
		return ErrInvalidWebsite
	}
	return nil
}

// Listing filter/types (used by application usecases)
type SortBy string

const (
	SortByCreatedAt SortBy = "created_at"
	SortByUpdatedAt SortBy = "updated_at"
	SortByName      SortBy = "avatar_name"
)

func IsValidSortBy(s SortBy) bool {
	switch s {
	case SortByCreatedAt, SortByUpdatedAt, SortByName:
		return true
	default:
		return false
	}
}

// ListFilter represents filters and pagination for listing avatars.
type ListFilter struct {
	UserID        *string
	NameContains  string
	WalletAddress *string

	IncludeDeleted bool
	Limit          int
	Offset         int

	SortBy SortBy
	Desc   bool
}

// Sanitize normalizes fields and applies safe defaults.
func (f *ListFilter) Sanitize() {
	if f.UserID != nil {
		v := strings.TrimSpace(*f.UserID)
		if v == "" {
			f.UserID = nil
		} else {
			f.UserID = &v
		}
	}
	f.NameContains = strings.TrimSpace(f.NameContains)
	if f.WalletAddress != nil {
		v := strings.TrimSpace(*f.WalletAddress)
		if v == "" {
			f.WalletAddress = nil
		} else {
			f.WalletAddress = &v
		}
	}

	if f.Limit < 0 {
		f.Limit = 0
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if !IsValidSortBy(f.SortBy) {
		f.SortBy = SortByCreatedAt
	}
}

// DDL
const AvatarsTableDDL = `
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
  CONSTRAINT ck_avatar_name_len  CHECK (char_length(avatar_name) <= 50),
  CONSTRAINT ck_bio_len          CHECK (bio IS NULL OR char_length(bio) <= 1000),

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
`
