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
//   - Items: []CartItem (no map key / no composite id)
//   - ExpiresAt: for Firestore TTL (auto deletion), updated on each cart mutation
//
// NOTE:
// - ordered フラグは持たない
// - order テーブル作成（注文確定）に合わせて、items を空にする（消費）
type Cart struct {
	// ID is Firestore docId (= avatarId).
	ID string `json:"id" firestore:"id"`

	// Items is a list of line items.
	// Uniqueness is defined by (inventoryId, listId, modelId).
	Items []CartItem `json:"items" firestore:"items"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`

	// ExpiresAt is used for Firestore TTL.
	// This should be set to a future timestamp and refreshed on each update.
	ExpiresAt time.Time `json:"expiresAt" firestore:"expiresAt"`
}

// NewCart creates a new cart doc.
// id is the Firestore docId (avatarId).
// items can be nil (treated as empty).
func NewCart(id string, items []CartItem, now time.Time) (*Cart, error) {
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
		c.Items = []CartItem{}
	}

	idx := findItemIndex(c.Items, inv, lid, mid)
	if idx >= 0 {
		c.Items[idx].Qty += qty
		// normalize fields (keep consistent)
		c.Items[idx].InventoryID = inv
		c.Items[idx].ListID = lid
		c.Items[idx].ModelID = mid
	} else {
		c.Items = append(c.Items, CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		})
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
		c.Items = []CartItem{}
	}

	idx := findItemIndex(c.Items, inv, lid, mid)

	if qty <= 0 {
		// remove if exists
		if idx >= 0 {
			c.Items = removeIndex(c.Items, idx)
		}
		c.touch(now)
		return c.validate()
	}

	if idx >= 0 {
		c.Items[idx] = CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		}
	} else {
		c.Items = append(c.Items, CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		})
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
func (c *Cart) ConsumeAll(now time.Time) ([]CartItem, error) {
	if c == nil {
		return nil, ErrInvalidCart
	}

	// snapshot (already normalized)
	snap := cloneItems(c.Items)

	// clear
	c.Items = []CartItem{}

	c.touch(now)
	if err := c.validate(); err != nil {
		return nil, err
	}
	return snap, nil
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
	if len(c.Items) == 0 {
		return nil
	}

	// normalize + merge duplicates + stable order
	c.Items = normalizeAndMerge(c.Items)

	// final validation
	for _, it := range c.Items {
		inv := strings.TrimSpace(it.InventoryID)
		lid := strings.TrimSpace(it.ListID)
		mid := strings.TrimSpace(it.ModelID)
		if inv == "" || lid == "" || mid == "" || it.Qty <= 0 {
			return ErrInvalidCart
		}
	}

	return nil
}

// ----------------------------
// Helpers
// ----------------------------

func findItemIndex(items []CartItem, inv, lid, mid string) int {
	for i := range items {
		if items[i].InventoryID == inv && items[i].ListID == lid && items[i].ModelID == mid {
			return i
		}
	}
	return -1
}

func removeIndex(items []CartItem, idx int) []CartItem {
	if idx < 0 || idx >= len(items) {
		return items
	}
	// preserve order
	return append(items[:idx], items[idx+1:]...)
}

type itemKey struct {
	inv string
	lid string
	mid string
}

func normalizeAndMerge(src []CartItem) []CartItem {
	// collect + normalize
	m := map[itemKey]CartItem{}

	for _, it := range src {
		inv := strings.TrimSpace(it.InventoryID)
		lid := strings.TrimSpace(it.ListID)
		mid := strings.TrimSpace(it.ModelID)
		qty := it.Qty

		if inv == "" || lid == "" || mid == "" || qty <= 0 {
			continue
		}

		k := itemKey{inv: inv, lid: lid, mid: mid}

		if exist, ok := m[k]; ok {
			exist.Qty += qty
			exist.InventoryID = inv
			exist.ListID = lid
			exist.ModelID = mid
			m[k] = exist
		} else {
			m[k] = CartItem{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			}
		}
	}

	// stable order
	keys := make([]itemKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].inv != keys[j].inv {
			return keys[i].inv < keys[j].inv
		}
		if keys[i].lid != keys[j].lid {
			return keys[i].lid < keys[j].lid
		}
		return keys[i].mid < keys[j].mid
	})

	out := make([]CartItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func cloneItems(src []CartItem) []CartItem {
	if len(src) == 0 {
		return []CartItem{}
	}
	// copy then normalize/merge to keep invariants
	cp := make([]CartItem, 0, len(src))
	for _, it := range src {
		cp = append(cp, it)
	}
	return normalizeAndMerge(cp)
}
