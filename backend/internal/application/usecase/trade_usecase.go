// backend/internal/application/usecase/trade_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"

	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeUsecaseNotConfigured = errors.New("trade usecase: not configured")
)

// TradeUsecase coordinates Trade creation for Resale Order items.
//
// Trade is used only for secondary-market transactions between Avatars:
//
//	buyer Avatar <-> seller Avatar
//
// One Trade belongs to exactly one Resale Order item:
//
//	orderId + orderItemIndex -> Trade
//
// Primary List sales do not create Trade.
//
// Trade is created when the Order is placed.
//
// Trade creation is idempotent. If a Trade already exists for a Resale Order
// item, EnsureForOrder leaves it unchanged.
//
// Buyer identity is taken from Order.AvatarID.
// Seller identity is taken from OrderItemSnapshot.SellerSnapshot.AvatarID.
//
// Cancellation, return, payment, dispatch, transfer, refund, and settlement
// state remain authoritative in their respective domains and are not copied
// into Trade.
type TradeUsecase struct {
	repo tradedom.Repository
}

func NewTradeUsecase(repo tradedom.Repository) *TradeUsecase {
	return &TradeUsecase{
		repo: repo,
	}
}

// EnsureForOrder ensures that every eligible Resale Order item has exactly
// one Trade.
//
// Preconditions:
//   - Order must already have a valid ID.
//   - Order must have a valid buyer Avatar ID.
//
// Items are handled as follows:
//   - cancelled item: skipped
//   - list item: skipped
//   - resale item: Trade ensured
//
// Idempotency:
//   - Existing Trade -> no-op.
//   - Missing Trade -> Create.
//   - ErrAlreadyExists during Create -> another concurrent execution created
//     the Trade, therefore treated as success.
//
// Each Resale Order item is processed independently. If one item fails after
// earlier items were created, a retry safely skips existing Trades and creates
// only the missing Trade.
func (u *TradeUsecase) EnsureForOrder(
	ctx context.Context,
	order orderdom.Order,
) error {
	if u == nil || u.repo == nil {
		return ErrTradeUsecaseNotConfigured
	}

	if order.ID == "" {
		return orderdom.ErrInvalidID
	}

	if order.AvatarID == "" {
		return orderdom.ErrInvalidAvatarID
	}

	for itemIndex, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		if item.Type != orderdom.OrderItemTypeResale {
			continue
		}

		if err := u.ensureForResaleOrderItem(
			ctx,
			order,
			itemIndex,
			item,
		); err != nil {
			return fmt.Errorf(
				"trade usecase: ensure order %s item %d: %w",
				order.ID,
				itemIndex,
				err,
			)
		}
	}

	return nil
}

func (u *TradeUsecase) ensureForResaleOrderItem(
	ctx context.Context,
	order orderdom.Order,
	itemIndex int,
	item orderdom.OrderItemSnapshot,
) error {
	if item.Type != orderdom.OrderItemTypeResale {
		return orderdom.ErrInvalidItemSnapshot
	}

	if item.SellerSnapshot.AvatarID == "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	_, err := u.repo.GetByOrderItem(
		ctx,
		order.ID,
		itemIndex,
	)

	switch {
	case err == nil:
		return nil

	case !errors.Is(err, tradedom.ErrNotFound):
		return err
	}

	trade, err := tradedom.NewAvatarTradeForCreate(
		"",
		order.ID,
		itemIndex,
		order.AvatarID,
		item.SellerSnapshot.AvatarID,
	)
	if err != nil {
		return err
	}

	_, err = u.repo.Create(
		ctx,
		trade,
	)
	if err != nil {
		if errors.Is(err, tradedom.ErrAlreadyExists) {
			return nil
		}

		return err
	}

	return nil
}
