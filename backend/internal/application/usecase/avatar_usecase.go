// backend/internal/application/usecase/avatar_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	cartdom "narratives/internal/domain/cart"
	walletdom "narratives/internal/domain/wallet"
)

type AvatarUsecase struct {
	avRepo AvatarRepo

	walletSvc  AvatarWalletService
	walletRepo WalletRepo

	cartRepo AvatarCartRepo

	now func() time.Time
}

func NewAvatarUsecase(
	avRepo AvatarRepo,
	walletSvc AvatarWalletService,
	walletRepo WalletRepo,
	cartRepo AvatarCartRepo,
	now func() time.Time,
) *AvatarUsecase {
	if now == nil {
		now = time.Now
	}

	return &AvatarUsecase{
		avRepo:     avRepo,
		walletSvc:  walletSvc,
		walletRepo: walletRepo,
		cartRepo:   cartRepo,
		now:        now,
	}
}

type AvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	Create(ctx context.Context, a avatardom.Avatar) (avatardom.Avatar, error)
	Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error)
	Delete(ctx context.Context, id string) error
}

type WalletRepo interface {
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type AvatarCartRepo interface {
	Upsert(ctx context.Context, c *cartdom.Cart) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error
}

type AvatarWalletService interface {
	OpenAvatarWallet(ctx context.Context, avatarID string) (avatardom.SolanaAvatarWallet, error)
}

func (u *AvatarUsecase) GetByID(ctx context.Context, id string) (avatardom.Avatar, error) {
	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}

	return u.avRepo.GetByID(ctx, id)
}

var (
	ErrInvalidUserUID             = errors.New("avatar: invalid userUid")
	ErrAvatarWalletServiceMissing = errors.New("avatar: wallet service not configured")
	ErrAvatarWalletAddressEmpty   = errors.New("avatar: opened wallet address is empty")
)

func (u *AvatarUsecase) DeleteAvatarCascade(ctx context.Context, avatarID string) error {
	if avatarID == "" {
		return avatardom.ErrInvalidID
	}

	if u.cartRepo != nil {
		_ = u.cartRepo.DeleteByAvatarID(ctx, avatarID)
	}

	if u.avRepo == nil {
		return errors.New("avatar repo not configured")
	}

	return u.avRepo.Delete(ctx, avatarID)
}

type CreateAvatarInput struct {
	UserID  string `json:"userId"`
	UserUID string `json:"userUid"`

	AvatarName   string  `json:"avatarName"`
	AvatarIcon   *string `json:"avatarIcon,omitempty"`
	Profile      *string `json:"profile,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
}

func (u *AvatarUsecase) Create(ctx context.Context, in CreateAvatarInput) (avatardom.Avatar, error) {
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}
	if u.walletSvc == nil {
		return avatardom.Avatar{}, ErrAvatarWalletServiceMissing
	}
	if u.walletRepo == nil {
		return avatardom.Avatar{}, errors.New("wallet repo not configured")
	}
	if u.cartRepo == nil {
		return avatardom.Avatar{}, errors.New("cart repo not configured")
	}

	userUID := in.UserUID
	if userUID == "" {
		userUID = in.UserID
	}
	if userUID == "" {
		return avatardom.Avatar{}, ErrInvalidUserUID
	}

	name := in.AvatarName
	if name == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	var avatarIcon *string
	if in.AvatarIcon != nil {
		s := *in.AvatarIcon
		if s != "" {
			if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
				return avatardom.Avatar{}, avatardom.ErrInvalidAvatarIcon
			}
			avatarIcon = &s
		}
	}

	profile := in.Profile
	externalLink := in.ExternalLink

	now := u.now().UTC()

	a := avatardom.Avatar{
		UserID:       userUID,
		AvatarName:   name,
		AvatarIcon:   avatarIcon,
		Profile:      profile,
		ExternalLink: externalLink,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := u.avRepo.Create(ctx, a)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	avatarID := created.ID
	if avatarID == "" {
		_ = u.avRepo.Delete(ctx, created.ID)
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	rollback := func() {
		if u.cartRepo != nil {
			_ = u.cartRepo.DeleteByAvatarID(ctx, avatarID)
		}
		if u.avRepo != nil {
			_ = u.avRepo.Delete(ctx, avatarID)
		}
	}

	cart, cerr := cartdom.NewCart(avatarID, nil, now)
	if cerr != nil {
		rollback()
		return avatardom.Avatar{}, cerr
	}

	if err := u.cartRepo.Upsert(ctx, cart); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	w, werr := u.walletSvc.OpenAvatarWallet(ctx, avatarID)
	if werr != nil {
		rollback()
		return avatardom.Avatar{}, werr
	}

	addr := w.Address
	if addr == "" {
		rollback()
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	patch := avatardom.AvatarPatch{
		WalletAddress: &addr,
	}

	updated, uerr := u.avRepo.Update(ctx, avatarID, patch)
	if uerr != nil {
		rollback()
		return avatardom.Avatar{}, uerr
	}

	created = updated

	walletRow, werr2 := walletdom.New(addr, nil, now)
	if werr2 != nil {
		rollback()
		return avatardom.Avatar{}, werr2
	}

	if err := u.walletRepo.Save(ctx, avatarID, walletRow); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	return created, nil
}

func (u *AvatarUsecase) Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error) {
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}

	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	if patch.AvatarName != nil && *patch.AvatarName == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	if patch.AvatarIcon != nil {
		s := *patch.AvatarIcon
		if s != "" &&
			!strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {
			return avatardom.Avatar{}, avatardom.ErrInvalidAvatarIcon
		}
	}

	return u.avRepo.Update(ctx, id, patch)
}

func (u *AvatarUsecase) Delete(ctx context.Context, avatarID string) error {
	return u.DeleteAvatarCascade(ctx, avatarID)
}
