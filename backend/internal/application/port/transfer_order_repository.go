// backend/internal/application/port/transfer_order_repository.go
package port

import (
	"context"
	"time"

	orderdom "narratives/internal/domain/order"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
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
// Implementations must use the canonical orderTransferItems read model and must
// not scan Order documents in memory.
//
// Resale fulfillment has a stronger consistency requirement than a primary List
// transfer. CompleteResaleReceivableFulfillment must atomically persist:
//
//  1. canonical Order item transferred=true,
//  2. orderTransferItems projection transferred=true,
//  3. matching SalesReceivable lifecycle state.
//
// SalesReceivable is aggregated by PaymentID + PayoutAccountID. When multiple
// active resale items belonging to the same seller are represented by one
// receivable, the receivable must remain pending until the final active item has
// been transferred. Only then may it transition pending -> available.
//
// These writes must commit in one persistence transaction so seller proceeds can
// never become payout-eligible independently from the Order transfer state.
type OrderRepoForTransfer interface {
	// FindEligibleTransferItem returns orderdom.ErrNotFound when no paid,
	// untransferred item exactly matches the input.
	//
	// Return-requested items must remain discoverable so a valid scan can record
	// tokenTransferVerifiedAt before token transfer is blocked.
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
	// Resale transfers must use CompleteResaleReceivableFulfillment so Order
	// transfer state and SalesReceivable availability are committed atomically.
	MarkTransferredItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) error

	// CompleteResaleReceivableFulfillment atomically completes one successfully
	// executed resale token transfer and updates its SalesReceivable.
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
	// - SalesReceivable belongs to the same Order / Payment.
	// - SalesReceivable seller identity matches the item's SellerSnapshot.
	// - SalesReceivable immutable allocation matches the persisted document.
	// - SalesReceivable status is pending.
	//
	// On success the same persistence transaction must:
	//
	// - mark the canonical Order item transferred,
	// - mark the orderTransferItems projection transferred,
	// - clear the transfer lock,
	// - keep SalesReceivable pending when another active resale item represented
	//   by the same receivable remains untransferred,
	// - otherwise transition SalesReceivable pending -> available.
	//
	// The returned SalesReceivable must represent the persisted state after the
	// transaction.
	CompleteResaleReceivableFulfillment(
		ctx context.Context,
		orderID string,
		itemIndex int,
		receivable salesreceivabledom.SalesReceivable,
		at time.Time,
	) (salesreceivabledom.SalesReceivable, error)
}
