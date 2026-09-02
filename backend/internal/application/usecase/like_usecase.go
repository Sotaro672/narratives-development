// backend/internal/application/usecase/like_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	likedom "narratives/internal/domain/like"
	listdom "narratives/internal/domain/list"
	resaledom "narratives/internal/domain/resale"
)

var (
	ErrLikeRepositoryMissing       = errors.New("like: repository is not configured")
	ErrLikeListRepositoryMissing   = errors.New("like: list repository is not configured")
	ErrLikeResaleRepositoryMissing = errors.New("like: resale repository is not configured")
)

// LikeListGetter resolves a primary-sale List target.
//
// A full list.Repository satisfies this interface.
type LikeListGetter interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// LikeResaleGetter resolves a secondary-market Resale target.
//
// A full resale.Repository satisfies this interface.
type LikeResaleGetter interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

type LikeUsecase struct {
	likeRepo   likedom.Repository
	listRepo   LikeListGetter
	resaleRepo LikeResaleGetter
	now        func() time.Time
}

func NewLikeUsecase(
	likeRepo likedom.Repository,
	listRepo LikeListGetter,
	resaleRepo LikeResaleGetter,
	now func() time.Time,
) *LikeUsecase {
	if now == nil {
		now = time.Now
	}

	return &LikeUsecase{
		likeRepo:   likeRepo,
		listRepo:   listRepo,
		resaleRepo: resaleRepo,
		now:        now,
	}
}

// LikeStatus represents viewer-specific favorite state.
//
// This is not persisted. It is intended for responses such as:
//
// GET    /mall/me/likes/list/{listId}
// PUT    /mall/me/likes/list/{listId}
// DELETE /mall/me/likes/list/{listId}
//
// and the equivalent resale routes.
type LikeStatus struct {
	TargetType likedom.TargetType `json:"targetType"`
	TargetID   string             `json:"targetId"`
	Liked      bool               `json:"liked"`
}

// ============================================================
// List
// ============================================================

// ListByAvatarID returns the current avatar's favorites.
//
// An empty filter returns both List and Resale Likes.
//
// Default order:
// - createdAt DESC
//
// Default pagination:
// - page = 1
// - perPage = 20
func (uc *LikeUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	filter likedom.Filter,
	sortSpec likedom.Sort,
	page likedom.Page,
) (likedom.PageResult[likedom.Like], error) {
	if err := uc.requireLikeRepository(); err != nil {
		return likedom.PageResult[likedom.Like]{}, err
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return likedom.PageResult[likedom.Like]{}, likedom.ErrInvalidAvatarID
	}

	if filter.TargetType != nil && !likedom.IsValidTargetType(*filter.TargetType) {
		return likedom.PageResult[likedom.Like]{}, likedom.ErrInvalidTargetType
	}

	if sortSpec.Column == "" {
		sortSpec.Column = "createdAt"
	}

	if sortSpec.Order == "" {
		sortSpec.Order = likedom.SortDesc
	}

	if page.Number <= 0 {
		page.Number = 1
	}

	if page.PerPage <= 0 {
		page.PerPage = 20
	}

	return uc.likeRepo.ListByAvatarID(
		ctx,
		avatarID,
		filter,
		sortSpec,
		page,
	)
}

// ============================================================
// Status
// ============================================================

