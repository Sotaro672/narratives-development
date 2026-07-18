// backend/internal/domain/avatar/entity.go
package avatar

import (
	"errors"
	"net/url"
	"time"

	userdom "narratives/internal/domain/user"
	walletdom "narratives/internal/domain/wallet"
)

// Avatar - ドメインエンティティ
//
// avatar_create.dart の入力を正として:
// - アバターアイコン画像 → AvatarIcon
// - アバター名           → AvatarName
// - プロフィール         → Profile
// - 外部リンク           → ExternalLink
//
// WalletAddress / timestamps はシステム側で管理されます。
type Avatar struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`

	AvatarName string `json:"avatarName"`

	// URLとPathを統一して1フィールドにする。
	// 例: "https://..."、"gs://bucket/..."、"avatars/..."
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	WalletAddress *string   `json:"walletAddress,omitempty"`
	Profile       *string   `json:"profile,omitempty"`
	ExternalLink  *string   `json:"externalLink,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SolanaAvatarWallet はAvatar作成時に開設したSolana walletを表します。
//
// 秘密鍵本体ではなく、公開鍵アドレスとSecret Managerの参照先のみを保持します。
type SolanaAvatarWallet struct {
	AvatarID   string `json:"avatarId"`
	Address    string `json:"address"`
	SecretName string `json:"secretName"`
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

	ErrInvalidWalletAddressLink = errors.New("avatar: invalid walletAddress link")
)

// New は永続化済みIDを持つAvatarを生成します。
func New(
	id string,
	userID string,
	avatarName string,
	avatarIcon *string,
	walletAddr *string,
	profile *string,
	externalLink *string,
	createdAt time.Time,
	updatedAt time.Time,
) (Avatar, error) {
	a := Avatar{
		ID:            id,
		UserID:        userID,
		AvatarName:    avatarName,
		AvatarIcon:    avatarIcon,
		WalletAddress: walletAddr,
		Profile:       profile,
		ExternalLink:  externalLink,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}

	if err := a.Validate(); err != nil {
		return Avatar{}, err
	}

	return a, nil
}

// NewAvatarInput はAvatar作成時のDomain入力です。
type NewAvatarInput struct {
	UserID       string
	AvatarName   string
	AvatarIcon   *string
	WalletAddr   *string
	Profile      *string
	ExternalLink *string
}

// NewForCreate はRepositoryでIDが採番される前のAvatarを生成します。
func NewForCreate(
	id string,
	input NewAvatarInput,
	now time.Time,
) (Avatar, error) {
	now = now.UTC()

	a := Avatar{
		ID:            id,
		UserID:        input.UserID,
		AvatarName:    input.AvatarName,
		AvatarIcon:    input.AvatarIcon,
		WalletAddress: input.WalletAddr,
		Profile:       input.Profile,
		ExternalLink:  input.ExternalLink,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// IDは空でもよい。Repository.Createで採番した後にValidateする。
	if err := a.validateForCreate(); err != nil {
		return Avatar{}, err
	}

	return a, nil
}

// NewFromStringTimes はRFC3339文字列を解析してAvatarを生成します。
func NewFromStringTimes(
	id string,
	userID string,
	avatarName string,
	avatarIcon *string,
	walletAddr *string,
	profile *string,
	externalLink *string,
	createdAt string,
	updatedAt string,
) (Avatar, error) {
	ct, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Avatar{}, ErrInvalidCreatedAt
	}

	ut, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Avatar{}, ErrInvalidUpdatedAt
	}

	return New(
		id,
		userID,
		avatarName,
		avatarIcon,
		walletAddr,
		profile,
		externalLink,
		ct.UTC(),
		ut.UTC(),
	)
}

// SetAvatarIcon はAvatarIconを更新します。
func (a *Avatar) SetAvatarIcon(v *string) error {
	if v != nil && len([]rune(*v)) > MaxIconLength {
		return ErrInvalidAvatarIcon
	}

	a.AvatarIcon = v
	return nil
}

