// backend/internal/domain/tokenBlueprint_review/repository_port.go
package tokenBlueprint_review

import (
	"context"

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
	TokenBlueprintID    string      `json:"tokenBlueprintId"` // required for parent aggregate
	ParentCommentID     *string     `json:"parentCommentId"`  // nil=no filter, ptr("")=top-level only, ptr(id)=children of the parent
	RootCommentID       string      `json:"rootCommentId"`    // optional
	AuthorID            string      `json:"authorId"`         // optional
	AuthorType          *AuthorType `json:"authorType"`       // optional
	IsOwnerComment      *bool       `json:"isOwnerComment"`   // optional
	Deleted             *bool       `json:"deleted"`          // optional
	Depth               *int        `json:"depth"`            // optional
}

// PatchComment is a partial update model for comment doc.
type PatchComment struct {
	Body           *string `json:"body"`
	Deleted        *bool   `json:"deleted"`
	IsOwnerComment *bool   `json:"isOwnerComment"`
	LikeCount      *int64  `json:"likeCount"`
	DislikeCount   *int64  `json:"dislikeCount"`
	ChildCount     *int64  `json:"childCount"`
}

// NewChildCountPatchFromComment creates a patch for persisting only the
// direct child count of a comment.
func NewChildCountPatchFromComment(
	comment Comment,
) PatchComment {
	return PatchComment{
		ChildCount: &comment.ChildCount,
	}
}

// NewReactionCountPatchFromComment creates a patch for persisting only
// reaction-related counters of a comment.
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
}

// CommentRepository manages comments collection under a tokenBlueprint:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}
type CommentRepository interface {
	// List lists comments under tokenBlueprintId with optional filters.
	List(ctx context.Context, filter FilterComment, sort common.Sort, page common.Page) (common.PageResult[Comment], error)

	// GetByParentID fetches a comment by parent tokenBlueprintId and commentId.
	GetByParentID(ctx context.Context, tokenBlueprintID, commentID string) (Comment, error)

	// CreateUnderParent creates a comment under tokenBlueprintId.
	CreateUnderParent(ctx context.Context, tokenBlueprintID string, comment Comment) (Comment, error)

	// UpdateUnderParent updates a comment directly under tokenBlueprintId by commentId.
	UpdateUnderParent(ctx context.Context, tokenBlueprintID, commentID string, patch PatchComment) (Comment, error)

	// DeleteUnderParent deletes a comment under tokenBlueprintId.
	DeleteUnderParent(ctx context.Context, tokenBlueprintID, commentID string) error
}

// ============================================================
// Reaction ports
// ============================================================

// TokenBlueprintReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/reactions/{actorType_actorId}
type TokenBlueprintReactionRepository interface {
	// FindByActor fetches a reaction by tokenBlueprintId + actorType + actorId.
	FindByActor(ctx context.Context, tokenBlueprintID string, actorType ActorType, actorID string) (TokenBlueprintReaction, error)

	// Upsert creates or updates a reaction doc.
	Upsert(ctx context.Context, reaction TokenBlueprintReaction) (TokenBlueprintReaction, error)
}

// CommentReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}/reactions/{actorType_actorId}
type CommentReactionRepository interface {
	// FindByActor fetches a reaction by tokenBlueprintId + commentId + actorType + actorId.
	FindByActor(ctx context.Context, tokenBlueprintID, commentID string, actorType ActorType, actorID string) (CommentReaction, error)

	// Upsert creates or updates a reaction doc.
	Upsert(ctx context.Context, reaction CommentReaction) (CommentReaction, error)
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