// GetStatus returns whether the avatar currently likes the target.
//
// Target existence is intentionally not checked here.
// This keeps status resolution cheap and allows stale favorites to be
// identified even if the underlying target was later suspended or removed.
func (uc *LikeUsecase) GetStatus(
	ctx context.Context,
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (LikeStatus, error) {
	if err := uc.requireLikeRepository(); err != nil {
		return LikeStatus{}, err
	}

	avatarID, targetID, err := normalizeLikeTarget(
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		return LikeStatus{}, err
	}

	exists, err := uc.likeRepo.Exists(
		ctx,
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		return LikeStatus{}, err
	}

	return newLikeStatus(
		targetType,
		targetID,
		exists,
	), nil
}

// ============================================================
// Add
// ============================================================

// Add adds one favorite.
//
// Rules:
// - one avatar can have at most one Like for the same target
// - List target must exist and be listing
// - Resale target must exist and be listing
// - repeated Add is idempotent
func (uc *LikeUsecase) Add(
	ctx context.Context,
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (LikeStatus, error) {
	if err := uc.requireLikeRepository(); err != nil {
		return LikeStatus{}, err
	}

	avatarID, targetID, err := normalizeLikeTarget(
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		return LikeStatus{}, err
	}

	exists, err := uc.likeRepo.Exists(
		ctx,
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		return LikeStatus{}, err
	}

	if exists {
		return newLikeStatus(
			targetType,
			targetID,
			true,
		), nil
	}

	if err := uc.requireLikeableTarget(
		ctx,
		targetType,
		targetID,
	); err != nil {
		return LikeStatus{}, err
	}

	entity, err := likedom.NewLike(
		likedom.NewLikeParams{
			AvatarID:   avatarID,
			TargetType: targetType,
			TargetID:   targetID,
			Now:        uc.nowUTC(),
		},
	)
	if err != nil {
		return LikeStatus{}, err
	}

	_, err = uc.likeRepo.Create(
		ctx,
		*entity,
	)
	if err != nil {
		// Another request may have created the same deterministic document
		// between Exists and Create. PUT semantics remain idempotent.
		if likedom.IsConflict(err) {
			return newLikeStatus(
				targetType,
				targetID,
				true,
			), nil
		}

		return LikeStatus{}, err
	}

	return newLikeStatus(
		targetType,
		targetID,
		true,
	), nil
}

// ============================================================
// Remove
// ============================================================

// Remove removes one favorite.
//
// Target existence is intentionally not required.
// An avatar must be able to remove an old favorite even when the List or
// Resale has since been suspended or deleted.
//
// Repeated Remove is idempotent.
func (uc *LikeUsecase) Remove(
	ctx context.Context,
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (LikeStatus, error) {
	if err := uc.requireLikeRepository(); err != nil {
		return LikeStatus{}, err
	}

	avatarID, targetID, err := normalizeLikeTarget(
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		return LikeStatus{}, err
	}

	if err := uc.likeRepo.Delete(
		ctx,
		avatarID,
		targetType,
		targetID,
	); err != nil {
		if !likedom.IsNotFound(err) {
			return LikeStatus{}, err
		}
	}

	return newLikeStatus(
		targetType,
		targetID,
		false,
	), nil
}

// ============================================================
// Target validation
// ============================================================

// requireLikeableTarget validates that a new Like points to an active target.
//
// Existing favorites are not removed automatically when a target later
// becomes suspended or sold. Cleanup/display policy belongs to the query or
// lifecycle handling layer.
func (uc *LikeUsecase) requireLikeableTarget(
	ctx context.Context,
	targetType likedom.TargetType,
	targetID string,
) error {
	switch targetType {
	case likedom.TargetTypeList:
		return uc.requireLikeableList(
			ctx,
			targetID,
		)

	case likedom.TargetTypeResale:
		return uc.requireLikeableResale(
			ctx,
			targetID,
		)

	default:
		return likedom.ErrInvalidTargetType
	}
}

func (uc *LikeUsecase) requireLikeableList(
	ctx context.Context,
	listID string,
) error {
	if uc == nil || uc.listRepo == nil {
		return ErrLikeListRepositoryMissing
	}

	entity, err := uc.listRepo.GetByID(
		ctx,
		listID,
	)
	if err != nil {
		if errors.Is(err, listdom.ErrNotFound) {
			return likedom.ErrNotFound
		}

		return err
	}

	if entity.Status != listdom.StatusListing {
		return likedom.ErrNotFound
	}

	return nil
}

func (uc *LikeUsecase) requireLikeableResale(
	ctx context.Context,
	resaleID string,
) error {
	if uc == nil || uc.resaleRepo == nil {
		return ErrLikeResaleRepositoryMissing
	}

	entity, err := uc.resaleRepo.GetByID(
		ctx,
		resaleID,
	)
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			return likedom.ErrNotFound
		}

		return err
	}

	if entity.Status != resaledom.StatusListing {
		return likedom.ErrNotFound
	}

	return nil
}

// ============================================================
// Configuration
// ============================================================

func (uc *LikeUsecase) requireLikeRepository() error {
	if uc == nil || uc.likeRepo == nil {
		return ErrLikeRepositoryMissing
	}

	return nil
}

// ============================================================
// Helpers
// ============================================================

func normalizeLikeTarget(
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (string, string, error) {
	avatarID = strings.TrimSpace(avatarID)
	targetID = strings.TrimSpace(targetID)

	if avatarID == "" {
		return "", "", likedom.ErrInvalidAvatarID
	}

	if !likedom.IsValidTargetType(targetType) {
		return "", "", likedom.ErrInvalidTargetType
	}

	if targetID == "" {
		return "", "", likedom.ErrInvalidTargetID
	}

	return avatarID, targetID, nil
}

func newLikeStatus(
	targetType likedom.TargetType,
	targetID string,
	liked bool,
) LikeStatus {
	return LikeStatus{
		TargetType: targetType,
		TargetID:   targetID,
		Liked:      liked,
	}
}

func (uc *LikeUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}
