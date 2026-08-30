// backend/internal/application/usecase/resale_review_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	avatardom "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
	resalereview "narratives/internal/domain/resale_review"
)

// ResaleReviewAvatarGetter resolves avatar display information.
//
// Authentication identity itself must come from middleware/handler.
// This dependency is used only for enriching comment responses.
type ResaleReviewAvatarGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}

// ResaleReviewCommentListItem is a comment response enriched with
// current avatar display information.
//
// AvatarName / AvatarIcon are not persisted in resaleReviews.
type ResaleReviewCommentListItem struct {
	resalereview.Comment

	AvatarName string `json:"avatarName"`
	AvatarIcon string `json:"avatarIcon"`
}

type ResaleReviewUsecase struct {
	resaleRepo resaledom.Repository
	reviewRepo resalereview.RepositoryPort
	avatarRepo ResaleReviewAvatarGetter
	now        func() time.Time
}

func NewResaleReviewUsecase(
	resaleRepo resaledom.Repository,
	reviewRepo resalereview.RepositoryPort,
	avatarRepo ResaleReviewAvatarGetter,
	now func() time.Time,
) *ResaleReviewUsecase {
	if now == nil {
		now = time.Now
	}

	return &ResaleReviewUsecase{
		resaleRepo: resaleRepo,
		reviewRepo: reviewRepo,
		avatarRepo: avatarRepo,
		now:        now,
	}
}

// ============================================================
// Summary
// ============================================================

// GetSummary returns viewer-specific interaction state.
//
// Rules:
// - target resale must exist
// - only listing resale exposes interaction state
// - seller can read the summary, but cannot like their own resale
// - missing resaleReview aggregate is treated as zero interactions
func (uc *ResaleReviewUsecase) GetSummary(
	ctx context.Context,
	resaleID string,
	viewerAvatarID string,
) (resalereview.InteractionSummary, error) {
	if err := uc.requireConfigured("ResaleReview.GetSummary"); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if resaleID == "" {
		return resalereview.InteractionSummary{}, resalereview.ErrInvalidResaleID
	}

	if viewerAvatarID == "" {
		return resalereview.InteractionSummary{}, resalereview.ErrInvalidAvatarID
	}

	if _, err := uc.requireListingResale(ctx, resaleID); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	aggregate, err := uc.reviewRepo.Aggregates().GetByID(ctx, resaleID)
	if err != nil {
		if !resalereview.IsNotFound(err) {
			return resalereview.InteractionSummary{}, err
		}

		return resalereview.NewInteractionSummary(
			resaleID,
			0,
			0,
			false,
		)
	}

	likedByMe, err := uc.reviewRepo.Likes().ExistsByAvatar(
		ctx,
		resaleID,
		viewerAvatarID,
	)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	return resalereview.NewInteractionSummary(
		resaleID,
		aggregate.LikeCount,
		aggregate.CommentCount,
		likedByMe,
	)
}

// ============================================================
// Like
// ============================================================

// AddLike adds one like from avatarID.
//
// Rules:
// - resale must be listing
// - avatar cannot like its own resale
// - one avatar can like one resale only once
func (uc *ResaleReviewUsecase) AddLike(
	ctx context.Context,
	resaleID string,
	avatarID string,
) (resalereview.InteractionSummary, error) {
	if err := uc.requireConfigured("ResaleReview.AddLike"); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if err := resalereview.ValidateLikeTarget(resaleID, avatarID); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	resale, err := uc.requireListingResale(ctx, resaleID)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if resale.AvatarID == avatarID {
		return resalereview.InteractionSummary{}, resalereview.ErrForbidden
	}

	like, err := resalereview.NewLike(
		resaleID,
		avatarID,
		uc.nowUTC(),
	)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	aggregate, _, err := uc.reviewRepo.Mutations().AddLike(ctx, *like)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	return resalereview.NewInteractionSummary(
		resaleID,
		aggregate.LikeCount,
		aggregate.CommentCount,
		true,
	)
}

