// backend/internal/application/usecase/avatar/usecase.go
package avatar

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
	cartdom "narratives/internal/domain/cart"
	walletdom "narratives/internal/domain/wallet"
)

type AvatarUsecase struct {
	avRepo AvatarRepo
	stRepo AvatarStateRepo

	walletSvc  AvatarWalletService
	walletRepo WalletRepo

	cartRepo CartRepo

	now func() time.Time
}

func NewAvatarUsecase(
	avRepo AvatarRepo,
	stRepo AvatarStateRepo,
) *AvatarUsecase {
	return &AvatarUsecase{
		avRepo: avRepo,
		stRepo: stRepo,
		now:    time.Now,
	}
}

func (u *AvatarUsecase) WithNow(now func() time.Time) *AvatarUsecase {
	u.now = now
	return u
}

func (u *AvatarUsecase) WithWalletService(svc AvatarWalletService) *AvatarUsecase {
	u.walletSvc = svc
	return u
}

func (u *AvatarUsecase) WithWalletRepo(r WalletRepo) *AvatarUsecase {
	u.walletRepo = r
	return u
}

func (u *AvatarUsecase) WithCartRepo(r CartRepo) *AvatarUsecase {
	u.cartRepo = r
	return u
}

type AvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	Create(ctx context.Context, a avatardom.Avatar) (avatardom.Avatar, error)
	Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error)
	Delete(ctx context.Context, id string) error
}

type AvatarStateRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
	Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error)
}

