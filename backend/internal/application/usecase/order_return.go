// backend/internal/application/usecase/order_return.go
package usecase

import (
	"context"
	"strings"

	orderdom "narratives/internal/domain/order"
)

// ReturnOrderItemInput identifies one Order item for which the purchaser
// requests a return.
//
// This input represents a return request only. It is not a refund instruction.
type ReturnOrderItemInput struct {
	ID        string
	AvatarID  string
	ItemIndex int
	Kind      orderdom.ReturnRequestKind
}

// ReturnItem records a purchaser return request.
//
// This method intentionally does not execute:
//
// - Stripe Refund
// - Stripe Transfer Reversal
// - Payment refund-state mutation
// - Settlement cancellation or reversal
//
// Those financial operations belong to RefundUsecase and must be started from
// a separate return-approval flow.
//
// The current RefundUsecase supports full Payment refunds only, while this
// method records an item-level request. A caller must therefore not translate
// one ReturnItem call directly into RefundByPaymentID without first confirming
// that the approved refund policy covers the complete Payment.
func (u *OrderUsecase) ReturnItem(
	ctx context.Context,
	in ReturnOrderItemInput,
) (orderdom.Order, error) {
	orderID := strings.TrimSpace(in.ID)
	if orderID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidID
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidAvatarID
	}

	if in.ItemIndex < 0 {
		return orderdom.Order{}, orderdom.ErrInvalidItems
	}

	order, err := u.repo.GetByID(ctx, orderID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if order.AvatarID != avatarID {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	if in.ItemIndex >= len(order.Items) {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	targetItem := order.Items[in.ItemIndex]

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.IsReturnCompleted ||
		targetItem.Transferred {
		return orderdom.Order{}, orderdom.ErrConflict
	}

	switch in.Kind {
	case orderdom.ReturnRequestKindUnopened:
		if targetItem.TokenTransferVerifiedAt != nil {
			return orderdom.Order{}, orderdom.ErrConflict
		}

	case orderdom.ReturnRequestKindOpened:

	default:
		return orderdom.Order{}, orderdom.ErrInvalidItemSnapshot
	}

	wasReturnRequested := targetItem.IsReturnRequested
	previousReturnRequestKind := targetItem.ReturnRequestKind

	if err := order.RequestItemReturn(
		in.ItemIndex,
		in.Kind,
		u.now().UTC(),
	); err != nil {
		return orderdom.Order{}, err
	}

	updatedItem := order.Items[in.ItemIndex]

	if wasReturnRequested &&
		previousReturnRequestKind == updatedItem.ReturnRequestKind {
		return order, nil
	}

	updated, err := u.repo.Update(ctx, order, nil)
	if err != nil {
		return orderdom.Order{}, err
	}

	return updated, nil
}

// CompleteReturnOrderItemInput identifies one Order item whose return has
// completed financially and may therefore be marked as completed.
//
// This command must only be called after the item-level Refund aggregate reports
// that all required purchaser refund and seller Transfer Reversal operations are
// financially completed.
type CompleteReturnOrderItemInput struct {
	ID        string
	ItemIndex int
}

// CompleteReturnItem records completion of an already requested item return.
//
// This method intentionally does not execute:
//
// - Stripe Refund
// - Stripe Transfer Reversal
// - Refund aggregate mutation
// - Inquiry resolution
//
// The caller must complete and persist those financial operations first.
// Order completion is the application-side state transition performed only after
// the Refund aggregate is financially completed.
func (u *OrderUsecase) CompleteReturnItem(
	ctx context.Context,
	in CompleteReturnOrderItemInput,
) (orderdom.Order, error) {
	orderID := strings.TrimSpace(in.ID)
	if orderID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidID
	}

	if in.ItemIndex < 0 {
		return orderdom.Order{}, orderdom.ErrInvalidItems
	}

	order, err := u.repo.GetByID(ctx, orderID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.ItemIndex >= len(order.Items) {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	targetItem := order.Items[in.ItemIndex]

	if targetItem.IsReturnCompleted {
		return order, nil
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		!targetItem.IsReturnRequested ||
		targetItem.ReturnRequestedAt == nil {
		return orderdom.Order{}, orderdom.ErrConflict
	}

	if err := order.CompleteItemReturn(
		in.ItemIndex,
		u.now().UTC(),
	); err != nil {
		return orderdom.Order{}, err
	}

	updated, err := u.repo.Update(ctx, order, nil)
	if err != nil {
		return orderdom.Order{}, err
	}

	return updated, nil
}
