// backend/internal/domain/resale_review/repository_port.go
package resale_review

import (
	"context"

	common "narratives/internal/domain/common"
)

// ============================================================
// Patch / Filter
// ============================================================

// PatchResaleReviewAggregate is a partial update model for:
//
// resaleReviews/{resaleId}
//
// Counter updates should be performed atomically with the corresponding
// like/comment write by the Firestore repository whenever possible.
type PatchResaleReviewAggregate struct {
	LikeCount    *int64 `json:"likeCount"`
	CommentCount *int64 `json:"commentCount"`
}

func NewPatchFromResaleReviewAggregate(
	aggregate ResaleReviewAggregate,
) PatchResaleReviewAggregate {
	return PatchResaleReviewAggregate{
		LikeCount:    &aggregate.LikeCount,
		CommentCount: &aggregate.CommentCount,
	}
}

func NewLikeCountPatchFromResaleReviewAggregate(
	aggregate ResaleReviewAggregate,
) PatchResaleReviewAggregate {
	return PatchResaleReviewAggregate{
		LikeCount: &aggregate.LikeCount,
	}
}

func NewCommentCountPatchFromResaleReviewAggregate(
	aggregate ResaleReviewAggregate,
) PatchResaleReviewAggregate {
	return PatchResaleReviewAggregate{
		CommentCount: &aggregate.CommentCount,
	}
}

// FilterLike is used for listing likes.
//
// ResaleID:
// - optional when listing by AvatarID across resale reviews.
//
// AvatarID:
// - optional when listing all likes for one resale.
//
// At least one of ResaleID / AvatarID should normally be specified by
// application.usecase.
type FilterLike struct {
	common.FilterCommon `json:",inline"`
	ResaleID            string `json:"resaleId"`
	AvatarID            string `json:"avatarId"`
}

// FilterComment is used for listing comments.
//
// ResaleID is normally required because comments belong to a resale review.
//
// Deleted:
// - nil   = no deleted-state filter
// - false = visible comments only
// - true  = deleted comments only
type FilterComment struct {
	common.FilterCommon `json:",inline"`
	ResaleID            string `json:"resaleId"`
	AvatarID            string `json:"avatarId"`
	Deleted             *bool  `json:"deleted"`
}

// PatchComment is a partial update model for:
//
// resaleReviews/{resaleId}/comments/{commentId}
//
// UpdateBody and MarkDeleted should first be executed against the domain
// entity, then the resulting values should be persisted using this patch.
type PatchComment struct {
	Body    *string `json:"body"`
	Deleted *bool   `json:"deleted"`
}

func NewContentPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		Body: &comment.Body,
	}
}

func NewDeletionPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		Body:    &comment.Body,
		Deleted: &comment.Deleted,
	}
}

// ============================================================
// Aggregate repository
// ============================================================

// AggregateRepository manages:
//
// resaleReviews/{resaleId}
type AggregateRepository interface {
	GetByID(
		ctx context.Context,
		resaleID string,
	) (ResaleReviewAggregate, error)

	Create(
		ctx context.Context,
		entity ResaleReviewAggregate,
	) (ResaleReviewAggregate, error)

	Update(
		ctx context.Context,
		resaleID string,
		patch PatchResaleReviewAggregate,
	) (ResaleReviewAggregate, error)

	Delete(
		ctx context.Context,
		resaleID string,
	) error
}

// ============================================================
// Like repository
// ============================================================

// LikeRepository manages:
//
// resaleReviews/{resaleId}/likes/{avatarId}
//
// avatarId is the document ID.
// Therefore one avatar can have at most one like for one resale.
type LikeRepository interface {
	// List lists likes by resaleId and/or avatarId.
	//
	// A Firestore implementation may use a collection-group query when
	// ResaleID is empty and AvatarID is specified.
	List(
		ctx context.Context,
		filter FilterLike,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[Like], error)

	// FindByAvatar returns the like made by avatarId for resaleId.
	//
	// ErrNotFound should be returned when the avatar has not liked the resale.
	FindByAvatar(
		ctx context.Context,
		resaleID string,
		avatarID string,
	) (Like, error)

	// ExistsByAvatar returns whether avatarId currently likes resaleId.
	//
	// This is intended for viewer-specific LikedByMe resolution without
	// forcing application.usecase to handle ErrNotFound as normal control flow.
	ExistsByAvatar(
		ctx context.Context,
		resaleID string,
		avatarID string,
	) (bool, error)

	// CreateUnderParent creates:
	//
	// resaleReviews/{resaleId}/likes/{avatarId}
	//
	// Implementations should return ErrConflict when the like already exists.
	CreateUnderParent(
		ctx context.Context,
		resaleID string,
		like Like,
	) (Like, error)

	// DeleteByAvatar physically deletes:
	//
	// resaleReviews/{resaleId}/likes/{avatarId}
	//
	// Unlike is represented by physical deletion.
	DeleteByAvatar(
		ctx context.Context,
		resaleID string,
		avatarID string,
	) error
}