type WalletRepo interface {
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type CartRepo interface {
	Upsert(ctx context.Context, c *cartdom.Cart) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error
}

type AvatarWalletService interface {
	OpenAvatarWallet(ctx context.Context, avatarID string) (avatardom.SolanaAvatarWallet, error)
}

type AvatarAggregate struct {
	Avatar avatardom.Avatar
	State  *avatarstate.AvatarState
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

func (u *AvatarUsecase) GetAggregate(ctx context.Context, id string) (AvatarAggregate, error) {
	a, err := u.GetByID(ctx, id)
	if err != nil {
		return AvatarAggregate{}, err
	}

	var stPtr *avatarstate.AvatarState
	if u.stRepo != nil {
		if st, err := u.stRepo.GetByAvatarID(ctx, id); err == nil && st.ID != "" {
			tmp := st
			stPtr = &tmp
		}
	}

	return AvatarAggregate{
		Avatar: a,
		State:  stPtr,
	}, nil
}

var (
	ErrInvalidUserUID             = errors.New("avatar: invalid userUid")
	ErrAvatarWalletAlreadyOpened  = errors.New("avatar: wallet already opened")
	ErrAvatarWalletServiceMissing = errors.New("avatar: wallet service not configured")
	ErrAvatarWalletAddressEmpty   = errors.New("avatar: opened wallet address is empty")
)

func (u *AvatarUsecase) OpenWallet(ctx context.Context, avatarID string) (avatardom.Avatar, error) {
	if avatarID == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}
	if u.walletSvc == nil {
		return avatardom.Avatar{}, ErrAvatarWalletServiceMissing
	}

	a, err := u.avRepo.GetByID(ctx, avatarID)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	if a.WalletAddress != nil && *a.WalletAddress != "" {
		return avatardom.Avatar{}, ErrAvatarWalletAlreadyOpened
	}

	w, err := u.walletSvc.OpenAvatarWallet(ctx, avatarID)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	addr := w.Address
	if addr == "" {
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	patch := avatardom.AvatarPatch{
		WalletAddress: &addr,
	}

	updated, err := u.avRepo.Update(ctx, avatarID, patch)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	if u.walletRepo != nil {
		now := u.now().UTC()
		if wrow, e := walletdom.New(addr, nil, now); e == nil {
			_ = u.walletRepo.Save(ctx, avatarID, wrow)
		}
	}

	return updated, nil
}

func (u *AvatarUsecase) TouchLastActive(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	if avatarID == "" {
		return avatarstate.AvatarState{}, avatardom.ErrInvalidID
	}
	if u.stRepo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}

	now := u.now().UTC()

	state := avatarstate.AvatarState{
		ID:           avatarID,
		LastActiveAt: now,
		UpdatedAt:    &now,
	}

	return u.stRepo.Upsert(ctx, state)
}

func (u *AvatarUsecase) UpdateAvatarState(
	ctx context.Context,
	avatarID string,
	patch avatarstate.AvatarStatePatch,
) (avatarstate.AvatarState, error) {
	if avatarID == "" {
		return avatarstate.AvatarState{}, avatardom.ErrInvalidID
	}
	if u.stRepo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}
	if u.avRepo == nil {
		return avatarstate.AvatarState{}, errors.New("avatar repo not configured")
	}

	if _, err := u.avRepo.GetByID(ctx, avatarID); err != nil {
		return avatarstate.AvatarState{}, err
	}

	now := u.now().UTC()

	current, err := u.stRepo.GetByAvatarID(ctx, avatarID)
	if err != nil && !isAvatarStateNotFound(err) {
		return avatarstate.AvatarState{}, err
	}
	if err != nil && isAvatarStateNotFound(err) {
		current = avatarstate.AvatarState{
			ID:           avatarID,
			LastActiveAt: now,
			UpdatedAt:    &now,
		}
	}
	if current.ID == "" {
		current.ID = avatarID
	}

	followerCount := current.FollowerCount
	followingCount := current.FollowingCount
	postCount := current.PostCount
	followers := cloneAvatarFollowRefs(current.Followers)
	following := cloneAvatarFollowRefs(current.Following)
	lastActiveAt := current.LastActiveAt
	updatedAt := current.UpdatedAt

	if patch.FollowerCount != nil {
		followerCount = patch.FollowerCount
	}
	if patch.FollowingCount != nil {
		followingCount = patch.FollowingCount
	}
	if patch.PostCount != nil {
		postCount = patch.PostCount
	}
	if patch.Followers != nil {
		followers = cloneAvatarFollowRefs(*patch.Followers)
		followerCount = nil
	}
	if patch.Following != nil {
		following = cloneAvatarFollowRefs(*patch.Following)
		followingCount = nil
	}
	if patch.LastActiveAt != nil {
		lastActiveAt = patch.LastActiveAt.UTC()
	}
	if lastActiveAt.IsZero() {
		lastActiveAt = now
	}
	if patch.UpdatedAt != nil {
		t := patch.UpdatedAt.UTC()
		updatedAt = &t
	} else {
		t := now
		updatedAt = &t
	}

	next, nerr := avatarstate.New(
		avatarID,
		followerCount,
		followingCount,
		postCount,
		followers,
		following,
		lastActiveAt,
		updatedAt,
	)
	if nerr != nil {
		return avatarstate.AvatarState{}, nerr
	}

	return u.stRepo.Upsert(ctx, next)
}

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
	if u.stRepo == nil {
		return avatardom.Avatar{}, errors.New("avatarState repo not configured")
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

	name := strings.TrimSpace(in.AvatarName)
	if name == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	var avatarIcon *string
	if in.AvatarIcon != nil {
		s := strings.TrimSpace(*in.AvatarIcon)
		if s != "" {
			if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
				return avatardom.Avatar{}, avatardom.ErrInvalidAvatarIcon
			}
			avatarIcon = &s
		}
	}

	now := u.now().UTC()

	a := avatardom.Avatar{
		UserID:       userUID,
		AvatarName:   name,
		AvatarIcon:   avatarIcon,
		Profile:      in.Profile,
		ExternalLink: in.ExternalLink,
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
		if u.avRepo != nil {
			_ = u.avRepo.Delete(ctx, avatarID)
		}
	}

	zero := int64(0)

	as, aerr := avatarstate.New(
		avatarID,
		&zero,
		&zero,
		&zero,
		[]avatarstate.AvatarFollowRef{},
		[]avatarstate.AvatarFollowRef{},
		now,
		&now,
	)
	if aerr != nil {
		rollback()
		return avatardom.Avatar{}, aerr
	}

	if _, err := u.stRepo.Upsert(ctx, as); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	cart, cerr := cartdom.NewCart(avatarID, nil, now)
	if cerr != nil {
		rollback()
		return avatardom.Avatar{}, cerr
	}

	log.Printf(`[avatar_uc] cart upsert start avatarId=%q`, avatarID)
	if err := u.cartRepo.Upsert(ctx, cart); err != nil {
		log.Printf(`[avatar_uc] cart upsert fail avatarId=%q err=%v`, avatarID, err)
		rollback()
		return avatardom.Avatar{}, err
	}
	log.Printf(`[avatar_uc] cart upsert ok avatarId=%q`, avatarID)

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

	if patch.AvatarIcon != nil {
		s := strings.TrimSpace(*patch.AvatarIcon)
		if s != "" &&
			!strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {
			return avatardom.Avatar{}, avatardom.ErrInvalidAvatarIcon
		}
		patch.AvatarIcon = &s
	}

	return u.avRepo.Update(ctx, id, patch)
}

func (u *AvatarUsecase) Delete(ctx context.Context, avatarID string) error {
	return u.DeleteAvatarCascade(ctx, avatarID)
}

func isAvatarStateNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, avatarstate.ErrNotFound) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

func cloneAvatarFollowRefs(in []avatarstate.AvatarFollowRef) []avatarstate.AvatarFollowRef {
	if len(in) == 0 {
		return []avatarstate.AvatarFollowRef{}
	}

	out := make([]avatarstate.AvatarFollowRef, 0, len(in))
	for _, item := range in {
		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   item.AvatarID,
			FollowedAt: item.FollowedAt.UTC(),
		})
	}

	return out
}
