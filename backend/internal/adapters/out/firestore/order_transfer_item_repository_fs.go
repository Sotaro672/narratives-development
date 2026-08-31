// backend/internal/adapters/out/firestore/order_transfer_item_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	transferdom "narratives/internal/domain/transfer"
)

var (
	ErrOrderTransferItemRepoNotConfigured = errors.New("order_transfer_item_repo_fs: not configured")
	ErrInvalidOrderTransferItemData       = errors.New("order_transfer_item_repo_fs: invalid projection data")
	ErrInvalidTransferOrderID             = errors.New("order_transfer_item_repo_fs: orderId is empty")
	ErrInvalidTransferItemIndex           = errors.New("order_transfer_item_repo_fs: itemIndex is invalid")
	ErrInvalidTransferAvatarID            = errors.New("order_transfer_item_repo_fs: avatarId is empty")
	ErrInvalidTransferProductID           = errors.New("order_transfer_item_repo_fs: productId is empty")
	ErrInvalidTransferTokenBlueprintID    = errors.New("order_transfer_item_repo_fs: tokenBlueprintId is empty")
	ErrOrderNotPaid                       = errors.New("order_transfer_item_repo_fs: order is not paid")
	ErrTransferItemCancelled              = errors.New("order_transfer_item_repo_fs: item is cancelled")
	ErrTransferItemTransferred            = errors.New("order_transfer_item_repo_fs: item already transferred")
	ErrTransferItemLocked                 = errors.New("order_transfer_item_repo_fs: item is locked")
	ErrTransferItemProjectionMismatch     = errors.New("order_transfer_item_repo_fs: order and projection do not match")
)

const defaultTransferLockTTL = 10 * time.Minute

// OrderRepoForTransferFS reads and updates the canonical orderTransferItems
// projection. Order documents are also read for verified-scan recording,
// transfer locking, and final transferred-state updates.
type OrderRepoForTransferFS struct {
	Client     *firestore.Client
	Collection string
	LockTTL    time.Duration
}

var _ applicationport.OrderRepoForTransfer = (*OrderRepoForTransferFS)(nil)

func NewOrderRepoForTransferFS(client *firestore.Client) *OrderRepoForTransferFS {
	return &OrderRepoForTransferFS{Client: client}
}

func (r *OrderRepoForTransferFS) transferItemsCol() *firestore.CollectionRef {
	collection := r.Collection
	if collection == "" {
		collection = "orderTransferItems"
	}
	return r.Client.Collection(collection)
}

func (r *OrderRepoForTransferFS) ordersCol() *firestore.CollectionRef {
	return r.Client.Collection("orders")
}

func (r *OrderRepoForTransferFS) transferItemDocID(orderID string, itemIndex int) string {
	return orderID + "__" + strconv.Itoa(itemIndex)
}

func (r *OrderRepoForTransferFS) transferItemDoc(orderID string, itemIndex int) *firestore.DocumentRef {
	return r.transferItemsCol().Doc(r.transferItemDocID(orderID, itemIndex))
}

func (r *OrderRepoForTransferFS) lockTTL() time.Duration {
	if r.LockTTL > 0 {
		return r.LockTTL
	}
	return defaultTransferLockTTL
}