// SetWalletAddress はWalletAddressを更新します。
func (a *Avatar) SetWalletAddress(v *string) error {
	a.WalletAddress = v
	return nil
}

// SetProfile はProfileを更新します。
func (a *Avatar) SetProfile(v *string) error {
	if v != nil && len([]rune(*v)) > MaxProfileLength {
		return ErrInvalidProfile
	}

	a.Profile = v
	return nil
}

// SetExternalLink はExternalLinkを更新します。
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

// UpdateAvatarName はAvatarNameを更新します。
func (a *Avatar) UpdateAvatarName(name string) error {
	if name == "" || len([]rune(name)) > MaxAvatarNameLength {
		return ErrInvalidAvatarName
	}

	a.AvatarName = name
	return nil
}

// SetUser はUserを所有者として設定します。
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

// SetWallet はWallet DomainからWalletAddressを設定します。
func (a *Avatar) SetWallet(w walletdom.Wallet) error {
	addr := w.WalletAddress
	if addr == "" {
		return ErrInvalidWalletAddressLink
	}

	a.WalletAddress = &addr
	return nil
}

// ClearWallet はWallet linkを解除します。
func (a *Avatar) ClearWallet() {
	a.WalletAddress = nil
}

// ValidateUserLink はUserIDが設定されていることを確認します。
func (a Avatar) ValidateUserLink() error {
	if a.UserID == "" {
		return ErrInvalidUserID
	}

	return nil
}

// ValidateWalletLink はWalletAddressが設定されていることを確認します。
func (a Avatar) ValidateWalletLink() error {
	if a.WalletAddress == nil || *a.WalletAddress == "" {
		return ErrInvalidWalletAddressLink
	}

	return nil
}

// ApplyPatch はPatchをAvatarのコピーへ適用し、更新後の不変条件を検証します。
func (a Avatar) ApplyPatch(
	patch AvatarPatch,
	updatedAt time.Time,
) (Avatar, error) {
	next := a

	if patch.AvatarName != nil {
		if err := next.UpdateAvatarName(*patch.AvatarName); err != nil {
			return Avatar{}, err
		}
	}

	if patch.AvatarIcon != nil {
		if err := next.SetAvatarIcon(patch.AvatarIcon); err != nil {
			return Avatar{}, err
		}
	}

	if patch.WalletAddress != nil {
		if *patch.WalletAddress == "" {
			return Avatar{}, ErrInvalidWalletAddressLink
		}

		if err := next.SetWalletAddress(patch.WalletAddress); err != nil {
			return Avatar{}, err
		}
	}

	if patch.Profile != nil {
		if err := next.SetProfile(patch.Profile); err != nil {
			return Avatar{}, err
		}
	}

	if patch.ExternalLink != nil {
		if err := next.SetExternalLink(patch.ExternalLink); err != nil {
			return Avatar{}, err
		}
	}

	next.UpdatedAt = updatedAt.UTC()

	if err := next.Validate(); err != nil {
		return Avatar{}, err
	}

	return next, nil
}

// Validate は永続化対象Avatarのすべての不変条件を検証します。
func (a Avatar) Validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}

	return a.validateForCreate()
}

// validateForCreate はID採番前に検証可能な不変条件を確認します。
func (a Avatar) validateForCreate() error {
	if a.UserID == "" {
		return ErrInvalidUserID
	}

	if a.AvatarName == "" ||
		len([]rune(a.AvatarName)) > MaxAvatarNameLength {
		return ErrInvalidAvatarName
	}

	if a.AvatarIcon != nil &&
		len([]rune(*a.AvatarIcon)) > MaxIconLength {
		return ErrInvalidAvatarIcon
	}

	if a.Profile != nil &&
		len([]rune(*a.Profile)) > MaxProfileLength {
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

	if a.UpdatedAt.IsZero() ||
		a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

// validateExternalLink はhttp/https URLを検証します。
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
