// backend/internal/domain/like/repository_port.go
package like

import (
	"context"

	common "narratives/internal/domain/common"
)

// Common type aliases.
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// Filter represents optional conditions when listing an avatar's Likes.
//
// AvatarID is intentionally not included here because Likes are always
// queried within one avatar's scope:
//
// likes/{avatarId}/items/{documentId}
type Filter struct {
	TargetType *TargetType
	TargetIDs  []string
}

// Repository manages avatar-scoped favorites.
//
// Firestore mapping:
//
// likes/{avatarId}/items/{documentId}
//
// documentId:
// - list_{listId}
// - resale_{resaleId}
//
// The repository is shared by both primary-sale List favorites and
// secondary-market Resale favorites.
type Repository interface {
	// ListByAvatarID returns Likes saved by one avatar.
	//
	// Typical usage for the Favorites page:
	//
	// ListByAvatarID(
	//     ctx,
	//     avatarID,
	//     Filter{},
	//     Sort{Column: "createdAt", Order: SortDesc},
	//     page,
	// )
	//
	// An empty filter returns both List and Resale Likes.
	ListByAvatarID(
		ctx context.Context,
		avatarID string,
		filter Filter,
		sort Sort,
		page Page,
	) (PageResult[Like], error)

	// Exists returns whether the avatar has already saved the target.
	//
	// This should normally be implemented as a direct document lookup using:
	//
	// likes/{avatarId}/items/{targetType}_{targetId}
	Exists(
		ctx context.Context,
		avatarID string,
		targetType TargetType,
		targetID string,
	) (bool, error)

	// Create persists one Like.
	//
	// The deterministic document ID guarantees uniqueness for:
	//
	// avatarId + targetType + targetId
	//
	// Implementations should return ErrConflict if the Like already exists.
	Create(
		ctx context.Context,
		entity Like,
	) (Like, error)

	// Delete physically removes one Like.
	//
	// Unlike is represented by physical deletion.
	//
	// Implementations should keep this operation idempotent where practical:
	// deleting an already missing Like should not create inconsistent state.
	Delete(
		ctx context.Context,
		avatarID string,
		targetType TargetType,
		targetID string,
	) error
}
