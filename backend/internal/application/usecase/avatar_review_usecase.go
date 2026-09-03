// backend/internal/application/usecase/avatar_review_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	avatarreviewdom "narratives/internal/domain/avatar_review"
	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrAvatarReviewUsecaseNotConfigured = errors.New(
		"avatar review usecase: not configured",
	)

	ErrAvatarReviewReviewerRequired = errors.New(
		"avatar review usecase: reviewer avatar is required",
	)

	ErrAvatarReviewTradeNotFound = errors.New(
		"avatar review usecase: trade not found",
	)

	ErrAvatarReviewOrderNotFound = errors.New(
		"avatar review usecase: order not found",
	)

	ErrAvatarReviewForbidden = errors.New(
		"avatar review usecase: forbidden",
	)

	ErrAvatarReviewUnsupportedTrade = errors.New(
		"avatar review usecase: unsupported trade",
	)

	ErrAvatarReviewOrderMismatch = errors.New(
		"avatar review usecase: order mismatch",
	)

	ErrAvatarReviewTransferIncomplete = errors.New(
		"avatar review usecase: token transfer is not completed",
	)
)

// ============================================================
// Usecase
// ============================================================

// AvatarReviewUsecase coordinates creation of a buyer-to-seller Avatar review.
//
// Avatar Review is available only for Avatar-to-Avatar Resale transactions.
//
// The client provides:
//
//   - orderId
//   - orderItemIndex
//   - evaluation
//   - comment
//
// Reviewer identity is taken from the authenticated Avatar context by the HTTP
// layer and passed as ReviewerAvatarID.
//
// RevieweeAvatarID must never be trusted from the client.
// The seller Avatar is resolved from the authoritative Trade and Order
// snapshots.
//
// A review can be created only after token transfer for the corresponding
// Order item has completed.
//
// One Trade can have at most one Avatar Review. The repository is responsible
// for enforcing that persistence constraint.
type AvatarReviewUsecase struct {
	reviewRepo avatarreviewdom.Repository
	tradeRepo  tradedom.Repository
	orderRepo  orderdom.Repository
}

func NewAvatarReviewUsecase(
	reviewRepo avatarreviewdom.Repository,
	tradeRepo tradedom.Repository,
	orderRepo orderdom.Repository,
) *AvatarReviewUsecase {
	return &AvatarReviewUsecase{
		reviewRepo: reviewRepo,
		tradeRepo:  tradeRepo,
		orderRepo:  orderRepo,
	}
}

// ============================================================
// Create
// ============================================================

type CreateAvatarReviewInput struct {
	OrderID        string
	OrderItemIndex int

	ReviewerAvatarID string

	Evaluation avatarreviewdom.Evaluation
	Comment    string
}

