// backend/internal/adapters/out/firestore/transfer_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
	transferdom "narratives/internal/domain/transfer"
)

// ============================================================
// OrderRepoForTransfer (Firestore)
// - implements usecase.OrderRepoForTransfer
// - This file is for order item transfer lock/mark repository.
// - Adjusted to match actual Firestore orders schema.
// ============================================================

var (
	ErrOrderRepoNotConfigured   = errors.New("order_repo_for_transfer_fs: not configured")
	ErrInvalidOrderID           = errors.New("order_repo_for_transfer_fs: orderId is empty")
	ErrInvalidTransferItemKey   = errors.New("order_repo_for_transfer_fs: itemKey is empty")
	ErrInvalidItemModelID       = errors.New("order_repo_for_transfer_fs: item modelId is empty")
	ErrInvalidTransferAvatarID  = errors.New("order_repo_for_transfer_fs: avatarId is empty")
	ErrOrderNotFound            = errors.New("order_repo_for_transfer_fs: order not found")
	ErrOrderNotPaid             = errors.New("order_repo_for_transfer_fs: order is not paid")
	ErrOrderItemsMissing        = errors.New("order_repo_for_transfer_fs: order items missing/invalid")
	ErrTransferItemNotFound     = errors.New("order_repo_for_transfer_fs: transfer item not found")
	ErrTransferItemTransferred  = errors.New("order_repo_for_transfer_fs: item already transferred")
	ErrTransferItemLocked       = errors.New("order_repo_for_transfer_fs: item is locked")
	ErrTransferItemKeyMalformed = errors.New("order_repo_for_transfer_fs: itemKey is malformed")
)

const defaultTransferLockTTL = 10 * time.Minute

type OrderRepoForTransferFS struct {
	Client *firestore.Client
}

var _ usecase.OrderRepoForTransfer = (*OrderRepoForTransferFS)(nil)

func NewOrderRepoForTransferFS(client *firestore.Client) *OrderRepoForTransferFS {
	return &OrderRepoForTransferFS{
		Client: client,
	}
}

func (r *OrderRepoForTransferFS) ordersCol() *firestore.CollectionRef {
	return r.Client.Collection("orders")
}

func (r *OrderRepoForTransferFS) orderDoc(orderID string) *firestore.DocumentRef {
	return r.ordersCol().Doc(orderID)
}

// ------------------------------------------------------------
// usecase.OrderRepoForTransfer
// ------------------------------------------------------------

// ListPaidByAvatarID returns paid orders for avatar.
//
// Current Firestore order schema:
//   - avatarId: string
//   - cartId: string
//   - createdAt: timestamp
//   - paid: bool
//   - items: []map{
//     type: string
//     inventoryId: string
//     isCanceled: bool
//     isDispatched: bool
//     listId: string
//     modelId: string
//     resaleId: string
//     productId: string
//     productBlueprintId: string
//     tokenBlueprintId: string
//     brandId: string
//     price: int64
//     qty: int64
//     transferred: bool
//     transferredAt: timestamp
//     }
//   - paymentMethodSnapshot: map
//   - shippingSnapshot: map
//   - userId: string
func (r *OrderRepoForTransferFS) ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return nil, ErrOrderRepoNotConfigured
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return nil, ErrInvalidTransferAvatarID
	}

	it := r.ordersCol().
		Where("avatarId", "==", avatarID).
		Documents(ctx)
	defer it.Stop()

	out := make([]orderdom.Order, 0, 8)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		raw := doc.Data()
		if raw == nil {
			continue
		}

		paid, _ := raw["paid"].(bool)
		if !paid {
			continue
		}

		o := orderdom.Order{
			ID:   doc.Ref.ID,
			Paid: paid,
		}

		if v, ok := raw["userId"].(string); ok {
			o.UserID = v
		}
		if v, ok := raw["avatarId"].(string); ok {
			o.AvatarID = v
		}
		if v, ok := raw["cartId"].(string); ok {
			o.CartID = v
		}
		if t, ok := raw["createdAt"].(time.Time); ok && !t.IsZero() {
			o.CreatedAt = t.UTC()
		}

		items, err := parseOrderItems(raw["items"])
		if err != nil {
			return nil, err
		}
		o.Items = items

		out = append(out, o)
	}

	return out, nil
}