func (r *OrderRepoForTransferFS) FindEligibleTransferItem(ctx context.Context, in applicationport.FindEligibleTransferItemInput) (applicationport.TransferTargetItem, error) {
	if r == nil || r.Client == nil {
		return applicationport.TransferTargetItem{}, ErrOrderTransferItemRepoNotConfigured
	}
	if in.AvatarID == "" {
		return applicationport.TransferTargetItem{}, ErrInvalidTransferAvatarID
	}
	if in.ProductID == "" {
		return applicationport.TransferTargetItem{}, ErrInvalidTransferProductID
	}
	if in.TokenBlueprintID == "" {
		return applicationport.TransferTargetItem{}, ErrInvalidTransferTokenBlueprintID
	}

	// Resale items are resolved first because productId identifies the item.
	resaleQuery := r.transferItemsCol().
		Where("avatarId", "==", in.AvatarID).
		Where("paid", "==", true).
		Where("isCancelled", "==", false).
		Where("transferred", "==", false).
		Where("itemType", "==", string(orderdom.OrderItemTypeResale)).
		Where("productId", "==", in.ProductID).
		Where("tokenBlueprintId", "==", in.TokenBlueprintID).
		OrderBy("createdAt", firestore.Asc)

	target, err := r.findOneEligibleTransferItem(ctx, resaleQuery)
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, orderdom.ErrNotFound) {
		return applicationport.TransferTargetItem{}, err
	}
	if in.ModelID == "" {
		return applicationport.TransferTargetItem{}, orderdom.ErrNotFound
	}

	listQuery := r.transferItemsCol().
		Where("avatarId", "==", in.AvatarID).
		Where("paid", "==", true).
		Where("isCancelled", "==", false).
		Where("transferred", "==", false).
		Where("itemType", "==", string(orderdom.OrderItemTypeList)).
		Where("modelId", "==", in.ModelID).
		Where("tokenBlueprintId", "==", in.TokenBlueprintID).
		OrderBy("createdAt", firestore.Asc)

	return r.findOneEligibleTransferItem(ctx, listQuery)
}

func (r *OrderRepoForTransferFS) findOneEligibleTransferItem(ctx context.Context, query firestore.Query) (applicationport.TransferTargetItem, error) {
	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				return applicationport.TransferTargetItem{}, orderdom.ErrNotFound
			}
			return applicationport.TransferTargetItem{}, err
		}

		projection, err := orderTransferItemFromSnapshot(snap)
		if err != nil {
			return applicationport.TransferTargetItem{}, err
		}
		if projection.IsCancelled || projection.Transferred || !projection.Paid {
			return applicationport.TransferTargetItem{}, ErrInvalidOrderTransferItemData
		}

		// New projections carry isReturnCompleted directly. Legacy projections
		// may not have the field, so canonical Order state is checked below.
		if projection.IsReturnCompleted {
			continue
		}

		returnCompleted, err := r.isReturnCompletedInCanonicalOrder(ctx, projection)
		if err != nil {
			return applicationport.TransferTargetItem{}, err
		}
		if returnCompleted {
			continue
		}
		return projection.toTransferTarget(), nil
	}
}

func (r *OrderRepoForTransferFS) isReturnCompletedInCanonicalOrder(ctx context.Context, projection orderTransferItemProjection) (bool, error) {
	orderSnap, err := r.ordersCol().Doc(projection.OrderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, orderdom.ErrNotFound
		}
		return false, err
	}

	order, err := docToOrder(orderSnap)
	if err != nil {
		return false, err
	}
	if order.AvatarID != projection.AvatarID || projection.ItemIndex < 0 || projection.ItemIndex >= len(order.Items) {
		return false, ErrTransferItemProjectionMismatch
	}

	item := order.Items[projection.ItemIndex]
	if !orderItemMatchesProjection(item, projection) {
		return false, ErrTransferItemProjectionMismatch
	}
	return item.IsReturnCompleted, nil
}

