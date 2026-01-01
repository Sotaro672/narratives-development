// backend/internal/domain/cart/entity.go
package cart

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidCart = errors.New("cart: invalid")
)

// DefaultCartTTL is the inactivity window after which the cart becomes eligible for auto deletion
// (Firestore TTL should be configured on expiresAt).
const DefaultCartTTL = 7 * 24 * time.Hour

// CartItem represents "one line item" in a cart.
// We keep inventoryId + listId + modelId to uniquely identify the selected product context.
type CartItem struct {
	InventoryID string `json:"inventoryId" firestore:"inventoryId"`
	ListID      string `json:"listId" firestore:"listId"`
	ModelID     string `json:"modelId" firestore:"modelId"`
	Qty         int    `json:"qty" firestore:"qty"`
}

// Cart represents "a cart document".
//   - docId = avatarId (Firestore)
//   - Items: itemKey -> CartItem
//     itemKey is a deterministic composite key from (inventoryId, listId, modelId)
//   - ExpiresAt: for Firestore TTL (auto deletion), updated on each cart mutation
//
// NOTE:
// - ordered フラグは持たない
// - order テーブル作成（注文確定）に合わせて、items から itemKey を削除（消費）する
type Cart struct {
	// ID is Firestore docId (= avatarId).
	ID string `json:"id" firestore:"id"`

	// Items is itemKey -> CartItem (includes qty)
	Items map[string]CartItem `json:"items" firestore:"items"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`

	// ExpiresAt is used for Firestore TTL.
	// This should be set to a future timestamp and refreshed on each update.
	ExpiresAt time.Time `json:"expiresAt" firestore:"expiresAt"`
}

// NewCart creates a new cart doc.
// id is the Firestore docId (avatarId).
// items can be nil (treated as empty).
func NewCart(id string, items map[string]CartItem, now time.Time) (*Cart, error) {
	docID := strings.TrimSpace(id)

	c := &Cart{
		ID:        docID,
		Items:     cloneItems(items),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(DefaultCartTTL),
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Add increases quantity for a (inventoryId, listId, modelId).
// qty must be >= 1.
func (c *Cart) Add(inventoryID, listID, modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	inv := strings.TrimSpace(inventoryID)
	lid := strings.TrimSpace(listID)
	mid := strings.TrimSpace(modelID)
	if inv == "" || lid == "" || mid == "" || qty <= 0 {
		return ErrInvalidCart
	}

	if c.Items == nil {
		c.Items = map[string]CartItem{}
	}

	key := makeItemKey(inv, lid, mid)

	if it, ok := c.Items[key]; ok {
		it.Qty = it.Qty + qty
		// normalize fields (keep consistent)
		it.InventoryID = inv
		it.ListID = lid
		it.ModelID = mid
		c.Items[key] = it
	} else {
		c.Items[key] = CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		}
	}

	c.touch(now)
	return c.validate()
}

// SetQty sets quantity for a (inventoryId, listId, modelId).
// If qty <= 0, it removes the item from the cart.
func (c *Cart) SetQty(inventoryID, listID, modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	inv := strings.TrimSpace(inventoryID)
	lid := strings.TrimSpace(listID)
	mid := strings.TrimSpace(modelID)
	if inv == "" || lid == "" || mid == "" {
		return ErrInvalidCart
	}

	if c.Items == nil {
		c.Items = map[string]CartItem{}
	}

	key := makeItemKey(inv, lid, mid)

	if qty <= 0 {
		delete(c.Items, key)
	} else {
		c.Items[key] = CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		}
	}

	c.touch(now)
	return c.validate()
}

// Remove removes a (inventoryId, listId, modelId) from the cart.
func (c *Cart) Remove(inventoryID, listID, modelID string, now time.Time) error {
	return c.SetQty(inventoryID, listID, modelID, 0, now)
}

// ConsumeAll clears items for order creation and returns a snapshot of items.
//
// 想定ユースケース:
// 1) cart.Items を元に order を作成（order テーブルに保存）
// 2) 同トランザクション/同リクエスト内で cart.ConsumeAll() を呼び、items を空にする
func (c *Cart) ConsumeAll(now time.Time) (map[string]CartItem, error) {
	if c == nil {
		return nil, ErrInvalidCart
	}

	// snapshot
	snap := cloneItems(c.Items)

	// clear
	if c.Items == nil {
		c.Items = map[string]CartItem{}
	} else {
		for k := range c.Items {
			delete(c.Items, k)
		}
	}

	c.touch(now)
	if err := c.validate(); err != nil {
		return nil, err
	}
	return snap, nil
}