// LockTransferItem acquires an item-level lock within an order.
// - fails if already transferred
// - fails if locked and not expired
//
// itemKey policy:
// - list item:   "list:" + modelId
// - resale item: "resale:" + resaleId
func (r *OrderRepoForTransferFS) LockTransferItem(ctx context.Context, orderID string, itemKey string, now time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}

	orderID = strings.TrimSpace(orderID)
	itemKey = strings.TrimSpace(itemKey)

	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemKey == "" {
		return ErrInvalidTransferItemKey
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	ref := r.orderDoc(orderID)
	lockUntil := now.Add(defaultTransferLockTTL)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrOrderNotFound
			}
			return err
		}
		if snap == nil || !snap.Exists() {
			return ErrOrderNotFound
		}

		raw := snap.Data()
		if raw == nil {
			return ErrOrderNotFound
		}

		if p, ok := raw["paid"].(bool); !ok || !p {
			return ErrOrderNotPaid
		}

		items, err := rawOrderItemMaps(raw["items"])
		if err != nil {
			return err
		}

		idx, itMap, err := findItemMapByItemKey(items, itemKey)
		if err != nil {
			return err
		}

		if b, ok := itMap["transferred"].(bool); ok && b {
			return ErrTransferItemTransferred
		}

		if lockedAt, ok := itMap["transferLockedAt"].(time.Time); ok && !lockedAt.IsZero() {
			if exp, ok := itMap["transferLockExpiresAt"].(time.Time); ok && !exp.IsZero() {
				if exp.After(now) {
					return ErrTransferItemLocked
				}
			} else {
				return ErrTransferItemLocked
			}
		}

		itMap["transferLockedAt"] = now
		itMap["transferLockExpiresAt"] = lockUntil
		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		return nil
	})
}

// UnlockTransferItem releases an item-level lock.
func (r *OrderRepoForTransferFS) UnlockTransferItem(ctx context.Context, orderID string, itemKey string) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}

	orderID = strings.TrimSpace(orderID)
	itemKey = strings.TrimSpace(itemKey)

	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemKey == "" {
		return ErrInvalidTransferItemKey
	}

	ref := r.orderDoc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrOrderNotFound
			}
			return err
		}
		if snap == nil || !snap.Exists() {
			return ErrOrderNotFound
		}

		raw := snap.Data()
		if raw == nil {
			return ErrOrderNotFound
		}

		items, err := rawOrderItemMaps(raw["items"])
		if err != nil {
			return err
		}

		idx, itMap, err := findItemMapByItemKey(items, itemKey)
		if err != nil {
			return err
		}

		delete(itMap, "transferLockedAt")
		delete(itMap, "transferLockExpiresAt")
		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		return nil
	})
}

// MarkTransferredItem marks an item as transferred and clears lock fields.
func (r *OrderRepoForTransferFS) MarkTransferredItem(ctx context.Context, orderID string, itemKey string, at time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}

	orderID = strings.TrimSpace(orderID)
	itemKey = strings.TrimSpace(itemKey)

	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemKey == "" {
		return ErrInvalidTransferItemKey
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()

	ref := r.orderDoc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrOrderNotFound
			}
			return err
		}
		if snap == nil || !snap.Exists() {
			return ErrOrderNotFound
		}

		raw := snap.Data()
		if raw == nil {
			return ErrOrderNotFound
		}

		if p, ok := raw["paid"].(bool); !ok || !p {
			return ErrOrderNotPaid
		}

		items, err := rawOrderItemMaps(raw["items"])
		if err != nil {
			return err
		}

		idx, itMap, err := findItemMapByItemKey(items, itemKey)
		if err != nil {
			return err
		}

		if b, ok := itMap["transferred"].(bool); ok && b {
			return ErrTransferItemTransferred
		}

		itMap["transferred"] = true
		itMap["transferredAt"] = at

		delete(itMap, "transferLockedAt")
		delete(itMap, "transferLockExpiresAt")

		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		return nil
	})
}

// ------------------------------------------------------------
// Helpers (OrderRepoForTransferFS)
// ------------------------------------------------------------