// ============================================================
// Comment repository
// ============================================================

// CommentRepository manages:
//
// resaleReviews/{resaleId}/comments/{commentId}
type CommentRepository interface {
	// List lists comments under a resale review.
	List(
		ctx context.Context,
		filter FilterComment,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[Comment], error)

	// GetByParentID fetches one comment by resaleId + commentId.
	GetByParentID(
		ctx context.Context,
		resaleID string,
		commentID string,
	) (Comment, error)

	// CreateUnderParent creates:
	//
	// resaleReviews/{resaleId}/comments/{commentId}
	CreateUnderParent(
		ctx context.Context,
		resaleID string,
		comment Comment,
	) (Comment, error)

	// UpdateUnderParent updates a comment under resaleId.
	UpdateUnderParent(
		ctx context.Context,
		resaleID string,
		commentID string,
		patch PatchComment,
	) (Comment, error)

	// DeleteUnderParent physically deletes a comment document.
	//
	// Normal user deletion should generally use Comment.MarkDeleted and
	// UpdateUnderParent instead. Physical deletion is primarily intended for
	// resale-review cleanup.
	DeleteUnderParent(
		ctx context.Context,
		resaleID string,
		commentID string,
	) error
}

// ============================================================
// Atomic mutation repository
// ============================================================

// MutationRepository represents operations that must keep the parent aggregate
// counters and child documents consistent.
//
// Firestore implementations should execute each operation in one transaction.
//
// This prevents states such as:
// - like document exists but LikeCount was not incremented
// - like document was deleted but LikeCount was not decremented
// - comment exists but CommentCount was not incremented
// - comment was logically deleted but CommentCount was not decremented
type MutationRepository interface {
	// AddLike creates the avatar like and increments LikeCount atomically.
	//
	// Implementations should return ErrConflict when the like already exists.
	AddLike(
		ctx context.Context,
		like Like,
	) (ResaleReviewAggregate, Like, error)

	// RemoveLike deletes the avatar like and decrements LikeCount atomically.
	//
	// Implementations should keep the operation idempotent where practical:
	// when no like exists, the counter must not be decremented.
	RemoveLike(
		ctx context.Context,
		resaleID string,
		avatarID string,
	) (ResaleReviewAggregate, error)

	// AddComment creates a comment and increments CommentCount atomically.
	AddComment(
		ctx context.Context,
		comment Comment,
	) (ResaleReviewAggregate, Comment, error)

	// MarkCommentDeleted logically deletes a comment and decrements
	// CommentCount atomically.
	//
	// When the comment is already deleted, the counter must not be
	// decremented again.
	MarkCommentDeleted(
		ctx context.Context,
		resaleID string,
		commentID string,
	) (ResaleReviewAggregate, Comment, error)
}

// ============================================================
// Cleanup repository
// ============================================================

// CleanupRepository physically removes all documents belonging to one
// resale review.
//
// Firestore does not automatically delete subcollections when the parent
// document is deleted, so aggregate deletion alone is insufficient.
//
// Implementations should delete:
// - resaleReviews/{resaleId}/likes/*
// - resaleReviews/{resaleId}/comments/*
// - resaleReviews/{resaleId}
type CleanupRepository interface {
	DeleteByResaleID(
		ctx context.Context,
		resaleID string,
	) error
}

// ============================================================
// Composite port
// ============================================================

// RepositoryPort bundles persistence ports for the resale_review domain.
type RepositoryPort interface {
	Aggregates() AggregateRepository
	Likes() LikeRepository
	Comments() CommentRepository
	Mutations() MutationRepository
	Cleanup() CleanupRepository
}