// RemoveLike removes the current avatar's like.
//
// This operation is idempotent at repository level: if the like does not
// exist, LikeCount is not decremented.
func (uc *ResaleReviewUsecase) RemoveLike(
	ctx context.Context,
	resaleID string,
	avatarID string,
) (resalereview.InteractionSummary, error) {
	if err := uc.requireConfigured("ResaleReview.RemoveLike"); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if err := resalereview.ValidateLikeTarget(resaleID, avatarID); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	resale, err := uc.requireListingResale(ctx, resaleID)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if resale.AvatarID == avatarID {
		return resalereview.InteractionSummary{}, resalereview.ErrForbidden
	}

	aggregate, err := uc.reviewRepo.Mutations().RemoveLike(
		ctx,
		resaleID,
		avatarID,
	)
	if err != nil {
		if resalereview.IsNotFound(err) {
			return resalereview.NewInteractionSummary(
				resaleID,
				0,
				0,
				false,
			)
		}

		return resalereview.InteractionSummary{}, err
	}

	return resalereview.NewInteractionSummary(
		resaleID,
		aggregate.LikeCount,
		aggregate.CommentCount,
		false,
	)
}

// ============================================================
// Comment list
// ============================================================

// ListComments returns visible comments for one resale.
//
// Seller comments and buyer comments use the same Avatar identity model.
// Deleted comments are excluded.
// Historical comments remain readable after suspended / sold.
func (uc *ResaleReviewUsecase) ListComments(
	ctx context.Context,
	resaleID string,
	page common.Page,
) (common.PageResult[ResaleReviewCommentListItem], error) {
	if err := uc.requireConfigured("ResaleReview.ListComments"); err != nil {
		return common.PageResult[ResaleReviewCommentListItem]{}, err
	}

	if resaleID == "" {
		return common.PageResult[ResaleReviewCommentListItem]{}, resalereview.ErrInvalidResaleID
	}

	if _, err := uc.requireExistingResale(ctx, resaleID); err != nil {
		return common.PageResult[ResaleReviewCommentListItem]{}, err
	}

	if page.Number <= 0 {
		page.Number = 1
	}

	if page.PerPage <= 0 {
		page.PerPage = 20
	}

	visible := false

	result, err := uc.reviewRepo.Comments().List(
		ctx,
		resalereview.FilterComment{
			ResaleID: resaleID,
			Deleted:  &visible,
		},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		page,
	)
	if err != nil {
		return common.PageResult[ResaleReviewCommentListItem]{}, err
	}

	items := make([]ResaleReviewCommentListItem, 0, len(result.Items))
	avatarCache := make(map[string]resaleReviewAvatarDisplay, 8)

	for _, comment := range result.Items {
		display := uc.resolveAvatarDisplay(
			ctx,
			comment.AvatarID,
			avatarCache,
		)

		items = append(items, ResaleReviewCommentListItem{
			Comment:    comment,
			AvatarName: display.Name,
			AvatarIcon: display.Icon,
		})
	}

	return common.PageResult[ResaleReviewCommentListItem]{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}, nil
}

// ============================================================
// Create comment
// ============================================================

type CreateResaleReviewCommentInput struct {
	ResaleID string
	AvatarID string
	Body     string
}

func (uc *ResaleReviewUsecase) CreateComment(
	ctx context.Context,
	input CreateResaleReviewCommentInput,
) (ResaleReviewCommentListItem, resalereview.InteractionSummary, error) {
	if err := uc.requireConfigured("ResaleReview.CreateComment"); err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	resaleID := input.ResaleID
	avatarID := input.AvatarID

	if resaleID == "" {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, resalereview.ErrInvalidResaleID
	}

	if avatarID == "" {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, resalereview.ErrInvalidAvatarID
	}

	resale, err := uc.requireListingResale(ctx, resaleID)
	if err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	comment, err := resalereview.NewComment(
		resalereview.NewCommentParams{
			ResaleID: resaleID,
			AvatarID: avatarID,
			Kind:     resalereview.CommentKindUser,
			Body:     input.Body,
			IsRead:   resale.AvatarID == avatarID,
			Now:      uc.nowUTC(),
		},
	)
	if err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	aggregate, created, err := uc.reviewRepo.Mutations().AddComment(
		ctx,
		*comment,
	)
	if err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	display := uc.resolveAvatarDisplay(
		ctx,
		created.AvatarID,
		nil,
	)

	likedByMe, err := uc.reviewRepo.Likes().ExistsByAvatar(
		ctx,
		resaleID,
		avatarID,
	)
	if err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	summary, err := resalereview.NewInteractionSummary(
		resaleID,
		aggregate.LikeCount,
		aggregate.CommentCount,
		likedByMe,
	)
	if err != nil {
		return ResaleReviewCommentListItem{}, resalereview.InteractionSummary{}, err
	}

	return ResaleReviewCommentListItem{
		Comment:    created,
		AvatarName: display.Name,
		AvatarIcon: display.Icon,
	}, summary, nil
}

