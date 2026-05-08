package avatar

import (
	"errors"
	"net/url"
	"time"

	avatarstate "narratives/internal/domain/avatarState"
	userdom "narratives/internal/domain/user"
	walletdom "narratives/internal/domain/wallet"
)

// Avatar - ドメインエンティティ
//
// ✅ avatar_create.dart の入力を正として:
// - アバターアイコン画像 → AvatarIcon（保存先/URL いずれでも可。アプリ側で統一した値を渡す）
// - アバター名           → AvatarName
// - プロフィール         → Profile
// - 外部リンク           → ExternalLink
//
// それ以外（AvatarState / WalletAddress / timestamps / deleted）はシステム側で管理され得るため保持します。
type Avatar struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`

	AvatarName string `json:"avatarName"`

	// ✅ CHANGED: URL と Path を統一して 1 フィールドに
	// - 例: "https://..." でも "gs://bucket/..." でも "avatars/..." でも可
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	AvatarState   avatarstate.AvatarState `json:"avatarState"`             // avatarState パッケージの型を使用
	WalletAddress *string                 `json:"walletAddress,omitempty"` // wallet ドメインへの外部キー（UIに表示しない前提）
	Profile       *string                 `json:"profile,omitempty"`
	ExternalLink  *string                 `json:"externalLink,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
	DeletedAt     *time.Time              `json:"deletedAt,omitempty"` // null 許容
}

// SolanaAvatarWallet
// Avatar 作成時に Solana ウォレットを開設し、秘密鍵は Secret Manager に保存する設計のため、
// ドメイン上は「公開鍵アドレス」と「秘密鍵参照（Secret の Version 名など）」だけを保持します。
type SolanaAvatarWallet struct {
	AvatarID   string `json:"avatarId"`
	Address    string `json:"address"`    // base58 public key
	SecretName string `json:"secretName"` // projects/<p>/secrets/<s>/versions/<v>
}

// Policy
var (
	MaxAvatarNameLength   = 50
	MaxProfileLength      = 1000
	MaxExternalLinkLength = 2048
	MaxIconLength         = 2048
)

// Errors
var (
	ErrInvalidID           = errors.New("avatar: invalid id")
	ErrInvalidUserID       = errors.New("avatar: invalid userId")
	ErrInvalidAvatarName   = errors.New("avatar: invalid avatarName")
	ErrInvalidProfile      = errors.New("avatar: invalid profile")
	ErrInvalidExternalLink = errors.New("avatar: invalid externalLink")
	ErrInvalidAvatarIcon   = errors.New("avatar: invalid avatarIcon")
	ErrInvalidCreatedAt    = errors.New("avatar: invalid createdAt")
	ErrInvalidUpdatedAt    = errors.New("avatar: invalid updatedAt")
	ErrInvalidDeletedAt    = errors.New("avatar: invalid deletedAt")

	// Link errors
	ErrInvalidWalletAddressLink = errors.New("avatar: invalid walletAddress link")
)

// Constructors