func (r *OrderRepoForTransferFS) ListEligibleTransferItemsByAvatarID(ctx context.Context, avatarID string) ([]orderdom.EligibleTransferItem, error) {
	if r == nil || r.Client == nil {
		return nil, ErrOrderTransferItemRepoNotConfigured
	}
	if avatarID == "" {
		return nil, ErrInvalidTransferAvatarID
	}

	iter := r.transferItemsCol().
		Where("avatarId", "==", avatarID).
		Where("paid", "==", true).
		Where("isCancelled", "==", false).
		Where("transferred", "==", false).
		OrderBy("createdAt", firestore.Asc).
		OrderBy("itemIndex", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	items := make([]orderdom.EligibleTransferItem, 0)
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		projection, err := orderTransferItemFromSnapshot(snap)
		if err != nil {
			return nil, err
		}
		if projection.AvatarID != avatarID || !projection.Paid || projection.IsCancelled || projection.Transferred {
			return nil, ErrInvalidOrderTransferItemData
		}
		if projection.IsReturnRequested || projection.IsReturnCompleted {
			continue
		}

		item := projection.toEligibleTransferItem()
		if err := item.Validate(); err != nil {
			return nil, fmt.Errorf("order transfer item %s: %w", snap.Ref.ID, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *OrderRepoForTransferFS) LockTransferItem(ctx context.Context, orderID string, itemIndex int, now time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return ErrInvalidTransferItemIndex
	}
	if now.IsZero() {
		return transferdom.ErrInvalidCreatedAt
	}

	now = now.UTC()
	lockExpiresAt := now.Add(r.lockTTL())
	projectionRef := r.transferItemDoc(orderID, itemIndex)
	orderRef := r.ordersCol().Doc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		projectionSnap, err := tx.Get(projectionRef)
		if err != nil {
			return mapOrderTransferItemNotFound(err)
		}
		orderSnap, err := tx.Get(orderRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}

		projection, err := orderTransferItemFromSnapshot(projectionSnap)
		if err != nil {
			return err
		}
		if projection.OrderID != orderID || projection.ItemIndex != itemIndex {
			return ErrTransferItemProjectionMismatch
		}
		if !projection.Paid {
			return ErrOrderNotPaid
		}
		if projection.IsCancelled {
			return ErrTransferItemCancelled
		}
		if projection.IsReturnCompleted || projection.IsReturnRequested {
			return orderdom.ErrConflict
		}
		if projection.Transferred {
			return ErrTransferItemTransferred
		}

		order, err := docToOrder(orderSnap)
		if err != nil {
			return err
		}
		if !order.Paid {
			return ErrOrderNotPaid
		}
		if itemIndex >= len(order.Items) {
			return ErrTransferItemProjectionMismatch
		}
		if order.Items[itemIndex].IsCancelled {
			return ErrTransferItemCancelled
		}
		if order.Items[itemIndex].IsReturnCompleted || order.Items[itemIndex].IsReturnRequested {
			return orderdom.ErrConflict
		}
		if order.Items[itemIndex].Transferred {
			return ErrTransferItemTransferred
		}
		if !orderItemMatchesProjection(order.Items[itemIndex], projection) {
			return ErrTransferItemProjectionMismatch
		}

		if projection.TransferLockedAt != nil {
			if projection.TransferLockExpiresAt == nil {
				return ErrInvalidOrderTransferItemData
			}
			if projection.TransferLockExpiresAt.After(now) {
				return ErrTransferItemLocked
			}
		} else if projection.TransferLockExpiresAt != nil {
			return ErrInvalidOrderTransferItemData
		}

		return tx.Update(projectionRef, []firestore.Update{
			{Path: "transferLockedAt", Value: now},
			{Path: "transferLockExpiresAt", Value: lockExpiresAt},
		})
	})
}

func (r *OrderRepoForTransferFS) MarkTokenTransferVerified(ctx context.Context, orderID string, itemIndex int, at time.Time) (orderdom.OrderItemSnapshot, error) {
	if r == nil || r.Client == nil {
		return orderdom.OrderItemSnapshot{}, ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return orderdom.OrderItemSnapshot{}, ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return orderdom.OrderItemSnapshot{}, ErrInvalidTransferItemIndex
	}
	if at.IsZero() {
		return orderdom.OrderItemSnapshot{}, transferdom.ErrInvalidCreatedAt
	}

	at = at.UTC()
	projectionRef := r.transferItemDoc(orderID, itemIndex)
	orderRef := r.ordersCol().Doc(orderID)
	var updatedItem orderdom.OrderItemSnapshot

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		projectionSnap, err := tx.Get(projectionRef)
		if err != nil {
			return mapOrderTransferItemNotFound(err)
		}
		orderSnap, err := tx.Get(orderRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}

		projection, err := orderTransferItemFromSnapshot(projectionSnap)
		if err != nil {
			return err
		}
		if projection.OrderID != orderID || projection.ItemIndex != itemIndex {
			return ErrTransferItemProjectionMismatch
		}
		if !projection.Paid {
			return ErrOrderNotPaid
		}
		if projection.IsCancelled {
			return ErrTransferItemCancelled
		}
		if projection.IsReturnCompleted {
			return orderdom.ErrConflict
		}
		if projection.Transferred {
			return ErrTransferItemTransferred
		}

		order, err := docToOrder(orderSnap)
		if err != nil {
			return err
		}
		if !order.Paid {
			return ErrOrderNotPaid
		}
		if itemIndex >= len(order.Items) {
			return ErrTransferItemProjectionMismatch
		}
		if order.Items[itemIndex].IsCancelled {
			return ErrTransferItemCancelled
		}
		if order.Items[itemIndex].IsReturnCompleted {
			return orderdom.ErrConflict
		}
		if order.Items[itemIndex].Transferred {
			return ErrTransferItemTransferred
		}
		if !orderItemMatchesProjection(order.Items[itemIndex], projection) {
			return ErrTransferItemProjectionMismatch
		}

		if err := order.MarkItemTokenTransferVerified(itemIndex, at); err != nil {
			return err
		}
		if err := order.Validate(); err != nil {
			return err
		}

		verifiedAt := order.Items[itemIndex].TokenTransferVerifiedAt
		if verifiedAt == nil || verifiedAt.IsZero() {
			return ErrInvalidOrderTransferItemData
		}

		if err := tx.Set(orderRef, orderToDoc(order), firestore.MergeAll); err != nil {
			return err
		}
		if err := tx.Set(projectionRef, map[string]any{
			"isReturnRequested":       order.Items[itemIndex].IsReturnRequested,
			"isReturnCompleted":       order.Items[itemIndex].IsReturnCompleted,
			"tokenTransferVerifiedAt": verifiedAt.UTC(),
		}, firestore.MergeAll); err != nil {
			return err
		}

		updatedItem = order.Items[itemIndex]
		return nil
	})
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}
	return updatedItem, nil
}

