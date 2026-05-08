// backend/internal/adapters/out/firestore/transfer_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
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
// - Adjusted to match your actual Firestore orders schema.
// ============================================================

var (
	ErrOrderRepoNotConfigured  = errors.New("order_repo_for_transfer_fs: not configured")
	ErrInvalidOrderID          = errors.New("order_repo_for_transfer_fs: orderId is empty")
	ErrInvalidItemModelID      = errors.New("order_repo_for_transfer_fs: item modelId is empty")
	ErrInvalidTransferAvatarID = errors.New("order_repo_for_transfer_fs: avatarId is empty")
	ErrOrderNotFound           = errors.New("order_repo_for_transfer_fs: order not found")
	ErrOrderNotPaid            = errors.New("order_repo_for_transfer_fs: order is not paid")
	ErrOrderItemsMissing       = errors.New("order_repo_for_transfer_fs: order items missing/invalid")
	ErrTransferItemNotFound    = errors.New("order_repo_for_transfer_fs: transfer item not found")
	ErrTransferItemTransferred = errors.New("order_repo_for_transfer_fs: item already transferred")
	ErrTransferItemLocked      = errors.New("order_repo_for_transfer_fs: item is locked")
)

const defaultTransferLockTTL = 10 * time.Minute

type OrderRepoForTransferFS struct {
	Client *firestore.Client

	// OrdersCollection defaults to "orders"
	OrdersCollection string
}

var _ usecase.OrderRepoForTransfer = (*OrderRepoForTransferFS)(nil)

func NewOrderRepoForTransferFS(client *firestore.Client) *OrderRepoForTransferFS {
	return &OrderRepoForTransferFS{
		Client:           client,
		OrdersCollection: "",
	}
}

func (r *OrderRepoForTransferFS) ordersCol() *firestore.CollectionRef {
	col := r.OrdersCollection
	if col == "" {
		col = os.Getenv("ORDERS_COLLECTION")
	}
	if col == "" {
		col = "orders"
	}
	return r.Client.Collection(col)
}

func (r *OrderRepoForTransferFS) orderDoc(orderID string) *firestore.DocumentRef {
	return r.ordersCol().Doc(orderID)
}

// ------------------------------------------------------------
// usecase.OrderRepoForTransfer
// ------------------------------------------------------------

// ListPaidByAvatarID returns "paid" orders for avatar.
//
// Your actual schema does NOT have `paid`.
// Shortest practical approach:
//   - query by avatarId only
//   - infer Paid:
//   - if `paid` exists -> use it
//   - else if `billingSnapshot` exists and is non-empty -> treat as paid
func (r *OrderRepoForTransferFS) ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return nil, ErrOrderRepoNotConfigured
	}
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

		o := orderdom.Order{}
		o.ID = doc.Ref.ID

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

		if p, ok := raw["paid"].(bool); ok {
			o.Paid = p
		} else {
			o.Paid = inferPaidFromOrder(raw)
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
// Your current item schema lacks `transferred`, so we treat missing as false.
func (r *OrderRepoForTransferFS) LockTransferItem(ctx context.Context, orderID string, itemModelID string, now time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemModelID == "" {
		return ErrInvalidItemModelID
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

		if p, ok := raw["paid"].(bool); ok && !p {
			return ErrOrderNotPaid
		}

		itemsAny, ok := raw["items"]
		if !ok {
			return ErrOrderItemsMissing
		}
		items, ok := itemsAny.([]any)
		if !ok {
			return ErrOrderItemsMissing
		}

		idx, itMap, err := findItemMapByModelID(items, itemModelID)
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

		log.Printf(
			"[order_repo_for_transfer_fs] lock acquired orderId=%s modelId=%s lockedAt=%s expiresAt=%s",
			orderID, itemModelID, now.Format(time.RFC3339), lockUntil.Format(time.RFC3339),
		)
		return nil
	})
}

// UnlockTransferItem releases an item-level lock (best-effort).
func (r *OrderRepoForTransferFS) UnlockTransferItem(ctx context.Context, orderID string, itemModelID string) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemModelID == "" {
		return ErrInvalidItemModelID
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

		itemsAny, ok := raw["items"]
		if !ok {
			return ErrOrderItemsMissing
		}
		items, ok := itemsAny.([]any)
		if !ok {
			return ErrOrderItemsMissing
		}

		idx, itMap, err := findItemMapByModelID(items, itemModelID)
		if err != nil {
			return err
		}

		delete(itMap, "transferLockedAt")
		delete(itMap, "transferLockExpiresAt")
		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		log.Printf("[order_repo_for_transfer_fs] lock released orderId=%s modelId=%s", orderID, itemModelID)
		return nil
	})
}

