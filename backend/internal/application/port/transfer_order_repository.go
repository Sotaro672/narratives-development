// backend/internal/application/port/transfer_order_repository.go
package port

import (
	"context"
	"time"

	orderdom "narratives/internal/domain/order"
	settlementdom "narratives/internal/domain/settlement"
)

// FindEligibleTransferItemInput identifies one transfer-eligible order item.
//
// AvatarID is the buyer-side avatar that owns the paid order.
// ProductID, ModelID, and TokenBlueprintID must identify the scanned product
// exactly.
type FindEligibleTransferItemInput struct {
	AvatarID         string
	ProductID        string
	ModelID          string
	TokenBlueprintID string
}

// TransferTargetItem is the minimal order-item state required by the transfer
// application flow.
type TransferTargetItem struct {
	OrderID   string
	ItemIndex int
	ItemType  orderdom.OrderItemType

	InventoryID string
	ModelID     string
	ResaleID    string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// OrderRepoForTransfer provides exact lookup and state transitions for one
// transfer-eligible order item.
//
// Implementations must use the canonical orderTransferItems read model and
// must not scan Order documents in memory.
//
// Resale fulfillment has a stronger consistency requirement than a primary
// List transfer. CompleteResaleTransferFulfillment must atomically persist:
//
//  1. canonical Order item transferred=true,
//  2. orderTransferItems projection transferred=true,
//  3. matching resale Settlement pending -> ready.
//
// These writes must commit in one persistence transaction so seller payout can
// never become eligible independently from the Order transfer state.
type OrderRepoForTransfer interface {
	// FindEligibleTransferItem returns orderdom.ErrNotFound when no paid,
	// untransferred item exactly matches the input.
	//
	// Return-requested items must remain discoverable so a valid scan can
	// record tokenTransferVerifiedAt before token transfer is blocked.
	FindEligibleTransferItem(
		ctx context.Context,
		in FindEligibleTransferItemInput,
	) (TransferTargetItem, error)

	MarkTokenTransferVerified(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) (orderdom.OrderItemSnapshot, error)

	LockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		now time.Time,
	) error

	UnlockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
	) error

	// MarkTransferredItem completes a primary List transfer.
	//
	// Resale transfers must use CompleteResaleTransferFulfillment so the Order
	// transfer state and Settlement readiness are committed atomically.
	MarkTransferredItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) error

	// CompleteResaleTransferFulfillment atomically completes one successfully
	// executed resale token transfer and crosses the seller payout boundary.
	//
	// Implementations must validate that:
	//
	// - Order and projection identify the requested item.
	// - Order is paid.
	// - Item type is resale.
	// - Item is not canceled.
	// - Item has no active or completed return.
	// - Item has not already been transferred.
	// - tokenTransferVerifiedAt exists.
	// - Settlement belongs to the same Order / Payment.
	// - Settlement seller exactly equals seller.
	// - Settlement status is pending.
	//
	// On success the same persistence transaction must:
	//
	// - mark the canonical Order item transferred,
	// - mark the orderTransferItems projection transferred,
	// - clear the transfer lock,
	// - transition Settlement pending -> ready.
	//
	// The returned Settlement must be the persisted ready Settlement.
	CompleteResaleTransferFulfillment(
		ctx context.Context,
		orderID string,
		itemIndex int,
		settlementID string,
		seller settlementdom.SellerIdentity,
		at time.Time,
	) (settlementdom.Settlement, error)
}