func (r *OrderRepoForTransferFS) UnlockTransferItem(ctx context.Context, orderID string, itemIndex int) error {
	if r == nil || r.Client == nil {
		return ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return ErrInvalidTransferItemIndex
	}

	ref := r.transferItemDoc(orderID, itemIndex)
	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return mapOrderTransferItemNotFound(err)
		}
		projection, err := orderTransferItemFromSnapshot(snap)
		if err != nil {
			return err
		}
		if projection.OrderID != orderID || projection.ItemIndex != itemIndex {
			return ErrTransferItemProjectionMismatch
		}
		return tx.Update(ref, []firestore.Update{
			{Path: "transferLockedAt", Value: firestore.Delete},
			{Path: "transferLockExpiresAt", Value: firestore.Delete},
		})
	})
}

func (r *OrderRepoForTransferFS) MarkTransferredItem(ctx context.Context, orderID string, itemIndex int, at time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return ErrInvalidTransferItemIndex
	}
	if at.IsZero() {
		return transferdom.ErrInvalidTransferredAt
	}

	at = at.UTC()
	projectionRef := r.transferItemDoc(orderID, itemIndex)
	orderRef := r.ordersCol().Doc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		projectionSnap, err := tx.Get(projectionRef)
		if err != nil {
			return mapOrderTransferItemNotFound(err)
		}
		orderSnap, err := tx.Get(orderRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}

		projection, err := orderTransferItemFromSnapshot(projectionSnap)
		if err != nil {
			return err
		}
		if projection.OrderID != orderID || projection.ItemIndex != itemIndex {
			return ErrTransferItemProjectionMismatch
		}
		if !projection.Paid {
			return ErrOrderNotPaid
		}
		if projection.IsCancelled {
			return ErrTransferItemCancelled
		}
		if projection.IsReturnCompleted || projection.IsReturnRequested {
			return orderdom.ErrConflict
		}
		if projection.Transferred {
			return ErrTransferItemTransferred
		}

		order, err := docToOrder(orderSnap)
		if err != nil {
			return err
		}
		if !order.Paid {
			return ErrOrderNotPaid
		}
		if itemIndex >= len(order.Items) {
			return ErrTransferItemProjectionMismatch
		}
		if order.Items[itemIndex].IsCancelled {
			return ErrTransferItemCancelled
		}
		if order.Items[itemIndex].IsReturnCompleted || order.Items[itemIndex].IsReturnRequested {
			return orderdom.ErrConflict
		}
		if !orderItemMatchesProjection(order.Items[itemIndex], projection) {
			return ErrTransferItemProjectionMismatch
		}
		if order.Items[itemIndex].Transferred {
			return ErrTransferItemTransferred
		}

		if err := order.UpdateItemTransferred(itemIndex, true, at); err != nil {
			return err
		}
		if err := order.Validate(); err != nil {
			return err
		}
		if err := tx.Set(orderRef, orderToDoc(order), firestore.MergeAll); err != nil {
			return err
		}

		verifiedAt := order.Items[itemIndex].TokenTransferVerifiedAt
		if verifiedAt == nil || verifiedAt.IsZero() {
			return ErrInvalidOrderTransferItemData
		}

		return tx.Update(projectionRef, []firestore.Update{
			{Path: "tokenTransferVerifiedAt", Value: verifiedAt.UTC()},
			{Path: "isReturnRequested", Value: false},
			{Path: "isReturnCompleted", Value: false},
			{Path: "transferred", Value: true},
			{Path: "transferredAt", Value: at},
			{Path: "transferLockedAt", Value: firestore.Delete},
			{Path: "transferLockExpiresAt", Value: firestore.Delete},
		})
	})
}

