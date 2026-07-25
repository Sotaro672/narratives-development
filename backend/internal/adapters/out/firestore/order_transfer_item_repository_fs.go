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

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
	transferdom "narratives/internal/domain/transfer"
)

var (
	ErrOrderTransferItemRepoNotConfigured = errors.New(
		"order_transfer_item_repo_fs: not configured",
	)
	ErrInvalidOrderTransferItemData = errors.New(
		"order_transfer_item_repo_fs: invalid projection data",
	)
	ErrInvalidTransferOrderID = errors.New(
		"order_transfer_item_repo_fs: orderId is empty",
	)
	ErrInvalidTransferItemIndex = errors.New(
		"order_transfer_item_repo_fs: itemIndex is invalid",
	)
	ErrInvalidTransferAvatarID = errors.New(
		"order_transfer_item_repo_fs: avatarId is empty",
	)
	ErrInvalidTransferProductID = errors.New(
		"order_transfer_item_repo_fs: productId is empty",
	)
	ErrInvalidTransferTokenBlueprintID = errors.New(
		"order_transfer_item_repo_fs: tokenBlueprintId is empty",
	)
	ErrOrderNotPaid = errors.New(
		"order_transfer_item_repo_fs: order is not paid",
	)
	ErrTransferItemTransferred = errors.New(
		"order_transfer_item_repo_fs: item already transferred",
	)
	ErrTransferItemLocked = errors.New(
		"order_transfer_item_repo_fs: item is locked",
	)
	ErrTransferItemProjectionMismatch = errors.New(
		"order_transfer_item_repo_fs: order and projection do not match",
	)
)

const defaultTransferLockTTL = 10 * time.Minute

// OrderRepoForTransferFS reads and updates the canonical orderTransferItems
// projection. Order documents are read only when MarkTransferredItem must
// update the Order aggregate and its projection atomically.
type OrderRepoForTransferFS struct {
	Client *firestore.Client

	Collection string
	LockTTL    time.Duration
}

var _ usecase.OrderRepoForTransfer = (*OrderRepoForTransferFS)(nil)

func NewOrderRepoForTransferFS(
	client *firestore.Client,
) *OrderRepoForTransferFS {
	return &OrderRepoForTransferFS{
		Client: client,
	}
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

func (r *OrderRepoForTransferFS) transferItemDocID(
	orderID string,
	itemIndex int,
) string {
	return orderID + "__" + strconv.Itoa(itemIndex)
}

func (r *OrderRepoForTransferFS) transferItemDoc(
	orderID string,
	itemIndex int,
) *firestore.DocumentRef {
	return r.transferItemsCol().Doc(
		r.transferItemDocID(orderID, itemIndex),
	)
}

func (r *OrderRepoForTransferFS) lockTTL() time.Duration {
	if r.LockTTL > 0 {
		return r.LockTTL
	}

	return defaultTransferLockTTL
}

func (r *OrderRepoForTransferFS) FindEligibleTransferItem(
	ctx context.Context,
	in usecase.FindEligibleTransferItemInput,
) (usecase.TransferTargetItem, error) {
	if r == nil || r.Client == nil {
		return usecase.TransferTargetItem{},
			ErrOrderTransferItemRepoNotConfigured
	}
	if in.AvatarID == "" {
		return usecase.TransferTargetItem{},
			ErrInvalidTransferAvatarID
	}
	if in.ProductID == "" {
		return usecase.TransferTargetItem{},
			ErrInvalidTransferProductID
	}
	if in.TokenBlueprintID == "" {
		return usecase.TransferTargetItem{},
			ErrInvalidTransferTokenBlueprintID
	}

	// Resale items are resolved first because productId identifies the item.
	resaleQuery := r.transferItemsCol().
		Where("avatarId", "==", in.AvatarID).
		Where("paid", "==", true).
		Where("transferred", "==", false).
		Where(
			"itemType",
			"==",
			string(orderdom.OrderItemTypeResale),
		).
		Where("productId", "==", in.ProductID).
		Where(
			"tokenBlueprintId",
			"==",
			in.TokenBlueprintID,
		).
		OrderBy("createdAt", firestore.Asc).
		Limit(1)

	target, err := r.findOneEligibleTransferItem(
		ctx,
		resaleQuery,
	)
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, orderdom.ErrNotFound) {
		return usecase.TransferTargetItem{}, err
	}

	if in.ModelID == "" {
		return usecase.TransferTargetItem{},
			orderdom.ErrNotFound
	}

	listQuery := r.transferItemsCol().
		Where("avatarId", "==", in.AvatarID).
		Where("paid", "==", true).
		Where("transferred", "==", false).
		Where(
			"itemType",
			"==",
			string(orderdom.OrderItemTypeList),
		).
		Where("modelId", "==", in.ModelID).
		Where(
			"tokenBlueprintId",
			"==",
			in.TokenBlueprintID,
		).
		OrderBy("createdAt", firestore.Asc).
		Limit(1)

	return r.findOneEligibleTransferItem(
		ctx,
		listQuery,
	)
}

