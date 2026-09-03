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
// CommentCount updates should be performed atomically with the corresponding
// comment write by the repository whenever possible.
type PatchResaleReviewAggregate struct {
	CommentCount *int64 `json:"commentCount"`
}

func NewPatchFromResaleReviewAggregate(
	aggregate ResaleReviewAggregate,
) PatchResaleReviewAggregate {
	return PatchResaleReviewAggregate{
		CommentCount: &aggregate.CommentCount,
	}
}

func NewCommentCountPatchFromResaleReviewAggregate(
	aggregate ResaleReviewAggregate,
) PatchResaleReviewAggregate {
	return PatchResaleReviewAggregate{
		CommentCount: &aggregate.CommentCount,
	}
}

// FilterComment is used for listing comments.
//
// ResaleID:
// - non-empty = comments are limited to the specified resale review
// - empty     = comments may be listed across resale reviews when AvatarID is specified
//
// AvatarID:
// - empty     = no avatar filter
// - non-empty = comments created by the specified avatar only
//
// At least one of ResaleID / AvatarID must be specified.
//
// Typical usages:
// - ResaleID only       = all comments for one resale
// - ResaleID + AvatarID = one avatar's comments for one resale
// - AvatarID only       = one avatar's comments across all resale reviews
//
// Deleted:
// - nil   = no deleted-state filter
// - false = visible comments only
// - true  = deleted comments only
//
// IsRead:
// - nil   = no read-state filter
// - false = unread comments only
// - true  = read comments only
//
// IsRead represents whether the resale owner has read the comment.
type FilterComment struct {
	common.FilterCommon `json:",inline"`
	ResaleID            string `json:"resaleId"`
	AvatarID            string `json:"avatarId"`
	Deleted             *bool  `json:"deleted"`
	IsRead              *bool  `json:"isRead"`
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
// Comment repository
// ============================================================

// CommentRepository manages:
//
// resaleReviews/{resaleId}/comments/{commentId}
//
// Comment body is immutable after creation.
// Read-state changes are handled by MutationRepository.MarkCommentsRead.
// Normal logical deletion is handled by MutationRepository.MarkCommentDeleted.
// Physical deletion is reserved for resale-review cleanup.
type CommentRepository interface {
	// List lists comments by resaleId and/or avatarId.
	//
	// Supported query forms:
	// - ResaleID only       = comments under one resale review
	// - ResaleID + AvatarID = comments by one avatar under one resale review
	// - AvatarID only       = comments by one avatar across resale reviews
	//
	// At least one of ResaleID / AvatarID must be specified.
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

	// DeleteUnderParent physically deletes a comment document.
	//
	// Normal logical deletion must use MutationRepository.MarkCommentDeleted.
	// Physical deletion is intended for resale-review cleanup.
	DeleteUnderParent(
		ctx context.Context,
		resaleID string,
		commentID string,
	) error
}

// ============================================================
// Atomic mutation repository
// ============================================================

// MutationRepository represents operations that mutate resale-review state.
//
// Operations that affect CommentCount should keep the parent aggregate and
// comment document consistent in one transaction.
//
// This prevents states such as:
// - comment exists but CommentCount was not incremented
// - comment was logically deleted but CommentCount was not decremented
type MutationRepository interface {
	// AddComment creates a comment and increments CommentCount atomically.
	AddComment(
		ctx context.Context,
		comment Comment,
	) (ResaleReviewAggregate, Comment, error)

	// MarkCommentsRead marks all visible unread comments under resaleId as read.
	//
	// IsRead represents whether the resale owner has read the comment.
	// Authorization that the caller is the resale owner must be validated by
	// application.usecase before invoking this repository method.
	//
	// Implementations should keep this operation idempotent:
	// comments already marked as read must remain unchanged.
	//
	// The returned value is the number of comments newly marked as read.
	MarkCommentsRead(
		ctx context.Context,
		resaleID string,
	) (int, error)

	// MarkCommentDeleted logically deletes a comment and decrements
	// CommentCount atomically.
	//
	// Authorization and delete eligibility must be validated by
	// application.usecase before invoking this repository method.
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
	Comments() CommentRepository
	Mutations() MutationRepository
	Cleanup() CleanupRepository
}