// CompleteResaleReceivableFulfillment atomically completes one successfully
// executed resale token transfer and makes that exact item's SalesReceivable
// available for a future BankPayout.
//
// One resale Order item maps to exactly one SalesReceivable identified by
// PaymentID + OrderItemIndex. The canonical Order item, orderTransferItems
// projection, and SalesReceivable pending -> available transition are committed
// in the same Firestore transaction.
//
// No other resale item belonging to the same seller participates in this
// fulfillment boundary. No Stripe Settlement or Stripe Transfer state is
// touched by this operation.
func (r *OrderRepoForTransferFS) CompleteResaleReceivableFulfillment(
	ctx context.Context,
	orderID string,
	itemIndex int,
	expected salesreceivabledom.SalesReceivable,
	at time.Time,
) (salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return salesreceivabledom.SalesReceivable{}, ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return salesreceivabledom.SalesReceivable{}, ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return salesreceivabledom.SalesReceivable{}, ErrInvalidTransferItemIndex
	}
	if at.IsZero() {
		return salesreceivabledom.SalesReceivable{}, transferdom.ErrInvalidTransferredAt
	}
	if err := expected.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if expected.OrderID != orderID ||
		expected.PaymentID != orderID ||
		expected.OrderItemIndex != itemIndex ||
		expected.ResaleID == "" ||
		expected.Status != salesreceivabledom.StatusPending {
		return salesreceivabledom.SalesReceivable{}, salesreceivabledom.ErrConflict
	}

	expectedID, err := salesreceivabledom.NewID(orderID, itemIndex)
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if expected.ID != expectedID {
		return salesreceivabledom.SalesReceivable{}, salesreceivabledom.ErrConflict
	}

	at = at.UTC()
	projectionRef := r.transferItemDoc(orderID, itemIndex)
	orderRef := r.ordersCol().Doc(orderID)
	receivableRef := r.Client.Collection(salesReceivablesCollection).Doc(expected.ID)
	var result salesreceivabledom.SalesReceivable

	err = r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Firestore transactions require all reads before writes.
		projectionSnap, err := tx.Get(projectionRef)
		if err != nil {
			return mapOrderTransferItemNotFound(err)
		}
		orderSnap, err := tx.Get(orderRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}
		receivableSnap, err := tx.Get(receivableRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return salesreceivabledom.ErrNotFound
			}
			return err
		}

		projection, err := orderTransferItemFromSnapshot(projectionSnap)
		if err != nil {
			return err
		}
		if projection.OrderID != orderID ||
			projection.ItemIndex != itemIndex ||
			projection.ItemType != orderdom.OrderItemTypeResale ||
			projection.ResaleID != expected.ResaleID {
			return ErrTransferItemProjectionMismatch
		}
		if !projection.Paid {
			return ErrOrderNotPaid
		}
		if projection.IsCancelled {
			return ErrTransferItemCancelled
		}
		if projection.IsReturnRequested || projection.IsReturnCompleted {
			return orderdom.ErrConflict
		}
		if projection.Transferred {
			return ErrTransferItemTransferred
		}
		if projection.TokenTransferVerifiedAt == nil || projection.TokenTransferVerifiedAt.IsZero() {
			return ErrInvalidOrderTransferItemData
		}

		order, err := docToOrder(orderSnap)
		if err != nil {
			return err
		}
		if order.ID != orderID || !order.Paid || itemIndex >= len(order.Items) {
			if !order.Paid {
				return ErrOrderNotPaid
			}
			return ErrTransferItemProjectionMismatch
		}

		item := order.Items[itemIndex]
		if item.Type != orderdom.OrderItemTypeResale ||
			item.ResaleID != expected.ResaleID ||
			item.Qty != 1 ||
			item.Price <= 0 ||
			item.Price != expected.GrossAmount {
			return ErrTransferItemProjectionMismatch
		}
		if item.IsCancelled {
			return ErrTransferItemCancelled
		}
		if item.IsReturnRequested || item.IsReturnCompleted {
			return orderdom.ErrConflict
		}
		if item.Transferred {
			return ErrTransferItemTransferred
		}
		if !orderItemMatchesProjection(item, projection) {
			return ErrTransferItemProjectionMismatch
		}
		if item.TokenTransferVerifiedAt == nil ||
			item.TokenTransferVerifiedAt.IsZero() ||
			!item.TokenTransferVerifiedAt.Equal(*projection.TokenTransferVerifiedAt) {
			return ErrInvalidOrderTransferItemData
		}

		receivable, err := docToSalesReceivable(receivableSnap)
		if err != nil {
			return err
		}
		if !salesReceivableImmutableFieldsEqual(receivable, expected) {
			return salesreceivabledom.ErrConflict
		}
		if receivable.OrderItemIndex != itemIndex ||
			receivable.ResaleID != item.ResaleID ||
			receivable.GrossAmount != item.Price {
			return salesreceivabledom.ErrConflict
		}
		if receivable.Status != salesreceivabledom.StatusPending {
			return salesreceivabledom.ErrInvalidStatusTransition
		}
		if !resaleSellerSnapshotMatchesReceivable(item.SellerSnapshot, receivable) {
			return salesreceivabledom.ErrConflict
		}

		if err := order.UpdateItemTransferred(itemIndex, true, at); err != nil {
			return err
		}
		if err := order.Validate(); err != nil {
			return err
		}
		if err := receivable.MarkAvailable(at); err != nil {
			return err
		}
		normalizeSalesReceivableTimestamps(&receivable)
		if err := receivable.Validate(); err != nil {
			return err
		}

		verifiedAt := order.Items[itemIndex].TokenTransferVerifiedAt
		if verifiedAt == nil || verifiedAt.IsZero() {
			return ErrInvalidOrderTransferItemData
		}

		if err := tx.Set(orderRef, orderToDoc(order), firestore.MergeAll); err != nil {
			return err
		}
		if err := tx.Update(projectionRef, []firestore.Update{
			{Path: "tokenTransferVerifiedAt", Value: verifiedAt.UTC()},
			{Path: "isReturnRequested", Value: false},
			{Path: "isReturnCompleted", Value: false},
			{Path: "transferred", Value: true},
			{Path: "transferredAt", Value: at},
			{Path: "transferLockedAt", Value: firestore.Delete},
			{Path: "transferLockExpiresAt", Value: firestore.Delete},
		}); err != nil {
			return err
		}
		if err := tx.Set(receivableRef, receivable); err != nil {
			return err
		}

		result = receivable
		return nil
	})
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}

	return result, nil
}

