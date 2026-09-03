// backend/internal/domain/avatar_review/repository_port.go
package avatar_review

import (
	"context"
	"errors"
)

// ============================================================
// Contract errors
// ============================================================

var (
	ErrNotFound      = errors.New("avatar_review: not found")
	ErrAlreadyExists = errors.New("avatar_review: already exists")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// ============================================================
// Read models
// ============================================================

// ReviewSummary represents the public evaluation totals for one reviewee Avatar.
type ReviewSummary struct {
	GoodCount         int64 `json:"goodCount"`
	DisappointedCount int64 `json:"disappointedCount"`
	Total             int64 `json:"total"`
}

// ListByRevieweeAvatarIDParams specifies one page of reviews received by an Avatar.
// Offset is zero-based. Limit must be positive.
type ListByRevieweeAvatarIDParams struct {
	RevieweeAvatarID string
	Limit            int
	Offset           int
}

// ============================================================
// Repository
// ============================================================

// Repository is the repository port for Avatar Review.
//
// Firestore:
//
//	avatarReviews/{tradeId}
//
// One Trade can have at most one Avatar Review.
//
// Review.ID is intentionally equal to Review.TradeID so that the persistence
// layer can enforce the one-review-per-trade constraint by using tradeId as
// the document ID.
//
// Avatar Review is immutable after creation. Editing and deletion are
// intentionally not part of this repository contract.
type Repository interface {
	// GetByTradeID retrieves the Avatar Review associated with one Trade.
	// Implementations should return ErrNotFound when no Review exists.
	GetByTradeID(ctx context.Context, tradeID string) (Review, error)

	// ListByRevieweeAvatarID retrieves reviews received by one Avatar.
	// Implementations should return an empty slice, not ErrNotFound, when no
	// reviews exist. Results should be ordered by CreatedAt descending.
	ListByRevieweeAvatarID(
		ctx context.Context,
		params ListByRevieweeAvatarIDParams,
	) ([]Review, error)

	// GetSummaryByRevieweeAvatarID returns public evaluation totals for one
	// Avatar. When no reviews exist, all counts should be zero and no
	// ErrNotFound should be returned.
	GetSummaryByRevieweeAvatarID(
		ctx context.Context,
		revieweeAvatarID string,
	) (ReviewSummary, error)

	// Create persists one Avatar Review.
	//
	// Implementations must enforce:
	//
	//	Review.ID == Review.TradeID
	//
	// and must prevent creation of more than one Review for the same Trade.
	// Implementations should return ErrAlreadyExists when a Review for the
	// same Trade already exists.
	Create(ctx context.Context, review Review) (Review, error)
}