// NewWithState は AvatarState(別ドメイン型) を含む新しいコンストラクタです。
func NewWithState(
	id, userID, avatarName string,
	state avatarstate.AvatarState,
	avatarIcon, walletAddr, profile, externalLink *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Avatar, error) {
	a := Avatar{
		ID:           id,
		UserID:       userID,
		AvatarName:   avatarName,
		AvatarIcon:   avatarIcon,
		Profile:      profile,
		ExternalLink: externalLink,
		CreatedAt:    createdAt.UTC(),
		UpdatedAt:    updatedAt.UTC(),
		AvatarState:  state,
	}

	// walletAddress はオプショナル
	a.WalletAddress = walletAddr

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

// NewForCreateWithState は作成用（now を使い回す）コンストラクタです。
func NewForCreateWithState(
	id string,
	state avatarstate.AvatarState,
	input struct {
		UserID       string
		AvatarName   string
		AvatarIcon   *string
		WalletAddr   *string
		Profile      *string
		ExternalLink *string
	},
	now time.Time,
) (Avatar, error) {
	now = now.UTC()
	return NewWithState(
		id,
		input.UserID,
		input.AvatarName,
		state,
		input.AvatarIcon,
		input.WalletAddr,
		input.Profile,
		input.ExternalLink,
		now,
		now,
		nil,
	)
}

// NewFromStringTimesWithState parses times and delegates to NewWithState.
func NewFromStringTimesWithState(
	id, userID, avatarName string,
	state avatarstate.AvatarState,
	avatarIcon, walletAddr, profile, externalLink *string,
	createdAt, updatedAt string,
	deletedAt *string,
) (Avatar, error) {
	ct, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Avatar{}, ErrInvalidCreatedAt
	}
	ut, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Avatar{}, ErrInvalidUpdatedAt
	}

	var dtPtr *time.Time
	if deletedAt != nil && *deletedAt != "" {
		dt, err := time.Parse(time.RFC3339, *deletedAt)
		if err != nil {
			return Avatar{}, ErrInvalidDeletedAt
		}
		dtUTC := dt.UTC()
		dtPtr = &dtUTC
	}

	return NewWithState(
		id,
		userID,
		avatarName,
		state,
		avatarIcon,
		walletAddr,
		profile,
		externalLink,
		ct.UTC(),
		ut.UTC(),
		dtPtr,
	)
}

// Mutators

func (a *Avatar) SetAvatarIcon(v *string) error {
	if v != nil && len([]rune(*v)) > MaxIconLength {
		return ErrInvalidAvatarIcon
	}
	a.AvatarIcon = v
	return nil
}

func (a *Avatar) SetWalletAddress(v *string) error {
	a.WalletAddress = v
	return nil
}

func (a *Avatar) SetProfile(v *string) error {
	if v != nil && len([]rune(*v)) > MaxProfileLength {
		return ErrInvalidProfile
	}
	a.Profile = v
	return nil
}

func (a *Avatar) SetExternalLink(v *string) error {
	if v != nil {
		if len([]rune(*v)) > MaxExternalLinkLength {
			return ErrInvalidExternalLink
		}
		if err := validateExternalLink(*v); err != nil {
			return err
		}
	}
	a.ExternalLink = v
	return nil
}

// ステート設定（avatarState ドメインの値をそのまま受け取る）
func (a *Avatar) SetState(state avatarstate.AvatarState) {
	a.AvatarState = state
}

// Mutators (name)
func (a *Avatar) UpdateAvatarName(name string) error {
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
	id := u.ID
	if id == "" {
		return ErrInvalidUserID
	}
	a.UserID = id
	return nil
}

// SetWallet sets the wallet link from a Wallet domain object (uses Wallet.WalletAddress).
func (a *Avatar) SetWallet(w walletdom.Wallet) error {
	addr := w.WalletAddress
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
	if a.UserID == "" {
		return ErrInvalidUserID
	}
	return nil
}

// ValidateWalletLink ensures WalletAddress is present (existence check is upper layer’s responsibility).
func (a Avatar) ValidateWalletLink() error {
	if a.WalletAddress == nil || *a.WalletAddress == "" {
		return ErrInvalidWalletAddressLink
	}
	return nil
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

	if a.AvatarIcon != nil && len([]rune(*a.AvatarIcon)) > MaxIconLength {
		return ErrInvalidAvatarIcon
	}

	if a.Profile != nil && len([]rune(*a.Profile)) > MaxProfileLength {
		return ErrInvalidProfile
	}
	if a.ExternalLink != nil {
		if len([]rune(*a.ExternalLink)) > MaxExternalLinkLength {
			return ErrInvalidExternalLink
		}
		if err := validateExternalLink(*a.ExternalLink); err != nil {
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

// External link validator (http/https only)
func validateExternalLink(s string) error {
	if s == "" {
		return nil
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return ErrInvalidExternalLink
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidExternalLink
	}
	if u.Host == "" {
		return ErrInvalidExternalLink
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
		v := *f.UserID
		if v == "" {
			f.UserID = nil
		} else {
			f.UserID = &v
		}
	}
	if f.WalletAddress != nil {
		v := *f.WalletAddress
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