func salesReceivableImmutableFieldsEqual(left, right salesreceivabledom.SalesReceivable) bool {
	return left.ID == right.ID &&
		left.OrderID == right.OrderID &&
		left.PaymentID == right.PaymentID &&
		left.OrderItemIndex == right.OrderItemIndex &&
		left.ResaleID == right.ResaleID &&
		left.AvatarID == right.AvatarID &&
		left.UserID == right.UserID &&
		left.PayoutAccountID == right.PayoutAccountID &&
		left.GrossAmount == right.GrossAmount &&
		left.PlatformFeeAmount == right.PlatformFeeAmount &&
		left.ReceivableAmount == right.ReceivableAmount &&
		left.Currency == right.Currency &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func resaleSellerSnapshotMatchesReceivable(seller orderdom.SellerSnapshot, receivable salesreceivabledom.SalesReceivable) bool {
	return seller.AvatarID == receivable.AvatarID &&
		seller.UserID == receivable.UserID &&
		seller.PayoutAccountID == receivable.PayoutAccountID &&
		seller.PayoutAccountID == seller.UserID &&
		seller.BrandID == "" &&
		seller.CompanyID == "" &&
		seller.AccountID == "" &&
		seller.StripeAccountID == ""
}

type orderTransferItemProjection struct {
	OrderID   string
	AvatarID  string
	ItemType  orderdom.OrderItemType
	ItemIndex int

	Paid                    bool
	IsCancelled             bool
	IsReturnRequested       bool
	IsReturnCompleted       bool
	TokenTransferVerifiedAt *time.Time
	Transferred             bool
	TransferredAt           *time.Time
	CreatedAt               time.Time

	TransferLockedAt      *time.Time
	TransferLockExpiresAt *time.Time

	ModelID     string
	InventoryID string
	ListID      string
	ResaleID    string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

func orderTransferItemFromSnapshot(snap *firestore.DocumentSnapshot) (orderTransferItemProjection, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return orderTransferItemProjection{}, orderdom.ErrNotFound
	}

	raw := snap.Data()
	if raw == nil {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}

	orderID, ok := requiredOrderTransferItemString(raw, "orderId")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	avatarID, ok := requiredOrderTransferItemString(raw, "avatarId")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	rawItemType, ok := requiredOrderTransferItemString(raw, "itemType")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	itemIndex, ok := requiredOrderTransferItemInt(raw, "itemIndex")
	if !ok || itemIndex < 0 {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	paid, ok := requiredOrderTransferItemBool(raw, "paid")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	isCancelled, ok := requiredOrderTransferItemBool(raw, "isCancelled")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	isReturnRequested, _, err := optionalOrderTransferItemBool(raw, "isReturnRequested")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	isReturnCompleted, _, err := optionalOrderTransferItemBool(raw, "isReturnCompleted")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	transferred, ok := requiredOrderTransferItemBool(raw, "transferred")
	if !ok {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	createdAt, ok := requiredOrderTransferItemTime(raw, "createdAt")
	if !ok || createdAt.IsZero() {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}

	expectedDocID := orderID + "__" + strconv.Itoa(itemIndex)
	if snap.Ref.ID != expectedDocID {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}

	projection := orderTransferItemProjection{
		OrderID:            orderID,
		AvatarID:           avatarID,
		ItemType:           orderdom.OrderItemType(rawItemType),
		ItemIndex:          itemIndex,
		Paid:               paid,
		IsCancelled:        isCancelled,
		IsReturnRequested:  isReturnRequested,
		IsReturnCompleted:  isReturnCompleted,
		Transferred:        transferred,
		CreatedAt:          createdAt.UTC(),
		ModelID:            optionalOrderTransferItemString(raw, "modelId"),
		InventoryID:        optionalOrderTransferItemString(raw, "inventoryId"),
		ListID:             optionalOrderTransferItemString(raw, "listId"),
		ResaleID:           optionalOrderTransferItemString(raw, "resaleId"),
		ProductID:          optionalOrderTransferItemString(raw, "productId"),
		ProductBlueprintID: optionalOrderTransferItemString(raw, "productBlueprintId"),
		TokenBlueprintID:   optionalOrderTransferItemString(raw, "tokenBlueprintId"),
		BrandID:            optionalOrderTransferItemString(raw, "brandId"),
	}

	tokenTransferVerifiedAt, tokenTransferVerifiedAtExists, err := optionalOrderTransferItemTime(raw, "tokenTransferVerifiedAt")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	if tokenTransferVerifiedAtExists {
		projection.TokenTransferVerifiedAt = &tokenTransferVerifiedAt
	}

	transferredAt, transferredAtExists, err := optionalOrderTransferItemTime(raw, "transferredAt")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	if projection.Transferred != transferredAtExists {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	if transferredAtExists {
		projection.TransferredAt = &transferredAt
	}

	if projection.IsReturnCompleted && !projection.IsReturnRequested {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	if projection.IsCancelled &&
		(projection.IsReturnRequested ||
			projection.IsReturnCompleted ||
			projection.Transferred ||
			projection.TokenTransferVerifiedAt != nil) {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}

	lockedAt, lockedAtExists, err := optionalOrderTransferItemTime(raw, "transferLockedAt")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	lockExpiresAt, lockExpiresAtExists, err := optionalOrderTransferItemTime(raw, "transferLockExpiresAt")
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	if lockedAtExists != lockExpiresAtExists {
		return orderTransferItemProjection{}, ErrInvalidOrderTransferItemData
	}
	if lockedAtExists {
		projection.TransferLockedAt = &lockedAt
		projection.TransferLockExpiresAt = &lockExpiresAt
	}

	item := projection.toEligibleTransferItem()
	if err := item.Validate(); err != nil {
		return orderTransferItemProjection{}, fmt.Errorf("order transfer item %s: %w", snap.Ref.ID, err)
	}

	return projection, nil
}

func (p orderTransferItemProjection) toEligibleTransferItem() orderdom.EligibleTransferItem {
	return orderdom.EligibleTransferItem{
		OrderID:            p.OrderID,
		ItemType:           p.ItemType,
		ItemIndex:          p.ItemIndex,
		ModelID:            p.ModelID,
		InventoryID:        p.InventoryID,
		ListID:             p.ListID,
		ResaleID:           p.ResaleID,
		ProductID:          p.ProductID,
		ProductBlueprintID: p.ProductBlueprintID,
		TokenBlueprintID:   p.TokenBlueprintID,
		BrandID:            p.BrandID,
	}
}

func (p orderTransferItemProjection) toTransferTarget() applicationport.TransferTargetItem {
	return applicationport.TransferTargetItem{
		OrderID:            p.OrderID,
		ItemIndex:          p.ItemIndex,
		ItemType:           p.ItemType,
		InventoryID:        p.InventoryID,
		ModelID:            p.ModelID,
		ResaleID:           p.ResaleID,
		ProductID:          p.ProductID,
		ProductBlueprintID: p.ProductBlueprintID,
		TokenBlueprintID:   p.TokenBlueprintID,
		BrandID:            p.BrandID,
	}
}

func orderItemMatchesProjection(item orderdom.OrderItemSnapshot, projection orderTransferItemProjection) bool {
	if item.Type != projection.ItemType ||
		item.ProductBlueprintID != projection.ProductBlueprintID ||
		item.TokenBlueprintID != projection.TokenBlueprintID {
		return false
	}

	switch item.Type {
	case orderdom.OrderItemTypeList:
		return item.ModelID == projection.ModelID &&
			item.InventoryID == projection.InventoryID &&
			item.ListID == projection.ListID
	case orderdom.OrderItemTypeResale:
		return item.ResaleID == projection.ResaleID &&
			item.ProductID == projection.ProductID &&
			item.BrandID == projection.BrandID
	default:
		return false
	}
}

func requiredOrderTransferItemString(raw map[string]any, field string) (string, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return "", false
	}
	text, ok := value.(string)
	return text, ok && text != ""
}

func optionalOrderTransferItemString(raw map[string]any, field string) string {
	value, exists := raw[field]
	if !exists || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func requiredOrderTransferItemBool(raw map[string]any, field string) (bool, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return false, false
	}
	result, ok := value.(bool)
	return result, ok
}

func optionalOrderTransferItemBool(raw map[string]any, field string) (bool, bool, error) {
	value, exists := raw[field]
	if !exists || value == nil {
		return false, false, nil
	}
	result, ok := value.(bool)
	if !ok {
		return false, false, ErrInvalidOrderTransferItemData
	}
	return result, true, nil
}

func requiredOrderTransferItemInt(raw map[string]any, field string) (int, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return 0, false
	}

	switch number := value.(type) {
	case int:
		return number, true
	case int64:
		return int(number), true
	default:
		return 0, false
	}
}

func requiredOrderTransferItemTime(raw map[string]any, field string) (time.Time, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false
	}
	result, ok := value.(time.Time)
	return result, ok && !result.IsZero()
}

func optionalOrderTransferItemTime(raw map[string]any, field string) (time.Time, bool, error) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false, nil
	}
	result, ok := value.(time.Time)
	if !ok || result.IsZero() {
		return time.Time{}, false, ErrInvalidOrderTransferItemData
	}
	return result.UTC(), true, nil
}

func mapOrderTransferItemNotFound(err error) error {
	if status.Code(err) == codes.NotFound {
		return orderdom.ErrNotFound
	}
	return err
}
