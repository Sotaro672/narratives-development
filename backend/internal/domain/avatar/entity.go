// backend/internal/domain/avatar/entity.go
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
//
// ✅ avatar_create.dart の入力を正として:
// - アバターアイコン画像 → AvatarIconURL / AvatarIconPath（保存先/URL）
// - アバター名           → AvatarName
// - プロフィール         → Profile
// - 外部リンク           → ExternalLink
//
// それ以外（AvatarState / WalletAddress / timestamps / deleted）はシステム側で管理され得るため保持します。
type Avatar struct {
	ID             string                  `json:"id"`
	UserID         string                  `json:"userId"`
	AvatarName     string                  `json:"avatarName"`
	AvatarIconURL  *string                 `json:"avatarIconUrl,omitempty"`  // 表示用URL（例: GCS署名付き/公開URL）
	AvatarIconPath *string                 `json:"avatarIconPath,omitempty"` // 保存パス（例: gs://bucket/path or avatars/...）
	AvatarState    avatarstate.AvatarState `json:"avatarState"`              // avatarState パッケージの型を使用
	WalletAddress  *string                 `json:"walletAddress,omitempty"`  // wallet ドメインへの外部キー（UIに表示しない前提）
	Profile        *string                 `json:"profile,omitempty"`
	ExternalLink   *string                 `json:"externalLink,omitempty"`
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
	DeletedAt      *time.Time              `json:"deletedAt,omitempty"` // null 許容
}

// Policy
var (
	MaxAvatarNameLength   = 50
	MaxProfileLength      = 1000
	MaxExternalLinkLength = 2048
	MaxIconURLLength      = 2048
	MaxIconPathLength     = 2048
)

// Errors
var (
	ErrInvalidID           = errors.New("avatar: invalid id")
	ErrInvalidUserID       = errors.New("avatar: invalid userId")
	ErrInvalidAvatarName   = errors.New("avatar: invalid avatarName")
	ErrInvalidProfile      = errors.New("avatar: invalid profile")
	ErrInvalidExternalLink = errors.New("avatar: invalid externalLink")
	ErrInvalidIconURL      = errors.New("avatar: invalid avatarIconUrl")
	ErrInvalidIconPath     = errors.New("avatar: invalid avatarIconPath")
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
	iconURL, iconPath, walletAddr, profile, externalLink *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Avatar, error) {
	a := Avatar{
		ID:             strings.TrimSpace(id),
		UserID:         strings.TrimSpace(userID),
		AvatarName:     strings.TrimSpace(avatarName),
		AvatarIconURL:  normalizePtr(iconURL),
		AvatarIconPath: normalizePtr(iconPath),
		Profile:        normalizePtr(profile),
		ExternalLink:   normalizePtr(externalLink),
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
		AvatarState:    state,
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
	iconURL, iconPath, walletAddr, profile, externalLink *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Avatar, error) {
	return NewWithState(
		id, userID, avatarName,
		avatarstate.AvatarState{},
		iconURL, iconPath, walletAddr, profile, externalLink,
		createdAt, updatedAt, deletedAt,
	)
}

// NewForCreateWithState は作成用（now を使い回す）コンストラクタです。
func NewForCreateWithState(
	id string,
	state avatarstate.AvatarState,
	input struct {
		UserID         string
		AvatarName     string
		AvatarIconURL  *string
		AvatarIconPath *string
		WalletAddr     *string
		Profile        *string
		ExternalLink   *string
	},
	now time.Time,
) (Avatar, error) {
	now = now.UTC()
	return NewWithState(
		id,
		input.UserID,
		input.AvatarName,
		state,
		input.AvatarIconURL,
		input.AvatarIconPath,
		input.WalletAddr,
		input.Profile,
		input.ExternalLink,
		now,
		now,
		nil,
	)
}

// 既存互換: 旧 NewForCreate は AvatarState を与えずゼロ値のまま
func NewForCreate(
	id string,
	input struct {
		UserID         string
		AvatarName     string
		AvatarIconURL  *string
		AvatarIconPath *string
		WalletAddr     *string
		Profile        *string
		ExternalLink   *string
	},
	now time.Time,
) (Avatar, error) {
	return NewForCreateWithState(id, avatarstate.AvatarState{}, input, now)
}

// NewFromStringTimesWithState parses times and delegates to NewWithState.
func NewFromStringTimesWithState(
	id, userID, avatarName string,
	state avatarstate.AvatarState,
	iconURL, iconPath, walletAddr, profile, externalLink *string,
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

	return NewWithState(
		id, userID, avatarName,
		state,
		iconURL, iconPath, walletAddr, profile, externalLink,
		ct, ut, dtPtr,
	)
}

// 既存互換: 旧 NewFromStringTimes は AvatarState を与えずゼロ値のまま
func NewFromStringTimes(
	id, userID, avatarName string,
	iconURL, iconPath, walletAddr, profile, externalLink *string,
	createdAt, updatedAt string,
	deletedAt *string,
) (Avatar, error) {
	return NewFromStringTimesWithState(
		id, userID, avatarName,
		avatarstate.AvatarState{},
		iconURL, iconPath, walletAddr, profile, externalLink,
		createdAt, updatedAt, deletedAt,
	)
}

// Mutators

func (a *Avatar) SetIconURL(v *string) error {
	v = normalizePtr(v)
	if v != nil {
		if len([]rune(*v)) > MaxIconURLLength {
			return ErrInvalidIconURL
		}
		// URLとして妥当であること（http/httpsのみ）
		if err := validateExternalLink(*v); err != nil {
			return ErrInvalidIconURL
		}
	}
	a.AvatarIconURL = v
	return nil
}

func (a *Avatar) SetIconPath(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxIconPathLength {
		return ErrInvalidIconPath
	}
	a.AvatarIconPath = v
	return nil
}

func (a *Avatar) SetWalletAddress(v *string) error {
	a.WalletAddress = normalizePtr(v)
	return nil
}

func (a *Avatar) SetProfile(v *string) error {
	v = normalizePtr(v)
	if v != nil && len([]rune(*v)) > MaxProfileLength {
		return ErrInvalidProfile
	}
	a.Profile = v
	return nil
}

func (a *Avatar) SetExternalLink(v *string) error {
	v = normalizePtr(v)
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

	if a.AvatarIconURL != nil {
		if len([]rune(*a.AvatarIconURL)) > MaxIconURLLength {
			return ErrInvalidIconURL
		}
		if err := validateExternalLink(*a.AvatarIconURL); err != nil {
			return ErrInvalidIconURL
		}
	}
	if a.AvatarIconPath != nil && len([]rune(*a.AvatarIconPath)) > MaxIconPathLength {
		return ErrInvalidIconPath
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

// External link validator (http/https only)
func validateExternalLink(s string) error {
	s = strings.TrimSpace(s)
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