// Consume removes specific itemKeys from the cart (partial consumption).
// order テーブル側が「一部の items を注文確定」する設計の場合に使う。
func (c *Cart) Consume(itemKeys []string, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}
	if len(itemKeys) == 0 {
		return nil
	}
	if c.Items == nil {
		c.Items = map[string]CartItem{}
		return nil
	}

	for _, k := range itemKeys {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		delete(c.Items, key)
	}

	c.touch(now)
	return c.validate()
}

func (c *Cart) touch(now time.Time) {
	c.UpdatedAt = now
	c.ExpiresAt = now.Add(DefaultCartTTL)
}

func (c *Cart) validate() error {
	if c == nil {
		return ErrInvalidCart
	}

	// ✅ docId (= avatarId) must exist
	if strings.TrimSpace(c.ID) == "" {
		return ErrInvalidCart
	}

	if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() || c.ExpiresAt.IsZero() {
		return ErrInvalidCart
	}
	if c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidCart
	}
	// ExpiresAt should not be in the past relative to UpdatedAt (TTL refresh basis).
	if c.ExpiresAt.Before(c.UpdatedAt) {
		return ErrInvalidCart
	}

	// Items can be empty, but if present each entry must be valid.
	if c.Items != nil {
		// deterministic validation order (useful for tests/debug)
		keys := make([]string, 0, len(c.Items))
		for k := range c.Items {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			key := strings.TrimSpace(k)
			it := c.Items[k]

			inv := strings.TrimSpace(it.InventoryID)
			lid := strings.TrimSpace(it.ListID)
			mid := strings.TrimSpace(it.ModelID)

			if inv == "" || lid == "" || mid == "" || it.Qty <= 0 {
				return ErrInvalidCart
			}

			normalizedKey := makeItemKey(inv, lid, mid)

			// normalize key if it had spaces / wrong composition
			if normalizedKey != key {
				// remove old
				delete(c.Items, k)

				// merge if normalized key already exists
				if exist, ok := c.Items[normalizedKey]; ok {
					exist.Qty = exist.Qty + it.Qty
					exist.InventoryID = inv
					exist.ListID = lid
					exist.ModelID = mid
					c.Items[normalizedKey] = exist
				} else {
					it.InventoryID = inv
					it.ListID = lid
					it.ModelID = mid
					c.Items[normalizedKey] = it
				}
			} else {
				// normalize fields even when key is same
				if it.InventoryID != inv || it.ListID != lid || it.ModelID != mid {
					it.InventoryID = inv
					it.ListID = lid
					it.ModelID = mid
					c.Items[normalizedKey] = it
				}
			}
		}
	}

	return nil
}

func makeItemKey(inventoryID, listID, modelID string) string {
	// IDs are assumed not to contain this delimiter. If that assumption changes,
	// switch to encoding/escaping (e.g., base64 or url.PathEscape for each part).
	return strings.TrimSpace(inventoryID) + "__" + strings.TrimSpace(listID) + "__" + strings.TrimSpace(modelID)
}

func cloneItems(src map[string]CartItem) map[string]CartItem {
	dst := map[string]CartItem{}
	if src == nil {
		return dst
	}

	// stable copy with normalization
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		it := src[k]

		inv := strings.TrimSpace(it.InventoryID)
		lid := strings.TrimSpace(it.ListID)
		mid := strings.TrimSpace(it.ModelID)
		qty := it.Qty

		if inv == "" || lid == "" || mid == "" || qty <= 0 {
			continue
		}

		key := makeItemKey(inv, lid, mid)

		if exist, ok := dst[key]; ok {
			exist.Qty = exist.Qty + qty
			exist.InventoryID = inv
			exist.ListID = lid
			exist.ModelID = mid
			dst[key] = exist
		} else {
			dst[key] = CartItem{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			}
		}
	}

	return dst
}