// Create creates one Avatar Review for a completed Resale Trade.
//
// Authorization / consistency rules:
//
//   - Reviewer must be the Trade buyer.
//   - Trade must represent an Avatar-to-Avatar transaction.
//   - Order must belong to the same buyer Avatar.
//   - Trade and Order item identities must match.
//   - Order item must be a Resale item.
//   - Order seller snapshot must match Trade seller Avatar.
//   - Token transfer must already be completed.
//   - Reviewee Avatar is always derived from Trade/Order, never from client input.
//
// Persistence uniqueness:
//
//	one Trade -> at most one Avatar Review
func (u *AvatarReviewUsecase) Create(
	ctx context.Context,
	input CreateAvatarReviewInput,
) (avatarreviewdom.Review, error) {
	if u == nil ||
		u.reviewRepo == nil ||
		u.tradeRepo == nil ||
		u.orderRepo == nil {
		return avatarreviewdom.Review{},
			ErrAvatarReviewUsecaseNotConfigured
	}

	orderID := strings.TrimSpace(input.OrderID)
	reviewerAvatarID := strings.TrimSpace(input.ReviewerAvatarID)

	if reviewerAvatarID == "" {
		return avatarreviewdom.Review{},
			ErrAvatarReviewReviewerRequired
	}

	if orderID == "" {
		return avatarreviewdom.Review{},
			avatarreviewdom.ErrInvalidOrderID
	}

	if input.OrderItemIndex < 0 {
		return avatarreviewdom.Review{},
			avatarreviewdom.ErrInvalidOrderItemIndex
	}

	// ------------------------------------------------------------
	// Resolve Trade
	// ------------------------------------------------------------

	trade, err := u.tradeRepo.GetByOrderItem(
		ctx,
		orderID,
		input.OrderItemIndex,
	)
	if err != nil {
		if errors.Is(err, tradedom.ErrNotFound) {
			return avatarreviewdom.Review{},
				ErrAvatarReviewTradeNotFound
		}

		return avatarreviewdom.Review{}, err
	}

	// GetByOrderItem should already guarantee this identity, but validate it
	// defensively before using the Trade as the review target.
	if trade.OrderID != orderID ||
		trade.OrderItemIndex != input.OrderItemIndex {
		return avatarreviewdom.Review{},
			ErrAvatarReviewOrderMismatch
	}

	// Only the Trade buyer can submit the post-transfer review.
	if trade.BuyerAvatarID != reviewerAvatarID {
		return avatarreviewdom.Review{},
			ErrAvatarReviewForbidden
	}

	// Avatar Review is only for secondary-market Avatar-to-Avatar Trades.
	//
	// Primary List transactions use a company seller and are not eligible.
	if trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" {
		return avatarreviewdom.Review{},
			ErrAvatarReviewUnsupportedTrade
	}

	if trade.SellerAvatarID == reviewerAvatarID {
		return avatarreviewdom.Review{},
			ErrAvatarReviewUnsupportedTrade
	}

	// ------------------------------------------------------------
	// Resolve authoritative Order
	// ------------------------------------------------------------

	order, err := u.orderRepo.GetByID(
		ctx,
		orderID,
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return avatarreviewdom.Review{},
				ErrAvatarReviewOrderNotFound
		}

		return avatarreviewdom.Review{}, err
	}

	// The Order itself must belong to the same authenticated buyer.
	if order.AvatarID != reviewerAvatarID {
		return avatarreviewdom.Review{},
			ErrAvatarReviewForbidden
	}

	// Trade buyer and Order buyer must describe the same transaction.
	if order.AvatarID != trade.BuyerAvatarID {
		return avatarreviewdom.Review{},
			ErrAvatarReviewOrderMismatch
	}

	if input.OrderItemIndex >= len(order.Items) {
		return avatarreviewdom.Review{},
			ErrAvatarReviewOrderMismatch
	}

	item := order.Items[input.OrderItemIndex]

	// Avatar Review is explicitly a Resale transaction review.
	if item.Type != orderdom.OrderItemTypeResale {
		return avatarreviewdom.Review{},
			ErrAvatarReviewUnsupportedTrade
	}

	// A Resale Order item must snapshot the actual seller Avatar.
	if strings.TrimSpace(item.SellerSnapshot.AvatarID) == "" {
		return avatarreviewdom.Review{},
			ErrAvatarReviewOrderMismatch
	}

	// Trade seller and immutable Order seller snapshot must agree.
	if item.SellerSnapshot.AvatarID != trade.SellerAvatarID {
		return avatarreviewdom.Review{},
			ErrAvatarReviewOrderMismatch
	}

	// ------------------------------------------------------------
	// Verify token transfer completion
	// ------------------------------------------------------------

	// Order is authoritative for token-transfer state.
	//
	// Review submission is allowed only after the item has actually been
	// transferred to the buyer.
	if !item.Transferred ||
		item.TransferredAt == nil ||
		item.TransferredAt.IsZero() {
		return avatarreviewdom.Review{},
			ErrAvatarReviewTransferIncomplete
	}

	// ------------------------------------------------------------
	// Build immutable domain entity
	// ------------------------------------------------------------

	review, err := avatarreviewdom.NewReview(
		avatarreviewdom.NewReviewParams{
			TradeID:          trade.ID,
			OrderID:          order.ID,
			OrderItemIndex:   input.OrderItemIndex,
			ReviewerAvatarID: reviewerAvatarID,
			RevieweeAvatarID: trade.SellerAvatarID,
			Evaluation:       input.Evaluation,
			Comment:          input.Comment,
			CreatedAt:        time.Now().UTC(),
		},
	)
	if err != nil {
		return avatarreviewdom.Review{}, err
	}

	// ------------------------------------------------------------
	// Persist
	// ------------------------------------------------------------

	created, err := u.reviewRepo.Create(
		ctx,
		review,
	)
	if err != nil {
		return avatarreviewdom.Review{}, err
	}

	return created, nil
}