// ============================================================
// Purchase comment
// ============================================================

// CreatePurchaseComment creates a system purchase notification after a resale
// has been sold.
//
// Rules:
// - target resale must exist and already be sold
// - buyerAvatarID identifies the purchasing avatar
// - body is generated from the current avatar name
// - purchase comments are unread for the resale owner when created
// - purchase comments are system events and cannot be deleted
func (uc *ResaleReviewUsecase) CreatePurchaseComment(
	ctx context.Context,
	resaleID string,
	buyerAvatarID string,
) error {
	if err := uc.requireConfigured("ResaleReview.CreatePurchaseComment"); err != nil {
		return err
	}

	if resaleID == "" {
		return resalereview.ErrInvalidResaleID
	}

	if buyerAvatarID == "" {
		return resalereview.ErrInvalidAvatarID
	}

	if uc.avatarRepo == nil {
		return ErrNotSupported("ResaleReview.CreatePurchaseComment.AvatarRepo")
	}

	resale, err := uc.requireExistingResale(ctx, resaleID)
	if err != nil {
		return err
	}

	if resale.Status != resaledom.StatusSold {
		return resalereview.ErrConflict
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, buyerAvatarID)
	if err != nil {
		return err
	}

	comment, err := resalereview.NewComment(
		resalereview.NewCommentParams{
			ResaleID: resaleID,
			AvatarID: buyerAvatarID,
			Kind:     resalereview.CommentKindPurchase,
			Body:     avatar.AvatarName + "が購入しました。",
			IsRead:   false,
			Now:      uc.nowUTC(),
		},
	)
	if err != nil {
		return err
	}

	_, _, err = uc.reviewRepo.Mutations().AddComment(
		ctx,
		*comment,
	)

	return err
}

// ============================================================
// Mark comments read
// ============================================================

// MarkCommentsRead marks all visible unread comments for one resale as read.
//
// Rules:
// - resale must exist
// - resale may be listing, suspended, or sold
// - only the resale owner may mark comments as read
// - repeated calls are idempotent
//
// The returned count is the number of comments newly marked as read.
func (uc *ResaleReviewUsecase) MarkCommentsRead(
	ctx context.Context,
	resaleID string,
	avatarID string,
) (int, error) {
	if err := uc.requireConfigured("ResaleReview.MarkCommentsRead"); err != nil {
		return 0, err
	}

	if resaleID == "" {
		return 0, resalereview.ErrInvalidResaleID
	}

	if avatarID == "" {
		return 0, resalereview.ErrInvalidAvatarID
	}

	resale, err := uc.requireExistingResale(ctx, resaleID)
	if err != nil {
		return 0, err
	}

	if resale.AvatarID != avatarID {
		return 0, resalereview.ErrForbidden
	}

	return uc.reviewRepo.Mutations().MarkCommentsRead(
		ctx,
		resaleID,
	)
}

// ============================================================
// Delete comment
// ============================================================

// DeleteComment logically deletes a comment.
//
// Rules:
// - resale must still be listing
// - only user comments can be deleted
// - only the original comment author may delete the comment
// - a comment already read by the resale owner cannot be deleted
// - another avatar's comment cannot be deleted, including by the resale owner
// - purchase comments cannot be deleted
// - CommentCount is decremented atomically by MutationRepository
func (uc *ResaleReviewUsecase) DeleteComment(
	ctx context.Context,
	resaleID string,
	commentID string,
	avatarID string,
) (resalereview.InteractionSummary, error) {
	if err := uc.requireConfigured("ResaleReview.DeleteComment"); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if resaleID == "" {
		return resalereview.InteractionSummary{}, resalereview.ErrInvalidResaleID
	}

	if commentID == "" {
		return resalereview.InteractionSummary{}, resalereview.ErrInvalidCommentID
	}

	if avatarID == "" {
		return resalereview.InteractionSummary{}, resalereview.ErrInvalidAvatarID
	}

	if _, err := uc.requireListingResale(ctx, resaleID); err != nil {
		return resalereview.InteractionSummary{}, err
	}

	comment, err := uc.reviewRepo.Comments().GetByParentID(
		ctx,
		resaleID,
		commentID,
	)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	if comment.Kind != resalereview.CommentKindUser {
		return resalereview.InteractionSummary{}, resalereview.ErrForbidden
	}

	if comment.AvatarID != avatarID {
		return resalereview.InteractionSummary{}, resalereview.ErrForbidden
	}

	if comment.IsRead {
		return resalereview.InteractionSummary{}, resalereview.ErrConflict
	}

	aggregate, _, err := uc.reviewRepo.Mutations().MarkCommentDeleted(
		ctx,
		resaleID,
		commentID,
	)
	if err != nil {
		return resalereview.InteractionSummary{}, err
	}

	return resalereview.NewInteractionSummary(
		resaleID,
		aggregate.LikeCount,
		aggregate.CommentCount,
		false,
	)
}

