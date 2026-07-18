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
	avRepo avatardom.Repository

	walletSvc  AvatarWalletService
	walletRepo WalletRepo

	cartRepo AvatarCartRepo

	now func() time.Time
}

func NewAvatarUsecase(
	avRepo avatardom.Repository,
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

type WalletRepo interface {
	Save(
		ctx context.Context,
		avatarID string,
		w walletdom.Wallet,
	) error
}

type AvatarCartRepo interface {
	Upsert(ctx context.Context, c *cartdom.Cart) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error
}

type AvatarWalletService interface {
	OpenAvatarWallet(
		ctx context.Context,
		avatarID string,
	) (avatardom.SolanaAvatarWallet, error)
}

var (
	ErrAvatarRepoNotConfigured = errors.New(
		"avatar: repository not configured",
	)
	ErrWalletRepoNotConfigured = errors.New(
		"avatar: wallet repository not configured",
	)
	ErrAvatarCartRepoNotConfigured = errors.New(
		"avatar: cart repository not configured",
	)
	ErrInvalidUserUID = errors.New(
		"avatar: invalid userUid",
	)
	ErrAvatarWalletServiceMissing = errors.New(
		"avatar: wallet service not configured",
	)
	ErrAvatarWalletAddressEmpty = errors.New(
		"avatar: opened wallet address is empty",
	)
)

func (u *AvatarUsecase) GetByID(
	ctx context.Context,
	id string,
) (avatardom.Avatar, error) {
	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	if u == nil || u.avRepo == nil {
		return avatardom.Avatar{}, ErrAvatarRepoNotConfigured
	}

	return u.avRepo.GetByID(ctx, id)
}

func (u *AvatarUsecase) DeleteAvatarCascade(
	ctx context.Context,
	avatarID string,
) error {
	if avatarID == "" {
		return avatardom.ErrInvalidID
	}

	if u == nil || u.avRepo == nil {
		return ErrAvatarRepoNotConfigured
	}

	if u.cartRepo != nil {
		_ = u.cartRepo.DeleteByAvatarID(ctx, avatarID)
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

func (u *AvatarUsecase) Create(
	ctx context.Context,
	in CreateAvatarInput,
) (avatardom.Avatar, error) {
	if u == nil || u.avRepo == nil {
		return avatardom.Avatar{}, ErrAvatarRepoNotConfigured
	}

	if u.walletSvc == nil {
		return avatardom.Avatar{}, ErrAvatarWalletServiceMissing
	}

	if u.walletRepo == nil {
		return avatardom.Avatar{}, ErrWalletRepoNotConfigured
	}

	if u.cartRepo == nil {
		return avatardom.Avatar{}, ErrAvatarCartRepoNotConfigured
	}

	userUID := in.UserUID
	if userUID == "" {
		userUID = in.UserID
	}

	if userUID == "" {
		return avatardom.Avatar{}, ErrInvalidUserUID
	}

	if in.AvatarName == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	var avatarIcon *string
	if in.AvatarIcon != nil {
		s := *in.AvatarIcon

		if s != "" {
			if !strings.HasPrefix(s, "http://") &&
				!strings.HasPrefix(s, "https://") {
				return avatardom.Avatar{},
					avatardom.ErrInvalidAvatarIcon
			}

			avatarIcon = &s
		}
	}

	now := u.now().UTC()

	a, err := avatardom.NewForCreate(
		"",
		avatardom.NewAvatarInput{
			UserID:       userUID,
			AvatarName:   in.AvatarName,
			AvatarIcon:   avatarIcon,
			Profile:      in.Profile,
			ExternalLink: in.ExternalLink,
		},
		now,
	)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	created, err := u.avRepo.Create(ctx, a)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	avatarID := created.ID
	if avatarID == "" {
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

	cart, err := cartdom.NewCart(avatarID, nil, now)
	if err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	if err := u.cartRepo.Upsert(ctx, cart); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	walletResult, err := u.walletSvc.OpenAvatarWallet(
		ctx,
		avatarID,
	)
	if err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	walletAddress := walletResult.Address
	if walletAddress == "" {
		rollback()
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	patch := avatardom.AvatarPatch{
		WalletAddress: &walletAddress,
	}

	updated, err := u.avRepo.Update(ctx, avatarID, patch)
	if err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	walletRow, err := walletdom.New(
		walletAddress,
		nil,
		now,
	)
	if err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	if err := u.walletRepo.Save(
		ctx,
		avatarID,
		walletRow,
	); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	return updated, nil
}

func (u *AvatarUsecase) Update(
	ctx context.Context,
	id string,
	patch avatardom.AvatarPatch,
) (avatardom.Avatar, error) {
	if u == nil || u.avRepo == nil {
		return avatardom.Avatar{}, ErrAvatarRepoNotConfigured
	}

	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	if patch.AvatarName != nil &&
		*patch.AvatarName == "" {
		return avatardom.Avatar{},
			avatardom.ErrInvalidAvatarName
	}

	if patch.AvatarIcon != nil {
		s := *patch.AvatarIcon

		if s != "" &&
			!strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {
			return avatardom.Avatar{},
				avatardom.ErrInvalidAvatarIcon
		}
	}

	current, err := u.avRepo.GetByID(ctx, id)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	if _, err := current.ApplyPatch(
		patch,
		u.now(),
	); err != nil {
		return avatardom.Avatar{}, err
	}

	return u.avRepo.Update(ctx, id, patch)
}

func (u *AvatarUsecase) Delete(
	ctx context.Context,
	avatarID string,
) error {
	return u.DeleteAvatarCascade(ctx, avatarID)
}
