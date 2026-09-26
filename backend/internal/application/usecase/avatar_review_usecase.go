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
// Constants
// ============================================================

const (
	DefaultAvatarReviewPage    = 1
	DefaultAvatarReviewPerPage = 20
	MaxAvatarReviewPerPage     = 100
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

// AvatarReviewUsecase coordinates public reads, review status resolution,
// and creation of buyer-to-seller Avatar reviews.
//
// Avatar Review is available only for Avatar-to-Avatar Resale transactions.
//
// Public reads expose reviews received by a target Avatar. Review status and
// creation are restricted to the authenticated buyer after token transfer
// completion.
//
// Reviewer identity is taken from the authenticated Avatar context by the HTTP
// layer. Reviewee identity is resolved from authoritative Trade and Order
// snapshots and must never be trusted from client input.
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
// Public list
// ============================================================

type ListAvatarReviewsInput struct {
	RevieweeAvatarID string
	Page             int
	PerPage          int
}

type ListAvatarReviewsResult struct {
	AvatarID          string                   `json:"avatarId"`
	GoodCount         int64                    `json:"goodCount"`
	DisappointedCount int64                    `json:"disappointedCount"`
	Total             int64                    `json:"total"`
	Page              int                      `json:"page"`
	PerPage           int                      `json:"perPage"`
	HasNext           bool                     `json:"hasNext"`
	Items             []avatarreviewdom.Review `json:"items"`
}

// ListByRevieweeAvatarID returns one public page of reviews received by an
// Avatar together with the current aggregate evaluation counts.
//
// No reviews is a normal result. In that case Items is an empty slice and all
// summary counts are zero.
func (u *AvatarReviewUsecase) ListByRevieweeAvatarID(
	ctx context.Context,
	input ListAvatarReviewsInput,
) (ListAvatarReviewsResult, error) {
	if u == nil || u.reviewRepo == nil {
		return ListAvatarReviewsResult{}, ErrAvatarReviewUsecaseNotConfigured
	}

	revieweeAvatarID := strings.TrimSpace(input.RevieweeAvatarID)
	if revieweeAvatarID == "" ||
		len(revieweeAvatarID) > avatarreviewdom.MaxReferenceIDLength ||
		strings.Contains(revieweeAvatarID, "/") {
		return ListAvatarReviewsResult{}, avatarreviewdom.ErrInvalidRevieweeAvatarID
	}

	page := input.Page
	if page == 0 {
		page = DefaultAvatarReviewPage
	}

	perPage := input.PerPage
	if perPage == 0 {
		perPage = DefaultAvatarReviewPerPage
	}

	if page < 1 || perPage < 1 || perPage > MaxAvatarReviewPerPage {
		return ListAvatarReviewsResult{}, avatarreviewdom.ErrInvalidPagination
	}

	offset := (page - 1) * perPage
	if offset < 0 {
		return ListAvatarReviewsResult{}, avatarreviewdom.ErrInvalidPagination
	}

	reviews, err := u.reviewRepo.ListByRevieweeAvatarID(
		ctx,
		avatarreviewdom.ListByRevieweeAvatarIDParams{
			RevieweeAvatarID: revieweeAvatarID,
			Limit:            perPage + 1,
			Offset:           offset,
		},
	)
	if err != nil {
		return ListAvatarReviewsResult{}, err
	}

	hasNext := len(reviews) > perPage
	if hasNext {
		reviews = reviews[:perPage]
	}

	if reviews == nil {
		reviews = []avatarreviewdom.Review{}
	}

	summary, err := u.reviewRepo.GetSummaryByRevieweeAvatarID(
		ctx,
		revieweeAvatarID,
	)
	if err != nil {
		return ListAvatarReviewsResult{}, err
	}

	return ListAvatarReviewsResult{
		AvatarID:          revieweeAvatarID,
		GoodCount:         summary.GoodCount,
		DisappointedCount: summary.DisappointedCount,
		Total:             summary.Total,
		Page:              page,
		PerPage:           perPage,
		HasNext:           hasNext,
		Items:             reviews,
	}, nil
}

// ============================================================
// Review status
// ============================================================

type GetAvatarReviewStatusInput struct {
	OrderID          string
	OrderItemIndex   int
	ReviewerAvatarID string
}

type GetAvatarReviewStatusResult struct {
	Eligible         bool   `json:"eligible"`
	Reviewed         bool   `json:"reviewed"`
	TradeID          string `json:"tradeId"`
	OrderID          string `json:"orderId"`
	OrderItemIndex   int    `json:"orderItemIndex"`
	RevieweeAvatarID string `json:"revieweeAvatarId"`
}

// GetStatusByOrderItem resolves whether the authenticated buyer can create an
// Avatar Review for one completed Resale order item and whether that review
// has already been submitted.
//
// The same authoritative Trade and Order checks used by Create are performed.
// Review existence is resolved by Trade ID because one Trade can have at most
// one Avatar Review.
func (u *AvatarReviewUsecase) GetStatusByOrderItem(
	ctx context.Context,
	input GetAvatarReviewStatusInput,
) (GetAvatarReviewStatusResult, error) {
	if u == nil ||
		u.reviewRepo == nil ||
		u.tradeRepo == nil ||
		u.orderRepo == nil {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewUsecaseNotConfigured
	}

	orderID := strings.TrimSpace(input.OrderID)
	reviewerAvatarID := strings.TrimSpace(input.ReviewerAvatarID)

	if reviewerAvatarID == "" {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewReviewerRequired
	}

	if orderID == "" {
		return GetAvatarReviewStatusResult{}, avatarreviewdom.ErrInvalidOrderID
	}

	if input.OrderItemIndex < 0 {
		return GetAvatarReviewStatusResult{}, avatarreviewdom.ErrInvalidOrderItemIndex
	}

	trade, err := u.tradeRepo.GetByOrderItem(
		ctx,
		orderID,
		input.OrderItemIndex,
	)
	if err != nil {
		if errors.Is(err, tradedom.ErrNotFound) {
			return GetAvatarReviewStatusResult{}, ErrAvatarReviewTradeNotFound
		}

		return GetAvatarReviewStatusResult{}, err
	}

	if trade.OrderID != orderID ||
		trade.OrderItemIndex != input.OrderItemIndex {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderMismatch
	}

	if trade.BuyerAvatarID != reviewerAvatarID {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewForbidden
	}

	if trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewUnsupportedTrade
	}

	if trade.SellerAvatarID == reviewerAvatarID {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewUnsupportedTrade
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		orderID,
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderNotFound
		}

		return GetAvatarReviewStatusResult{}, err
	}

	if order.AvatarID != reviewerAvatarID {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewForbidden
	}

	if order.AvatarID != trade.BuyerAvatarID {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderMismatch
	}

	if input.OrderItemIndex >= len(order.Items) {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderMismatch
	}

	item := order.Items[input.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewUnsupportedTrade
	}

	if strings.TrimSpace(item.SellerSnapshot.AvatarID) == "" {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderMismatch
	}

	if item.SellerSnapshot.AvatarID != trade.SellerAvatarID {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewOrderMismatch
	}

	if !item.Transferred ||
		item.TransferredAt == nil ||
		item.TransferredAt.IsZero() {
		return GetAvatarReviewStatusResult{}, ErrAvatarReviewTransferIncomplete
	}

	reviewed := false

	_, err = u.reviewRepo.GetByTradeID(
		ctx,
		trade.ID,
	)
	if err == nil {
		reviewed = true
	} else if !errors.Is(err, avatarreviewdom.ErrNotFound) {
		return GetAvatarReviewStatusResult{}, err
	}

	return GetAvatarReviewStatusResult{
		Eligible:         true,
		Reviewed:         reviewed,
		TradeID:          trade.ID,
		OrderID:          order.ID,
		OrderItemIndex:   input.OrderItemIndex,
		RevieweeAvatarID: trade.SellerAvatarID,
	}, nil
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
		return avatarreviewdom.Review{}, ErrAvatarReviewUsecaseNotConfigured
	}

	orderID := strings.TrimSpace(input.OrderID)
	reviewerAvatarID := strings.TrimSpace(input.ReviewerAvatarID)

	if reviewerAvatarID == "" {
		return avatarreviewdom.Review{}, ErrAvatarReviewReviewerRequired
	}

	if orderID == "" {
		return avatarreviewdom.Review{}, avatarreviewdom.ErrInvalidOrderID
	}

	if input.OrderItemIndex < 0 {
		return avatarreviewdom.Review{}, avatarreviewdom.ErrInvalidOrderItemIndex
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
			return avatarreviewdom.Review{}, ErrAvatarReviewTradeNotFound
		}

		return avatarreviewdom.Review{}, err
	}

	if trade.OrderID != orderID ||
		trade.OrderItemIndex != input.OrderItemIndex {
		return avatarreviewdom.Review{}, ErrAvatarReviewOrderMismatch
	}

	if trade.BuyerAvatarID != reviewerAvatarID {
		return avatarreviewdom.Review{}, ErrAvatarReviewForbidden
	}

	if trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" {
		return avatarreviewdom.Review{}, ErrAvatarReviewUnsupportedTrade
	}

	if trade.SellerAvatarID == reviewerAvatarID {
		return avatarreviewdom.Review{}, ErrAvatarReviewUnsupportedTrade
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
			return avatarreviewdom.Review{}, ErrAvatarReviewOrderNotFound
		}

		return avatarreviewdom.Review{}, err
	}

	if order.AvatarID != reviewerAvatarID {
		return avatarreviewdom.Review{}, ErrAvatarReviewForbidden
	}

	if order.AvatarID != trade.BuyerAvatarID {
		return avatarreviewdom.Review{}, ErrAvatarReviewOrderMismatch
	}

	if input.OrderItemIndex >= len(order.Items) {
		return avatarreviewdom.Review{}, ErrAvatarReviewOrderMismatch
	}

	item := order.Items[input.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale {
		return avatarreviewdom.Review{}, ErrAvatarReviewUnsupportedTrade
	}

	if strings.TrimSpace(item.SellerSnapshot.AvatarID) == "" {
		return avatarreviewdom.Review{}, ErrAvatarReviewOrderMismatch
	}

	if item.SellerSnapshot.AvatarID != trade.SellerAvatarID {
		return avatarreviewdom.Review{}, ErrAvatarReviewOrderMismatch
	}

	// ------------------------------------------------------------
	// Verify token transfer completion
	// ------------------------------------------------------------

	if !item.Transferred ||
		item.TransferredAt == nil ||
		item.TransferredAt.IsZero() {
		return avatarreviewdom.Review{}, ErrAvatarReviewTransferIncomplete
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