// MarkTransferredItem marks an item as transferred and clears lock fields.
func (r *OrderRepoForTransferFS) MarkTransferredItem(ctx context.Context, orderID string, itemModelID string, at time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}
	if orderID == "" {
		return ErrInvalidOrderID
	}
	if itemModelID == "" {
		return ErrInvalidItemModelID
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

		if p, ok := raw["paid"].(bool); ok && !p {
			return ErrOrderNotPaid
		}

		itemsAny, ok := raw["items"]
		if !ok {
			return ErrOrderItemsMissing
		}
		items, ok := itemsAny.([]any)
		if !ok {
			return ErrOrderItemsMissing
		}

		idx, itMap, err := findItemMapByModelID(items, itemModelID)
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

		log.Printf(
			"[order_repo_for_transfer_fs] marked transferred orderId=%s modelId=%s transferredAt=%s",
			orderID, itemModelID, at.Format(time.RFC3339),
		)
		return nil
	})
}

// ------------------------------------------------------------
// Helpers (OrderRepoForTransferFS)
// ------------------------------------------------------------

func inferPaidFromOrder(raw map[string]any) bool {
	if bs, ok := raw["billingSnapshot"].(map[string]any); ok && bs != nil && len(bs) > 0 {
		return true
	}
	return false
}

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

		if s, ok := m["modelId"].(string); ok {
			it.ModelID = s
		} else if s, ok := m["modelID"].(string); ok {
			it.ModelID = s
		}

		if s, ok := m["inventoryId"].(string); ok {
			it.InventoryID = s
		} else if s, ok := m["inventoryID"].(string); ok {
			it.InventoryID = s
		}

		switch n := m["qty"].(type) {
		case int:
			it.Qty = n
		case int64:
			it.Qty = int(n)
		case float64:
			it.Qty = int(n)
		}

		switch n := m["price"].(type) {
		case int:
			it.Price = n
		case int64:
			it.Price = int(n)
		case float64:
			it.Price = int(n)
		}

		if b, ok := m["transferred"].(bool); ok {
			it.Transferred = b
		} else {
			it.Transferred = false
		}

		if t, ok := m["transferredAt"].(time.Time); ok && !t.IsZero() {
			tt := t.UTC()
			it.TransferredAt = &tt
		}

		out = append(out, it)
	}
	return out, nil
}

func findItemMapByModelID(items []any, modelID string) (int, map[string]any, error) {
	if modelID == "" {
		return -1, nil, ErrInvalidItemModelID
	}

	for i, v := range items {
		m, ok := v.(map[string]any)
		if !ok || m == nil {
			continue
		}

		var got string
		if s, ok := m["modelId"].(string); ok {
			got = s
		} else if s, ok := m["modelID"].(string); ok {
			got = s
		}

		if got == modelID {
			return i, m, nil
		}
	}

	return -1, nil, ErrTransferItemNotFound
}

// ============================================================
// TransferRepo (Firestore)
// - implements usecase.TransferRepo
//
// このUsecaseは「transfer テーブルの起票・更新」が必須。
// ここで transfers を永続化する。
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
// 実装: transferAttemptCounters/{productId}.nextAttempt をトランザクションで採番する。
func (r *TransferRepositoryFS) NextAttempt(ctx context.Context, productID string) (int, error) {
	if r == nil || r.Client == nil {
		return 0, ErrTransferRepoNotConfigured
	}
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

// Create creates a new transfer attempt record (typically pending).
//
// 正規保存キーは lowerCamelCase に統一する。
// これにより Create と Update のキー名揺れを防ぐ。
//
// 共有 transfer 対応:
//   - orderId が "share:<fromAvatarId>:<toAvatarId>:<productId>" の形式なら
//     transferKind/shareRef/fromAvatarId/toAvatarId を追加保存する。
//   - これにより既存の transferdom.Transfer を壊さず、share_transfer_usecase の結果も
//     Firestore 上で区別できるようにする。
func (r *TransferRepositoryFS) Create(ctx context.Context, t transferdom.Transfer) error {
	if r == nil || r.Client == nil {
		return ErrTransferRepoNotConfigured
	}
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

		// 既存 create の avatarId は receiver として使われているので、
		// share では意味が分かるように receiver 側も明示保存しておく。
		doc["receiverAvatarId"] = t.AvatarID
	} else if t.OrderID != "" {
		doc["transferKind"] = "order"
	}

	_, err := r.transfersCol().Doc(docID).Create(ctx, doc)
	return err
}

// Update updates an existing transfer attempt record by (productId, attempt).
// Update は patch 指定フィールドのみ merge 更新する。
func (r *TransferRepositoryFS) Update(ctx context.Context, productID string, attempt int, p transferdom.TransferPatch) error {
	if r == nil || r.Client == nil {
		return ErrTransferRepoNotConfigured
	}
	if productID == "" {
		return ErrInvalidTransferProductID
	}
	if attempt <= 0 {
		return ErrInvalidTransferAttempt
	}

	update := map[string]any{
		"updatedAt": time.Now().UTC(),
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