func parseOrderItems(v any) ([]orderdom.OrderItemSnapshot, error) {
	if v == nil {
		return []orderdom.OrderItemSnapshot{}, nil
	}

	arr, ok := v.([]any)
	if !ok {
		return nil, ErrOrderItemsMissing
	}

	out := make([]orderdom.OrderItemSnapshot, 0, len(arr))
	for _, x := range arr {
		m, ok := x.(map[string]any)
		if !ok || m == nil {
			continue
		}

		it := orderdom.OrderItemSnapshot{}

		if s, ok := m["type"].(string); ok {
			it.Type = orderdom.OrderItemType(strings.TrimSpace(s))
		}

		if s, ok := m["modelId"].(string); ok {
			it.ModelID = strings.TrimSpace(s)
		}

		if s, ok := m["inventoryId"].(string); ok {
			it.InventoryID = strings.TrimSpace(s)
		}

		if s, ok := m["listId"].(string); ok {
			it.ListID = strings.TrimSpace(s)
		}

		if s, ok := m["resaleId"].(string); ok {
			it.ResaleID = strings.TrimSpace(s)
		}

		if s, ok := m["productId"].(string); ok {
			it.ProductID = strings.TrimSpace(s)
		}

		if s, ok := m["productBlueprintId"].(string); ok {
			it.ProductBlueprintID = strings.TrimSpace(s)
		}

		if s, ok := m["tokenBlueprintId"].(string); ok {
			it.TokenBlueprintID = strings.TrimSpace(s)
		}

		if s, ok := m["brandId"].(string); ok {
			it.BrandID = strings.TrimSpace(s)
		}

		if it.Type == "" {
			it.Type = inferOrderItemTypeFromSnapshot(it)
		}

		it.Qty = intFromAny(m["qty"])
		it.Price = intFromAny(m["price"])

		if b, ok := m["isCanceled"].(bool); ok {
			it.IsCanceled = b
		}

		if b, ok := m["isDispatched"].(bool); ok {
			it.IsDispatched = b
		}

		if b, ok := m["transferred"].(bool); ok {
			it.Transferred = b
		}

		if t, ok := m["transferredAt"].(time.Time); ok && !t.IsZero() {
			tt := t.UTC()
			it.TransferredAt = &tt
		}

		out = append(out, it)
	}

	return out, nil
}

func rawOrderItemMaps(v any) ([]any, error) {
	if v == nil {
		return nil, ErrOrderItemsMissing
	}

	items, ok := v.([]any)
	if !ok {
		return nil, ErrOrderItemsMissing
	}

	return items, nil
}

func findItemMapByItemKey(items []any, itemKey string) (int, map[string]any, error) {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" {
		return -1, nil, ErrInvalidTransferItemKey
	}

	for i, v := range items {
		m, ok := v.(map[string]any)
		if !ok || m == nil {
			continue
		}

		if itemMapMatchesItemKey(m, itemKey) {
			return i, m, nil
		}
	}

	return -1, nil, ErrTransferItemNotFound
}

func itemMapMatchesItemKey(m map[string]any, itemKey string) bool {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" || m == nil {
		return false
	}

	itemType, rawID, ok := parseTransferItemKey(itemKey)
	if !ok {
		return false
	}

	switch itemType {
	case orderdom.OrderItemTypeResale:
		resaleID := stringFromAny(m["resaleId"])
		return resaleID != "" && resaleID == rawID

	case orderdom.OrderItemTypeList:
		modelID := stringFromAny(m["modelId"])
		return modelID != "" && modelID == rawID

	default:
		return false
	}
}

func parseTransferItemKey(itemKey string) (orderdom.OrderItemType, string, bool) {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" {
		return "", "", false
	}

	parts := strings.SplitN(itemKey, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	itemType := orderdom.OrderItemType(strings.TrimSpace(parts[0]))
	id := strings.TrimSpace(parts[1])

	if id == "" {
		return "", "", false
	}

	switch itemType {
	case orderdom.OrderItemTypeList, orderdom.OrderItemTypeResale:
		return itemType, id, true
	default:
		return "", "", false
	}
}

func inferOrderItemTypeFromSnapshot(it orderdom.OrderItemSnapshot) orderdom.OrderItemType {
	if strings.TrimSpace(it.ResaleID) != "" || strings.TrimSpace(it.ProductID) != "" {
		return orderdom.OrderItemTypeResale
	}

	if strings.TrimSpace(it.ModelID) != "" ||
		strings.TrimSpace(it.InventoryID) != "" ||
		strings.TrimSpace(it.ListID) != "" {
		return orderdom.OrderItemTypeList
	}

	return ""
}

func stringFromAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// ============================================================
// TransferRepo (Firestore)
// - implements usecase.TransferRepo
//
// This usecase requires transfer table creation/update.
// Transfers are persisted here.
// ============================================================

var (
	ErrTransferRepoNotConfigured = errors.New("transfer_repo_fs: not configured")
	ErrInvalidTransferProductID  = errors.New("transfer_repo_fs: productId is empty")
	ErrInvalidTransferAttempt    = errors.New("transfer_repo_fs: attempt is invalid")
)

type TransferRepositoryFS struct {
	Client *firestore.Client

	// TransfersCollection defaults to "transfers"
	TransfersCollection string

	// AttemptCountersCollection defaults to "transferAttemptCounters"
	AttemptCountersCollection string
}

var _ usecase.TransferRepo = (*TransferRepositoryFS)(nil)
var _ transferdom.TransferQueryPort = (*TransferRepositoryFS)(nil)