// ============================================================
// Cleanup
// ============================================================

// DeleteByResaleID physically removes the whole resaleReview tree.
//
// This should be invoked when a Resale itself is physically deleted.
// Firestore does not automatically remove subcollections when deleting
// the parent resaleReview document.
func (uc *ResaleReviewUsecase) DeleteByResaleID(
	ctx context.Context,
	resaleID string,
) error {
	if err := uc.requireConfigured("ResaleReview.DeleteByResaleID"); err != nil {
		return err
	}

	if resaleID == "" {
		return resalereview.ErrInvalidResaleID
	}

	return uc.reviewRepo.Cleanup().DeleteByResaleID(
		ctx,
		resaleID,
	)
}

// ============================================================
// Internal helpers
// ============================================================

func (uc *ResaleReviewUsecase) requireConfigured(
	operation string,
) error {
	if uc == nil {
		return ErrNotSupported(operation)
	}

	if uc.resaleRepo == nil {
		return ErrNotSupported(operation + ".ResaleRepo")
	}

	if uc.reviewRepo == nil {
		return ErrNotSupported(operation + ".ReviewRepo")
	}

	if uc.reviewRepo.Aggregates() == nil ||
		uc.reviewRepo.Likes() == nil ||
		uc.reviewRepo.Comments() == nil ||
		uc.reviewRepo.Mutations() == nil ||
		uc.reviewRepo.Cleanup() == nil {
		return ErrNotSupported(operation + ".Repository")
	}

	return nil
}

// requireExistingResale verifies that the target resale exists.
//
// This is used for read-only comment access so historical comments remain
// visible after the resale becomes suspended or sold.
func (uc *ResaleReviewUsecase) requireExistingResale(
	ctx context.Context,
	resaleID string,
) (resaledom.Resale, error) {
	if resaleID == "" {
		return resaledom.Resale{}, resalereview.ErrInvalidResaleID
	}

	item, err := uc.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			return resaledom.Resale{}, resalereview.ErrNotFound
		}

		return resaledom.Resale{}, err
	}

	return item, nil
}

// requireListingResale verifies that interaction target exists and is
// currently listed.
//
// Suspended / sold resales remain readable through ListComments, but
// Like / Comment create / delete operations are blocked.
func (uc *ResaleReviewUsecase) requireListingResale(
	ctx context.Context,
	resaleID string,
) (resaledom.Resale, error) {
	item, err := uc.requireExistingResale(ctx, resaleID)
	if err != nil {
		return resaledom.Resale{}, err
	}

	if item.Status != resaledom.StatusListing {
		return resaledom.Resale{}, resalereview.ErrNotFound
	}

	return item, nil
}

type resaleReviewAvatarDisplay struct {
	Name string
	Icon string
}

func (uc *ResaleReviewUsecase) resolveAvatarDisplay(
	ctx context.Context,
	avatarID string,
	cache map[string]resaleReviewAvatarDisplay,
) resaleReviewAvatarDisplay {
	if avatarID == "" || uc == nil || uc.avatarRepo == nil {
		return resaleReviewAvatarDisplay{}
	}

	if cache != nil {
		if value, ok := cache[avatarID]; ok {
			return value
		}
	}

	display := resaleReviewAvatarDisplay{}

	avatar, err := uc.avatarRepo.GetByID(ctx, avatarID)
	if err == nil {
		display.Name = avatar.AvatarName

		if avatar.AvatarIcon != nil {
			display.Icon = *avatar.AvatarIcon
		}
	}

	if cache != nil {
		cache[avatarID] = display
	}

	return display
}

func (uc *ResaleReviewUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}
