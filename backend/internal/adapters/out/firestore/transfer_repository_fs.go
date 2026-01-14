// backend\internal\adapters\out\firestore\transfer_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
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
	ErrInvalidTransferAvatarID = errors.New("order_repo_for_transfer_fs: avatarId is empty") // ✅ avoid DuplicateDecl
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
	col := strings.TrimSpace(r.OrdersCollection)
	if col == "" {
		col = strings.TrimSpace(os.Getenv("ORDERS_COLLECTION"))
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
// ✅ Your actual schema does NOT have `paid`.
// ✅ Shortest practical approach:
//   - query by avatarId only
//   - infer Paid:
//   - if `paid` exists -> use it
//   - else if `billingSnapshot` exists and is non-empty -> treat as paid
func (r *OrderRepoForTransferFS) ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return nil, ErrOrderRepoNotConfigured
	}
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return nil, ErrInvalidTransferAvatarID
	}

	it := r.ordersCol().
		Where("avatarId", "==", aid).
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
		o.ID = strings.TrimSpace(doc.Ref.ID)

		// userId
		if v, ok := raw["userId"].(string); ok {
			o.UserID = strings.TrimSpace(v)
		}

		// avatarId
		if v, ok := raw["avatarId"].(string); ok {
			o.AvatarID = strings.TrimSpace(v)
		}

		// cartId
		if v, ok := raw["cartId"].(string); ok {
			o.CartID = strings.TrimSpace(v)
		}

		// createdAt
		if t, ok := raw["createdAt"].(time.Time); ok && !t.IsZero() {
			o.CreatedAt = t.UTC()
		}

		// paid (explicit or inferred)
		if p, ok := raw["paid"].(bool); ok {
			o.Paid = p
		} else {
			o.Paid = inferPaidFromOrder(raw)
		}

		// items
		items, err := parseOrderItems(raw["items"])
		if err != nil {
			return nil, err
		}
		o.Items = items

		out = append(out, o)
	}

	// NOTE:
	// Usecase側でも o.Paid をチェックしているが、
	// ここで Paid を推定しておくと探索が成功する。
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
	oid := strings.TrimSpace(orderID)
	mid := strings.TrimSpace(itemModelID)
	if oid == "" {
		return ErrInvalidOrderID
	}
	if mid == "" {
		return ErrInvalidItemModelID
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	ref := r.orderDoc(oid)
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

		// paid check: if field exists and false -> not paid
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

		idx, itMap, err := findItemMapByModelID(items, mid)
		if err != nil {
			return err
		}

		// already transferred? (missing => false)
		if b, ok := itMap["transferred"].(bool); ok && b {
			return ErrTransferItemTransferred
		}

		// lock check
		if lockedAt, ok := itMap["transferLockedAt"].(time.Time); ok && !lockedAt.IsZero() {
			if exp, ok := itMap["transferLockExpiresAt"].(time.Time); ok && !exp.IsZero() {
				if exp.After(now) {
					return ErrTransferItemLocked
				}
			} else {
				return ErrTransferItemLocked
			}
		}

		// set lock
		itMap["transferLockedAt"] = now
		itMap["transferLockExpiresAt"] = lockUntil
		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		log.Printf(
			"[order_repo_for_transfer_fs] lock acquired orderId=%s modelId=%s lockedAt=%s expiresAt=%s",
			maskShort(oid), maskShort(mid), now.Format(time.RFC3339), lockUntil.Format(time.RFC3339),
		)
		return nil
	})
}

// UnlockTransferItem releases an item-level lock (best-effort).
func (r *OrderRepoForTransferFS) UnlockTransferItem(ctx context.Context, orderID string, itemModelID string) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}
	oid := strings.TrimSpace(orderID)
	mid := strings.TrimSpace(itemModelID)
	if oid == "" {
		return ErrInvalidOrderID
	}
	if mid == "" {
		return ErrInvalidItemModelID
	}

	ref := r.orderDoc(oid)

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

		idx, itMap, err := findItemMapByModelID(items, mid)
		if err != nil {
			return err
		}

		delete(itMap, "transferLockedAt")
		delete(itMap, "transferLockExpiresAt")
		items[idx] = itMap

		if err := tx.Set(ref, map[string]any{"items": items}, firestore.MergeAll); err != nil {
			return err
		}

		log.Printf("[order_repo_for_transfer_fs] lock released orderId=%s modelId=%s", maskShort(oid), maskShort(mid))
		return nil
	})
}

// MarkTransferredItem marks an item as transferred and clears lock fields.
// Your current item schema lacks transferred/transferredAt -> we add them.
func (r *OrderRepoForTransferFS) MarkTransferredItem(ctx context.Context, orderID string, itemModelID string, at time.Time) error {
	if r == nil || r.Client == nil {
		return ErrOrderRepoNotConfigured
	}
	oid := strings.TrimSpace(orderID)
	mid := strings.TrimSpace(itemModelID)
	if oid == "" {
		return ErrInvalidOrderID
	}
	if mid == "" {
		return ErrInvalidItemModelID
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()

	ref := r.orderDoc(oid)

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

		// paid check: if field exists and false -> not paid
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

		idx, itMap, err := findItemMapByModelID(items, mid)
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
			maskShort(oid), maskShort(mid), at.Format(time.RFC3339),
		)
		return nil
	})
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func inferPaidFromOrder(raw map[string]any) bool {
	// ✅ your order has billingSnapshot when paid (based on your shown doc)
	if bs, ok := raw["billingSnapshot"].(map[string]any); ok && bs != nil && len(bs) > 0 {
		return true
	}
	// if you later add explicit status/payment fields, extend here
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

		// modelId
		if s, ok := m["modelId"].(string); ok {
			it.ModelID = strings.TrimSpace(s)
		} else if s, ok := m["modelID"].(string); ok {
			it.ModelID = strings.TrimSpace(s)
		}

		// inventoryId
		if s, ok := m["inventoryId"].(string); ok {
			it.InventoryID = strings.TrimSpace(s)
		} else if s, ok := m["inventoryID"].(string); ok {
			it.InventoryID = strings.TrimSpace(s)
		}

		// qty
		switch n := m["qty"].(type) {
		case int:
			it.Qty = n
		case int64:
			it.Qty = int(n)
		case float64:
			it.Qty = int(n)
		}

		// price
		switch n := m["price"].(type) {
		case int:
			it.Price = n
		case int64:
			it.Price = int(n)
		case float64:
			it.Price = int(n)
		}

		// transferred (missing => false)
		if b, ok := m["transferred"].(bool); ok {
			it.Transferred = b
		} else {
			it.Transferred = false
		}

		// transferredAt (optional)
		if t, ok := m["transferredAt"].(time.Time); ok && !t.IsZero() {
			tt := t.UTC()
			it.TransferredAt = &tt
		}

		out = append(out, it)
	}
	return out, nil
}

func findItemMapByModelID(items []any, modelID string) (int, map[string]any, error) {
	mid := strings.TrimSpace(modelID)
	if mid == "" {
		return -1, nil, ErrInvalidItemModelID
	}

	for i, v := range items {
		m, ok := v.(map[string]any)
		if !ok || m == nil {
			continue
		}
		var got string
		if s, ok := m["modelId"].(string); ok {
			got = strings.TrimSpace(s)
		} else if s, ok := m["modelID"].(string); ok {
			got = strings.TrimSpace(s)
		}
		if got == mid {
			return i, m, nil
		}
	}
	return -1, nil, ErrTransferItemNotFound
}

func maskShort(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