func (r *OrderRepoForTransferFS) findOneEligibleTransferItem(
	ctx context.Context,
	query firestore.Query,
) (usecase.TransferTargetItem, error) {
	iter := query.Documents(ctx)
	defer iter.Stop()

	snap, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return usecase.TransferTargetItem{},
				orderdom.ErrNotFound
		}

		return usecase.TransferTargetItem{}, err
	}

	projection, err :=
		orderTransferItemFromSnapshot(snap)
	if err != nil {
		return usecase.TransferTargetItem{}, err
	}
	if projection.Transferred || !projection.Paid {
		return usecase.TransferTargetItem{},
			ErrInvalidOrderTransferItemData
	}

	return projection.toTransferTarget(), nil
}

func (r *OrderRepoForTransferFS) ListEligibleTransferItemsByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]orderdom.EligibleTransferItem, error) {
	if r == nil || r.Client == nil {
		return nil,
			ErrOrderTransferItemRepoNotConfigured
	}
	if avatarID == "" {
		return nil, ErrInvalidTransferAvatarID
	}

	iter := r.transferItemsCol().
		Where("avatarId", "==", avatarID).
		Where("paid", "==", true).
		Where("transferred", "==", false).
		OrderBy("createdAt", firestore.Asc).
		OrderBy("itemIndex", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	items := make(
		[]orderdom.EligibleTransferItem,
		0,
	)

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		projection, err :=
			orderTransferItemFromSnapshot(snap)
		if err != nil {
			return nil, err
		}
		if projection.AvatarID != avatarID ||
			!projection.Paid ||
			projection.Transferred {
			return nil,
				ErrInvalidOrderTransferItemData
		}

		item :=
			projection.toEligibleTransferItem()
		if err := item.Validate(); err != nil {
			return nil, fmt.Errorf(
				"order transfer item %s: %w",
				snap.Ref.ID,
				err,
			)
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *OrderRepoForTransferFS) LockTransferItem(
	ctx context.Context,
	orderID string,
	itemIndex int,
	now time.Time,
) error {
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
	ref := r.transferItemDoc(
		orderID,
		itemIndex,
	)

	return r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snap, err := tx.Get(ref)
			if err != nil {
				return mapOrderTransferItemNotFound(
					err,
				)
			}

			projection, err :=
				orderTransferItemFromSnapshot(snap)
			if err != nil {
				return err
			}
			if projection.OrderID != orderID ||
				projection.ItemIndex != itemIndex {
				return ErrTransferItemProjectionMismatch
			}
			if !projection.Paid {
				return ErrOrderNotPaid
			}
			if projection.Transferred {
				return ErrTransferItemTransferred
			}

			if projection.TransferLockedAt != nil {
				if projection.TransferLockExpiresAt ==
					nil {
					return ErrInvalidOrderTransferItemData
				}
				if projection.
					TransferLockExpiresAt.
					After(now) {
					return ErrTransferItemLocked
				}
			} else if projection.
				TransferLockExpiresAt != nil {
				return ErrInvalidOrderTransferItemData
			}

			return tx.Update(
				ref,
				[]firestore.Update{
					{
						Path:  "transferLockedAt",
						Value: now,
					},
					{
						Path:  "transferLockExpiresAt",
						Value: lockExpiresAt,
					},
				},
			)
		},
	)
}

func (r *OrderRepoForTransferFS) UnlockTransferItem(
	ctx context.Context,
	orderID string,
	itemIndex int,
) error {
	if r == nil || r.Client == nil {
		return ErrOrderTransferItemRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidTransferOrderID
	}
	if itemIndex < 0 {
		return ErrInvalidTransferItemIndex
	}

	ref := r.transferItemDoc(
		orderID,
		itemIndex,
	)

	return r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snap, err := tx.Get(ref)
			if err != nil {
				return mapOrderTransferItemNotFound(
					err,
				)
			}

			projection, err :=
				orderTransferItemFromSnapshot(snap)
			if err != nil {
				return err
			}
			if projection.OrderID != orderID ||
				projection.ItemIndex != itemIndex {
				return ErrTransferItemProjectionMismatch
			}

			return tx.Update(
				ref,
				[]firestore.Update{
					{
						Path:  "transferLockedAt",
						Value: firestore.Delete,
					},
					{
						Path:  "transferLockExpiresAt",
						Value: firestore.Delete,
					},
				},
			)
		},
	)
}

