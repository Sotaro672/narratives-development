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
	ErrTradeOrderNotPaid         = errors.New("trade usecase: order is not paid")
	ErrTradeUnsupportedItemType  = errors.New("trade usecase: unsupported order item type")
)

// TradeUsecase coordinates Trade creation from a paid Order.
//
// One Trade belongs to exactly one Order item:
//
//	orderId + orderItemIndex -> Trade
//
// Trade creation is idempotent. If a Trade already exists for an Order item,
// EnsureForPaidOrder leaves it unchanged.
//
// Buyer identity is always taken from Order.AvatarID.
//
// Seller identity is taken from the immutable SellerSnapshot stored in the
// Order item:
//   - list: company / brand
//   - resale: avatar
//
// Cancellation, return, dispatch, transfer, refund, and settlement state remain
// authoritative in their respective domains and are not copied into Trade.
type TradeUsecase struct {
	repo tradedom.Repository
}

func NewTradeUsecase(repo tradedom.Repository) *TradeUsecase {
	return &TradeUsecase{repo: repo}
}

// EnsureForPaidOrder ensures that every eligible Order item has exactly one
// Trade.
//
// Preconditions:
//   - Order must already be paid.
//   - Order ID and buyer Avatar ID must be valid.
//   - Cancelled items are skipped.
//
// Idempotency:
//   - Existing Trade -> no-op.
//   - Missing Trade -> Create.
//   - ErrAlreadyExists during Create -> another concurrent execution created
//     the Trade, therefore treated as success.
//
// This method intentionally processes each Order item independently. If one
// item fails after earlier items were created, a retry safely skips the already
// created Trades and continues from the missing item.
func (u *TradeUsecase) EnsureForPaidOrder(
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
	if !order.Paid {
		return ErrTradeOrderNotPaid
	}

	for itemIndex, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		if err := u.ensureForOrderItem(
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

func (u *TradeUsecase) ensureForOrderItem(
	ctx context.Context,
	order orderdom.Order,
	itemIndex int,
	item orderdom.OrderItemSnapshot,
) error {
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

	trade, err := newTradeFromOrderItem(
		order,
		itemIndex,
		item,
	)
	if err != nil {
		return err
	}

	_, err = u.repo.Create(ctx, trade)
	if err != nil {
		if errors.Is(err, tradedom.ErrAlreadyExists) {
			return nil
		}
		return err
	}

	return nil
}

func newTradeFromOrderItem(
	order orderdom.Order,
	itemIndex int,
	item orderdom.OrderItemSnapshot,
) (tradedom.Trade, error) {
	switch item.Type {
	case orderdom.OrderItemTypeList:
		return tradedom.NewCompanyTradeForCreate(
			"",
			order.ID,
			itemIndex,
			order.AvatarID,
			item.SellerSnapshot.CompanyID,
			item.SellerSnapshot.BrandID,
		)

	case orderdom.OrderItemTypeResale:
		return tradedom.NewAvatarTradeForCreate(
			"",
			order.ID,
			itemIndex,
			order.AvatarID,
			item.SellerSnapshot.AvatarID,
		)

	default:
		return tradedom.Trade{}, fmt.Errorf(
			"%w: %q",
			ErrTradeUnsupportedItemType,
			item.Type,
		)
	}
}