// NewTransferRepositoryFS is referenced from DI as outfs.NewTransferRepositoryFS(...).
func NewTransferRepositoryFS(client *firestore.Client) *TransferRepositoryFS {
	return &TransferRepositoryFS{
		Client:                    client,
		TransfersCollection:       "",
		AttemptCountersCollection: "",
	}
}

func (r *TransferRepositoryFS) transfersCol() *firestore.CollectionRef {
	col := r.TransfersCollection
	if col == "" {
		col = os.Getenv("TRANSFERS_COLLECTION")
	}
	if col == "" {
		col = "transfers"
	}
	return r.Client.Collection(col)
}

func (r *TransferRepositoryFS) countersCol() *firestore.CollectionRef {
	col := r.AttemptCountersCollection
	if col == "" {
		col = os.Getenv("TRANSFER_ATTEMPT_COUNTERS_COLLECTION")
	}
	if col == "" {
		col = "transferAttemptCounters"
	}
	return r.Client.Collection(col)
}

// transferDocID returns flat doc id: "<productId>__<attempt>".
func (r *TransferRepositoryFS) transferDocID(productID string, attempt int) string {
	return productID + "__" + strconv.Itoa(attempt)
}

func (r *TransferRepositoryFS) counterDoc(productID string) *firestore.DocumentRef {
	return r.countersCol().Doc(productID)
}

// NextAttempt returns the next monotonically increasing attempt number for a productId.
// Implementation: transferAttemptCounters/{productId}.nextAttempt is incremented in a transaction.
func (r *TransferRepositoryFS) NextAttempt(ctx context.Context, productID string) (int, error) {
	if r == nil || r.Client == nil {
		return 0, ErrTransferRepoNotConfigured
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return 0, ErrInvalidTransferProductID
	}

	ref := r.counterDoc(productID)

	var out int

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				out = 1
				now := time.Now().UTC()
				return tx.Set(ref, map[string]any{
					"productId":     productID,
					"nextAttempt":   int64(2),
					"updatedAt":     now,
					"initializedAt": now,
				}, firestore.MergeAll)
			}
			return err
		}

		raw := snap.Data()
		var next int64 = 1
		if raw != nil {
			switch v := raw["nextAttempt"].(type) {
			case int64:
				next = v
			case int:
				next = int64(v)
			case float64:
				next = int64(v)
			}
		}
		if next <= 0 {
			next = 1
		}

		out = int(next)

		return tx.Set(ref, map[string]any{
			"productId":   productID,
			"nextAttempt": next + 1,
			"updatedAt":   time.Now().UTC(),
		}, firestore.MergeAll)
	})

	if err != nil {
		return 0, err
	}

	return out, nil
}

// Create creates a new transfer attempt record.
//
// Save keys are normalized to lowerCamelCase.
// This prevents key-name drift between Create and Update.
//
// Share transfer support:
//   - if orderId is "share:<fromAvatarId>:<toAvatarId>:<productId>",
//     transferKind/shareRef/fromAvatarId/toAvatarId are also saved.
//   - this keeps transferdom.Transfer unchanged while making share transfers distinguishable in Firestore.
func (r *TransferRepositoryFS) Create(ctx context.Context, t transferdom.Transfer) error {
	if r == nil || r.Client == nil {
		return ErrTransferRepoNotConfigured
	}

	t.ProductID = strings.TrimSpace(t.ProductID)
	if t.ProductID == "" {
		return ErrInvalidTransferProductID
	}
	if t.Attempt <= 0 {
		return ErrInvalidTransferAttempt
	}

	now := time.Now().UTC()
	createdAt := now
	if !t.CreatedAt.IsZero() {
		createdAt = t.CreatedAt.UTC()
	}
	updatedAt := createdAt

	docID := r.transferDocID(t.ProductID, t.Attempt)

	doc := map[string]any{
		"attempt":         int64(t.Attempt),
		"avatarId":        t.AvatarID,
		"createdAt":       createdAt,
		"errorMsg":        t.ErrorMsg,
		"errorType":       t.ErrorType,
		"mintAddress":     t.MintAddress,
		"orderId":         t.OrderID,
		"productId":       t.ProductID,
		"status":          t.Status,
		"toWalletAddress": t.ToWalletAddress,
		"txSignature":     t.TxSignature,
		"updatedAt":       updatedAt,
	}

	if share, ok := parseShareTransferRef(t.OrderID, t.ProductID); ok {
		doc["transferKind"] = "share"
		doc["shareRef"] = t.OrderID
		doc["fromAvatarId"] = share.FromAvatarID
		doc["toAvatarId"] = share.ToAvatarID

		// Existing Create stores avatarId as receiver, so share transfer also stores receiver explicitly.
		doc["receiverAvatarId"] = t.AvatarID
	} else if t.OrderID != "" {
		doc["transferKind"] = "order"
	}

	_, err := r.transfersCol().Doc(docID).Create(ctx, doc)
	return err
}