func (r *OrderRepoForTransferFS) MarkTransferredItem(
	ctx context.Context,
	orderID string,
	itemIndex int,
	at time.Time,
) error {
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

	projectionRef := r.transferItemDoc(
		orderID,
		itemIndex,
	)
	orderRef := r.ordersCol().Doc(orderID)

	return r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			projectionSnap, err :=
				tx.Get(projectionRef)
			if err != nil {
				return mapOrderTransferItemNotFound(
					err,
				)
			}

			orderSnap, err := tx.Get(orderRef)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return orderdom.ErrNotFound
				}

				return err
			}

			projection, err :=
				orderTransferItemFromSnapshot(
					projectionSnap,
				)
			if err != nil {
				return err
			}
			if projection.OrderID != orderID ||
				projection.ItemIndex != itemIndex {
				return ErrTransferItemProjectionMismatch
			}
			if !projection.Paid {
				return ErrOrderNotPaid
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
			if !orderItemMatchesProjection(
				order.Items[itemIndex],
				projection,
			) {
				return ErrTransferItemProjectionMismatch
			}
			if order.Items[itemIndex].Transferred {
				return ErrTransferItemTransferred
			}

			if err := order.UpdateItemTransferred(
				itemIndex,
				true,
				at,
			); err != nil {
				return err
			}
			if err := order.Validate(); err != nil {
				return err
			}

			if err := tx.Set(
				orderRef,
				orderToDoc(order),
				firestore.MergeAll,
			); err != nil {
				return err
			}

			return tx.Update(
				projectionRef,
				[]firestore.Update{
					{
						Path:  "transferred",
						Value: true,
					},
					{
						Path:  "transferredAt",
						Value: at,
					},
					{
						Path:  "transferLockedAt",
						Value: firestore.Delete,
					},
					{
						Path:  "transferLockExpiresAt",
						Value: firestore.Delete,
					},
				},
			)
		},
	)
}

