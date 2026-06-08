// backend/internal/domain/tokenBlueprint_review/repository_port.go
package tokenBlueprint_review

import (
	"context"

	common "narratives/internal/domain/common"
)

// ============================================================
// Filter / Patch
// ============================================================

// FilterTokenBlueprintReviewAggregate is for listing tokenBlueprint-level aggregates.
type FilterTokenBlueprintReviewAggregate struct {
	common.FilterCommon `json:",inline"`
	TokenBlueprintID    string `json:"tokenBlueprintId"` // exact match (optional)
}

// PatchTokenBlueprintReviewAggregate is a partial update model for aggregate doc.
// (Repository implementation should validate allowed fields.)
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

// FilterTokenBlueprintReaction is for listing reactions on the tokenBlueprint aggregate.
type FilterTokenBlueprintReaction struct {
	common.FilterCommon `json:",inline"`
	TokenBlueprintID    string        `json:"tokenBlueprintId"` // required
	ActorID             string        `json:"actorId"`          // optional
	ActorType           *ActorType    `json:"actorType"`        // optional
	Type                *ReactionType `json:"type"`             // optional
}

// PatchTokenBlueprintReaction is a partial update model for tokenBlueprint reaction doc.
type PatchTokenBlueprintReaction struct {
	Type *ReactionType `json:"type"`
}

// FilterCommentReaction is for listing reactions under a comment.
type FilterCommentReaction struct {
	common.FilterCommon `json:",inline"`
	TokenBlueprintID    string        `json:"tokenBlueprintId"` // required
	CommentID           string        `json:"commentId"`        // required
	ActorID             string        `json:"actorId"`          // optional
	ActorType           *ActorType    `json:"actorType"`        // optional
	Type                *ReactionType `json:"type"`             // optional
}

// PatchCommentReaction is a partial update model for comment reaction doc.
type PatchCommentReaction struct {
	Type *ReactionType `json:"type"`
}

// ============================================================
// Ports (Repository interfaces)
// ============================================================

// TokenBlueprintAggregateRepository manages the parent document:
// tokenBlueprintReviews/{tokenBlueprintId}
type TokenBlueprintAggregateRepository interface {
	common.Repository[TokenBlueprintReviewAggregate, FilterTokenBlueprintReviewAggregate, PatchTokenBlueprintReviewAggregate]
}

// CommentRepository manages comments collection under a tokenBlueprint:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}
//
// NOTE: The underlying common.Repository uses id string as the entity id.
// For this domain, id should be commentId.
// TokenBlueprintID is passed via filter / explicit parent methods below.
type CommentRepository interface {
	common.Repository[Comment, FilterComment, PatchComment]

	// GetByParentID fetches a comment by parent tokenBlueprintId and commentId.
	GetByParentID(ctx context.Context, tokenBlueprintID, commentID string) (Comment, error)

	// UpdateUnderParent updates a comment directly under tokenBlueprintId by commentId.
	// Use this instead of collection-group based Update when tokenBlueprintID is already known.
	UpdateUnderParent(ctx context.Context, tokenBlueprintID, commentID string, patch PatchComment) (Comment, error)

	// ListByTokenBlueprintID lists comments that have the same tokenBlueprintId.
	ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Comment, error)

	// ListTopLevelByTokenBlueprintID lists only top-level comments.
	ListTopLevelByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Comment, error)

	// ListByParentCommentID lists direct children of parentCommentID.
	ListByParentCommentID(ctx context.Context, tokenBlueprintID, parentCommentID string) ([]Comment, error)

	// ListByRootCommentID lists all comments in the same thread rooted at rootCommentID.
	ListByRootCommentID(ctx context.Context, tokenBlueprintID, rootCommentID string) ([]Comment, error)

	// ListOwnerCommentsByTokenBlueprintID lists comments authored as owner comments.
	ListOwnerCommentsByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Comment, error)

	// CreateUnderParent creates a comment under tokenBlueprintId.
	CreateUnderParent(ctx context.Context, tokenBlueprintID string, comment Comment) (Comment, error)

	// DeleteUnderParent deletes a comment under tokenBlueprintId (hard delete).
	DeleteUnderParent(ctx context.Context, tokenBlueprintID, commentID string) error
}

// ============================================================
// Reaction ports (subcollections)
// ============================================================

// TokenBlueprintReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/reactions/{actorType_actorId}
type TokenBlueprintReactionRepository interface {
	common.Repository[TokenBlueprintReaction, FilterTokenBlueprintReaction, PatchTokenBlueprintReaction]

	// FindByActor fetches a reaction by tokenBlueprintId + actorType + actorId.
	FindByActor(ctx context.Context, tokenBlueprintID string, actorType ActorType, actorID string) (TokenBlueprintReaction, error)

	// FindByDocumentID fetches a reaction by tokenBlueprintId + repository doc id.
	// Expected doc id format is "{actorType}_{actorId}".
	FindByDocumentID(ctx context.Context, tokenBlueprintID, reactionDocumentID string) (TokenBlueprintReaction, error)

	// ListByTokenBlueprintID lists all reactions under the tokenBlueprint.
	ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]TokenBlueprintReaction, error)

	// ListByActor lists reactions by a specific actor.
	ListByActor(ctx context.Context, actorType ActorType, actorID string) ([]TokenBlueprintReaction, error)

	// Upsert creates or updates a reaction doc.
	Upsert(ctx context.Context, reaction TokenBlueprintReaction) (TokenBlueprintReaction, error)

	// DeleteByActor deletes a reaction by tokenBlueprintId + actorType + actorId.
	DeleteByActor(ctx context.Context, tokenBlueprintID string, actorType ActorType, actorID string) error
}

// CommentReactionRepository manages:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}/reactions/{actorType_actorId}
type CommentReactionRepository interface {
	common.Repository[CommentReaction, FilterCommentReaction, PatchCommentReaction]

	// FindByActor fetches a reaction by tokenBlueprintId + commentId + actorType + actorId.
	FindByActor(ctx context.Context, tokenBlueprintID, commentID string, actorType ActorType, actorID string) (CommentReaction, error)

	// FindByDocumentID fetches a reaction by tokenBlueprintId + commentId + repository doc id.
	// Expected doc id format is "{actorType}_{actorId}".
	FindByDocumentID(ctx context.Context, tokenBlueprintID, commentID, reactionDocumentID string) (CommentReaction, error)

	// ListByCommentID lists all reactions under the comment.
	ListByCommentID(ctx context.Context, tokenBlueprintID, commentID string) ([]CommentReaction, error)

	// ListByActor lists comment reactions by a specific actor.
	ListByActor(ctx context.Context, actorType ActorType, actorID string) ([]CommentReaction, error)

	// Upsert creates or updates a reaction doc.
	Upsert(ctx context.Context, reaction CommentReaction) (CommentReaction, error)

	// DeleteByActor deletes a reaction by tokenBlueprintId + commentId + actorType + actorId.
	DeleteByActor(ctx context.Context, tokenBlueprintID, commentID string, actorType ActorType, actorID string) error
}

// ============================================================
// Convenience composite port (optional)
// ============================================================

// RepositoryPort bundles all repositories for this domain.
type RepositoryPort interface {
	TokenBlueprintAggregates() TokenBlueprintAggregateRepository
	Comments() CommentRepository
	TokenBlueprintReactions() TokenBlueprintReactionRepository
	CommentReactions() CommentReactionRepository
}