// Update updates an existing transfer attempt record by (productId, attempt).
// Update only merges patch-specified fields.
func (r *TransferRepositoryFS) Update(ctx context.Context, productID string, attempt int, p transferdom.TransferPatch) error {
	if r == nil || r.Client == nil {
		return ErrTransferRepoNotConfigured
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ErrInvalidTransferProductID
	}
	if attempt <= 0 {
		return ErrInvalidTransferAttempt
	}

	now := time.Now().UTC()

	update := map[string]any{
		"updatedAt": now,
	}

	if p.Status != nil {
		update["status"] = *p.Status
	}
	if p.TxSignature != nil {
		update["txSignature"] = *p.TxSignature
	}
	if p.ErrorType != nil {
		update["errorType"] = *p.ErrorType
	}
	if p.ErrorMsg != nil {
		update["errorMsg"] = *p.ErrorMsg
	}
	if p.MintAddress != nil {
		update["mintAddress"] = *p.MintAddress
	}

	docID := r.transferDocID(productID, attempt)
	_, err := r.transfersCol().Doc(docID).Set(ctx, update, firestore.MergeAll)
	return err
}

// ResolveTransferredAtByMintAddress resolves the transfer execution time by mintAddress.
//
// transfers collection では transferredAt を持たず、createdAt を transfer 実行日時として扱う。
// そのため戻り値の TransferredAt には createdAt を入れる。
func (r *TransferRepositoryFS) ResolveTransferredAtByMintAddress(
	ctx context.Context,
	mintAddress string,
) (transferdom.ResolveTransferredAtByMintAddressResult, error) {
	if r == nil || r.Client == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{}, ErrTransferRepoNotConfigured
	}

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return transferdom.ResolveTransferredAtByMintAddressResult{}, transferdom.ErrInvalidMintAddress
	}

	iter := r.transfersCol().
		Where("mintAddress", "==", m).
		Where("status", "==", string(transferdom.StatusSucceeded)).
		OrderBy("createdAt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return transferdom.ResolveTransferredAtByMintAddressResult{}, transferdom.ErrNotFound
		}
		return transferdom.ResolveTransferredAtByMintAddressResult{}, err
	}

	if doc == nil || doc.Ref == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{}, transferdom.ErrNotFound
	}

	raw := doc.Data()
	if raw == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{}, transferdom.ErrNotFound
	}

	createdAt := timeFromRaw(raw, "createdAt")
	if createdAt.IsZero() {
		return transferdom.ResolveTransferredAtByMintAddressResult{}, transferdom.ErrNotFound
	}

	productID := stringFromRaw(raw, "productId")
	avatarID := stringFromRaw(raw, "avatarId")
	attempt := intFromRaw(raw, "attempt")

	return transferdom.ResolveTransferredAtByMintAddressResult{
		ProductID:     productID,
		Attempt:       attempt,
		AvatarID:      avatarID,
		MintAddress:   m,
		TransferredAt: createdAt,
	}, nil
}

type shareTransferRef struct {
	FromAvatarID string
	ToAvatarID   string
	ProductID    string
}

func parseShareTransferRef(ref string, fallbackProductID string) (shareTransferRef, bool) {
	if ref == "" {
		return shareTransferRef{}, false
	}

	parts := strings.Split(ref, ":")
	if len(parts) != 4 {
		return shareTransferRef{}, false
	}
	if parts[0] != "share" {
		return shareTransferRef{}, false
	}
	if parts[1] == "" || parts[2] == "" {
		return shareTransferRef{}, false
	}

	productID := parts[3]
	if productID == "" {
		productID = fallbackProductID
	}
	if productID == "" {
		return shareTransferRef{}, false
	}

	return shareTransferRef{
		FromAvatarID: parts[1],
		ToAvatarID:   parts[2],
		ProductID:    productID,
	}, true
}

func stringFromRaw(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}

	v, ok := raw[key].(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(v)
}

func intFromRaw(raw map[string]any, key string) int {
	if raw == nil {
		return 0
	}

	return intFromAny(raw[key])
}

func timeFromRaw(raw map[string]any, key string) time.Time {
	if raw == nil {
		return time.Time{}
	}

	t, ok := raw[key].(time.Time)
	if !ok || t.IsZero() {
		return time.Time{}
	}

	return t.UTC()
}