type orderTransferItemProjection struct {
	OrderID   string
	AvatarID  string
	ItemType  orderdom.OrderItemType
	ItemIndex int

	Paid          bool
	Transferred   bool
	TransferredAt *time.Time
	CreatedAt     time.Time

	TransferLockedAt      *time.Time
	TransferLockExpiresAt *time.Time

	ModelID     string
	InventoryID string
	ListID      string

	ResaleID string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

func orderTransferItemFromSnapshot(
	snap *firestore.DocumentSnapshot,
) (orderTransferItemProjection, error) {
	if snap == nil ||
		snap.Ref == nil ||
		!snap.Exists() {
		return orderTransferItemProjection{},
			orderdom.ErrNotFound
	}

	raw := snap.Data()
	if raw == nil {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	orderID, ok := requiredOrderTransferItemString(
		raw,
		"orderId",
	)
	if !ok {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	avatarID, ok := requiredOrderTransferItemString(
		raw,
		"avatarId",
	)
	if !ok {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	rawItemType, ok := requiredOrderTransferItemString(
		raw,
		"itemType",
	)
	if !ok {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	itemIndex, ok := requiredOrderTransferItemInt(
		raw,
		"itemIndex",
	)
	if !ok || itemIndex < 0 {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	paid, ok := requiredOrderTransferItemBool(
		raw,
		"paid",
	)
	if !ok {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	transferred, ok := requiredOrderTransferItemBool(
		raw,
		"transferred",
	)
	if !ok {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	createdAt, ok := requiredOrderTransferItemTime(
		raw,
		"createdAt",
	)
	if !ok || createdAt.IsZero() {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	expectedDocID :=
		orderID + "__" + strconv.Itoa(itemIndex)
	if snap.Ref.ID != expectedDocID {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}

	projection := orderTransferItemProjection{
		OrderID:  orderID,
		AvatarID: avatarID,
		ItemType: orderdom.OrderItemType(
			rawItemType,
		),
		ItemIndex:   itemIndex,
		Paid:        paid,
		Transferred: transferred,
		CreatedAt:   createdAt.UTC(),

		ModelID: optionalOrderTransferItemString(
			raw,
			"modelId",
		),
		InventoryID: optionalOrderTransferItemString(
			raw,
			"inventoryId",
		),
		ListID: optionalOrderTransferItemString(
			raw,
			"listId",
		),
		ResaleID: optionalOrderTransferItemString(
			raw,
			"resaleId",
		),
		ProductID: optionalOrderTransferItemString(
			raw,
			"productId",
		),
		ProductBlueprintID: optionalOrderTransferItemString(
			raw,
			"productBlueprintId",
		),
		TokenBlueprintID: optionalOrderTransferItemString(
			raw,
			"tokenBlueprintId",
		),
		BrandID: optionalOrderTransferItemString(
			raw,
			"brandId",
		),
	}

	transferredAt, transferredAtExists, err :=
		optionalOrderTransferItemTime(
			raw,
			"transferredAt",
		)
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	if projection.Transferred !=
		transferredAtExists {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}
	if transferredAtExists {
		projection.TransferredAt =
			&transferredAt
	}

	lockedAt, lockedAtExists, err :=
		optionalOrderTransferItemTime(
			raw,
			"transferLockedAt",
		)
	if err != nil {
		return orderTransferItemProjection{}, err
	}

	lockExpiresAt, lockExpiresAtExists, err :=
		optionalOrderTransferItemTime(
			raw,
			"transferLockExpiresAt",
		)
	if err != nil {
		return orderTransferItemProjection{}, err
	}
	if lockedAtExists != lockExpiresAtExists {
		return orderTransferItemProjection{},
			ErrInvalidOrderTransferItemData
	}
	if lockedAtExists {
		projection.TransferLockedAt = &lockedAt
		projection.TransferLockExpiresAt =
			&lockExpiresAt
	}

	item := projection.toEligibleTransferItem()
	if err := item.Validate(); err != nil {
		return orderTransferItemProjection{},
			fmt.Errorf(
				"order transfer item %s: %w",
				snap.Ref.ID,
				err,
			)
	}

	return projection, nil
}

func (
	p orderTransferItemProjection,
) toEligibleTransferItem() orderdom.EligibleTransferItem {
	return orderdom.EligibleTransferItem{
		OrderID:   p.OrderID,
		ItemType:  p.ItemType,
		ItemIndex: p.ItemIndex,

		ModelID:     p.ModelID,
		InventoryID: p.InventoryID,
		ListID:      p.ListID,

		ResaleID: p.ResaleID,

		ProductID:          p.ProductID,
		ProductBlueprintID: p.ProductBlueprintID,
		TokenBlueprintID:   p.TokenBlueprintID,
		BrandID:            p.BrandID,
	}
}

func (
	p orderTransferItemProjection,
) toTransferTarget() usecase.TransferTargetItem {
	return usecase.TransferTargetItem{
		OrderID:   p.OrderID,
		ItemIndex: p.ItemIndex,
		ItemType:  p.ItemType,

		InventoryID: p.InventoryID,
		ModelID:     p.ModelID,
		ResaleID:    p.ResaleID,

		ProductID:          p.ProductID,
		ProductBlueprintID: p.ProductBlueprintID,
		TokenBlueprintID:   p.TokenBlueprintID,
		BrandID:            p.BrandID,
	}
}

func orderItemMatchesProjection(
	item orderdom.OrderItemSnapshot,
	projection orderTransferItemProjection,
) bool {
	if item.Type != projection.ItemType ||
		item.ProductBlueprintID !=
			projection.ProductBlueprintID ||
		item.TokenBlueprintID !=
			projection.TokenBlueprintID {
		return false
	}

	switch item.Type {
	case orderdom.OrderItemTypeList:
		return item.ModelID ==
			projection.ModelID &&
			item.InventoryID ==
				projection.InventoryID &&
			item.ListID ==
				projection.ListID

	case orderdom.OrderItemTypeResale:
		return item.ResaleID ==
			projection.ResaleID &&
			item.ProductID ==
				projection.ProductID &&
			item.BrandID ==
				projection.BrandID

	default:
		return false
	}
}

func requiredOrderTransferItemString(
	raw map[string]any,
	field string,
) (string, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return "", false
	}

	text, ok := value.(string)
	return text, ok && text != ""
}

func optionalOrderTransferItemString(
	raw map[string]any,
	field string,
) string {
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

func requiredOrderTransferItemBool(
	raw map[string]any,
	field string,
) (bool, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return false, false
	}

	result, ok := value.(bool)
	return result, ok
}

func requiredOrderTransferItemInt(
	raw map[string]any,
	field string,
) (int, bool) {
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

func requiredOrderTransferItemTime(
	raw map[string]any,
	field string,
) (time.Time, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false
	}

	result, ok := value.(time.Time)

	return result,
		ok && !result.IsZero()
}

func optionalOrderTransferItemTime(
	raw map[string]any,
	field string,
) (time.Time, bool, error) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false, nil
	}

	result, ok := value.(time.Time)
	if !ok || result.IsZero() {
		return time.Time{},
			false,
			ErrInvalidOrderTransferItemData
	}

	return result.UTC(), true, nil
}

func mapOrderTransferItemNotFound(
	err error,
) error {
	if status.Code(err) == codes.NotFound {
		return orderdom.ErrNotFound
	}

	return err
}
