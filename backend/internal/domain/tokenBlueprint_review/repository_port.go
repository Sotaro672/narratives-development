// backend/internal/domain/tokenBlueprint_review/repository_port.go

package tokenBlueprint_review

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// ============================================================
// Patch / Filter
// ============================================================

// PatchTokenBlueprintReviewAggregate is a partial update model for aggregate doc.
// Repository implementation should validate allowed fields.
type PatchTokenBlueprintReviewAggregate struct {
	LikeCount            *int64  `json:"likeCount"`
	DislikeCount         *int64  `json:"dislikeCount"`
	TopLevelCommentCount *int64  `json:"topLevelCommentCount"`
	TotalCommentCount    *int64  `json:"totalCommentCount"`
	PinnedCommentID      *string `json:"pinnedCommentId"`
}

// NewPatchFromTokenBlueprintReviewAggregate creates a repository patch from
// the current aggregate state.
//
// This is used when domain methods mutate aggregate counters and the
// application layer needs to persist the changed counter fields.
func NewPatchFromTokenBlueprintReviewAggregate(
	agg TokenBlueprintReviewAggregate,
) PatchTokenBlueprintReviewAggregate {
	return PatchTokenBlueprintReviewAggregate{
		LikeCount:            &agg.LikeCount,
		DislikeCount:         &agg.DislikeCount,
		TopLevelCommentCount: &agg.TopLevelCommentCount,
		TotalCommentCount:    &agg.TotalCommentCount,
		PinnedCommentID:      &agg.PinnedCommentID,
	}
}

// FilterComment is for listing comments under a tokenBlueprintId.
// This supports both top-level comments and nested replies.
type FilterComment struct {
	common.FilterCommon `json:",inline"`
	TokenBlueprintID    string      `json:"tokenBlueprintId"`
	ParentCommentID     *string     `json:"parentCommentId"`
	RootCommentID       string      `json:"rootCommentId"`
	AuthorID            string      `json:"authorId"`
	AuthorType          *AuthorType `json:"authorType"`
	Deleted             *bool       `json:"deleted"`
	Depth               *int        `json:"depth"`
}

// PatchComment is a partial update model for comment doc.
// UpdatedAt is set only when the comment body is edited.
type PatchComment struct {
	Body         *string    `json:"body"`
	Deleted      *bool      `json:"deleted"`
	LikeCount    *int64     `json:"likeCount"`
	DislikeCount *int64     `json:"dislikeCount"`
	ChildCount   *int64     `json:"childCount"`
	UpdatedAt    *time.Time `json:"updatedAt"`
}

// NewBodyPatchFromComment creates a patch for persisting an edited comment body.
// UpdatedAt is included because body editing is the only trigger for comment UpdatedAt.
func NewBodyPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		Body:      &comment.Body,
		UpdatedAt: comment.UpdatedAt,
	}
}

// NewDeletePatchFromComment creates a patch for persisting a logically deleted comment.
// Deletion does not change UpdatedAt because UpdatedAt represents body edits only.
func NewDeletePatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		Body:    &comment.Body,
		Deleted: &comment.Deleted,
	}
}

// NewChildCountPatchFromComment creates a patch for persisting only the
// direct child count of a comment. Child count changes do not change UpdatedAt.
func NewChildCountPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		ChildCount: &comment.ChildCount,
	}
}

// NewReactionCountPatchFromComment creates a patch for persisting only
// reaction-related counters of a comment. Reaction changes do not change UpdatedAt.
func NewReactionCountPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		LikeCount:    &comment.LikeCount,
		DislikeCount: &comment.DislikeCount,
		ChildCount:   &comment.ChildCount,
	}
}

// ============================================================
// Ports
// ============================================================

// TokenBlueprintAggregateRepository manages the parent document:
// tokenBlueprintReviews/{tokenBlueprintId}
type TokenBlueprintAggregateRepository interface {
	GetByID(ctx context.Context, id string) (TokenBlueprintReviewAggregate, error)
	Create(ctx context.Context, entity TokenBlueprintReviewAggregate) (TokenBlueprintReviewAggregate, error)
	Update(ctx context.Context, id string, patch PatchTokenBlueprintReviewAggregate) (TokenBlueprintReviewAggregate, error)
	Delete(ctx context.Context, id string) error
}

// CommentRepository manages comments collection under a tokenBlueprint:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}
type CommentRepository interface {
	List(ctx context.Context, filter FilterComment, sort common.Sort, page common.Page) (common.PageResult[Comment], error)
	GetByParentID(ctx context.Context, tokenBlueprintID, commentID string) (Comment, error)
	CreateUnderParent(ctx context.Context, tokenBlueprintID string, comment Comment) (Comment, error)
	UpdateUnderParent(ctx context.Context, tokenBlueprintID, commentID string, patch PatchComment) (Comment, error)
	DeleteUnderParent(ctx context.Context, tokenBlueprintID, commentID string) error
}

// ============================================================
// Reaction ports
// ============================================================

// TokenBlueprintReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/reactions/{actorType_actorId}
type TokenBlueprintReactionRepository interface {
	FindByActor(
		ctx context.Context,
		tokenBlueprintID string,
		actorType ActorType,
		actorID string,
	) (TokenBlueprintReaction, error)

	Upsert(
		ctx context.Context,
		reaction TokenBlueprintReaction,
	) (TokenBlueprintReaction, error)
}

// CommentReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}/reactions/{actorType_actorId}
type CommentReactionRepository interface {
	FindByActor(
		ctx context.Context,
		tokenBlueprintID string,
		commentID string,
		actorType ActorType,
		actorID string,
	) (CommentReaction, error)

	Upsert(
		ctx context.Context,
		reaction CommentReaction,
	) (CommentReaction, error)
}

// ============================================================
// Composite port
// ============================================================

// RepositoryPort bundles all repositories for this domain.
type RepositoryPort interface {
	TokenBlueprintAggregates() TokenBlueprintAggregateRepository
	Comments() CommentRepository
	TokenBlueprintReactions() TokenBlueprintReactionRepository
	CommentReactions() CommentReactionRepository
}
