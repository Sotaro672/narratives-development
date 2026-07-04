// backend/internal/domain/cart/entity.go
package cart

import (
	"errors"
	"sort"
	"time"
)

var (
	ErrInvalidCart = errors.New("cart: invalid")
)

// DefaultCartTTL is the inactivity window after which the cart becomes eligible for auto deletion
// (Firestore TTL should be configured on expiresAt).
const DefaultCartTTL = 7 * 24 * time.Hour

type CartItemType string

const (
	CartItemTypeList   CartItemType = "list"
	CartItemTypeResale CartItemType = "resale"
)

// CartItem represents one line item in a cart.
type CartItem struct {
	Type CartItemType `json:"type" firestore:"type"`

	// Regular list item fields.
	InventoryID string `json:"inventoryId,omitempty" firestore:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty" firestore:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty" firestore:"modelId,omitempty"`

	// Resale item fields. Resale items do not link to inventory.
	ResaleID  string `json:"resaleId,omitempty" firestore:"resaleId,omitempty"`
	ProductID string `json:"productId,omitempty" firestore:"productId,omitempty"`

	Qty int `json:"qty" firestore:"qty"`
}

// Cart represents a cart document.
//
//   - docId = avatarId (Firestore)
//   - Items: itemKey -> CartItem
//   - ExpiresAt: for Firestore TTL (auto deletion), updated on each cart mutation
type Cart struct {
	// ID is Firestore docId (= avatarId).
	ID string `json:"id" firestore:"-"`

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
	c := &Cart{
		ID:        id,
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

// Add increases quantity for a regular list item (inventoryId, listId, modelId).
// qty must be >= 1.
func (c *Cart) Add(inventoryID, listID, modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	inv := inventoryID
	lid := listID
	mid := modelID
	if inv == "" || lid == "" || mid == "" || qty <= 0 {
		return ErrInvalidCart
	}

	if c.Items == nil {
		c.Items = map[string]CartItem{}
	}

	key := makeItemKey(inv, lid, mid)
	if it, ok := c.Items[key]; ok && cartItemTypeOrDefault(it.Type) == CartItemTypeList {
		qty = qty + it.Qty
	}

	c.Items[key] = CartItem{
		Type:        CartItemTypeList,
		InventoryID: inv,
		ListID:      lid,
		ModelID:     mid,
		Qty:         qty,
	}

	c.touch(now)
	return c.validate()
}

// AddResale adds or replaces a resale item in the cart.
// Resale items are one-off product transfers and always use qty=1.
func (c *Cart) AddResale(resaleID, productID string, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	rid := resaleID
	pid := productID
	if rid == "" || pid == "" {
		return ErrInvalidCart
	}

	if c.Items == nil {
		c.Items = map[string]CartItem{}
	}

	c.Items[makeResaleItemKey(rid, pid)] = CartItem{
		Type:      CartItemTypeResale,
		ResaleID:  rid,
		ProductID: pid,
		Qty:       1,
	}

	c.touch(now)
	return c.validate()
}

// SetQty sets quantity for a regular list item (inventoryId, listId, modelId).
// If qty <= 0, it removes the item from the cart.
func (c *Cart) SetQty(inventoryID, listID, modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	inv := inventoryID
	lid := listID
	mid := modelID
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
			Type:        CartItemTypeList,
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		}
	}

	c.touch(now)
	return c.validate()
}

// Remove removes a regular list item (inventoryId, listId, modelId) from the cart.
func (c *Cart) Remove(inventoryID, listID, modelID string, now time.Time) error {
	return c.SetQty(inventoryID, listID, modelID, 0, now)
}

// RemoveResale removes a resale item from the cart.
func (c *Cart) RemoveResale(resaleID, productID string, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}

	rid := resaleID
	pid := productID
	if rid == "" || pid == "" {
		return ErrInvalidCart
	}

	if c.Items == nil {
		c.Items = map[string]CartItem{}
	}

	delete(c.Items, makeResaleItemKey(rid, pid))
	c.touch(now)
	return c.validate()
}

// ConsumeAll clears items for order creation and returns a snapshot of items.
func (c *Cart) ConsumeAll(now time.Time) (map[string]CartItem, error) {
	if c == nil {
		return nil, ErrInvalidCart
	}

	snap := cloneItems(c.Items)

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
		key := k
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

	if c.ID == "" {
		return ErrInvalidCart
	}

	if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() || c.ExpiresAt.IsZero() {
		return ErrInvalidCart
	}
	if c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidCart
	}
	if c.ExpiresAt.Before(c.UpdatedAt) {
		return ErrInvalidCart
	}

	if len(c.Items) > 0 {
		keys := make([]string, 0, len(c.Items))
		for k := range c.Items {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			it, ok := c.Items[k]
			if !ok {
				continue
			}

			normalizedKey, normalized, err := normalizeItem(it)
			if err != nil {
				return err
			}

			if normalizedKey != k {
				delete(c.Items, k)
			}

			if exist, ok := c.Items[normalizedKey]; ok && normalizedKey != k {
				c.Items[normalizedKey] = mergeItems(exist, normalized)
			} else {
				c.Items[normalizedKey] = normalized
			}
		}
	}

	return nil
}

func normalizeItem(it CartItem) (string, CartItem, error) {
	itemType := cartItemTypeOrDefault(it.Type)

	switch itemType {
	case CartItemTypeList:
		inv := it.InventoryID
		lid := it.ListID
		mid := it.ModelID
		if inv == "" || lid == "" || mid == "" || it.Qty <= 0 {
			return "", CartItem{}, ErrInvalidCart
		}

		return makeItemKey(inv, lid, mid), CartItem{
			Type:        CartItemTypeList,
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         it.Qty,
		}, nil

	case CartItemTypeResale:
		rid := it.ResaleID
		pid := it.ProductID
		if rid == "" || pid == "" {
			return "", CartItem{}, ErrInvalidCart
		}

		return makeResaleItemKey(rid, pid), CartItem{
			Type:      CartItemTypeResale,
			ResaleID:  rid,
			ProductID: pid,
			Qty:       1,
		}, nil

	default:
		return "", CartItem{}, ErrInvalidCart
	}
}

func mergeItems(existing, incoming CartItem) CartItem {
	existingType := cartItemTypeOrDefault(existing.Type)
	incomingType := cartItemTypeOrDefault(incoming.Type)

	if existingType == CartItemTypeList && incomingType == CartItemTypeList {
		incoming.Qty = existing.Qty + incoming.Qty
	}

	return incoming
}

func cartItemTypeOrDefault(itemType CartItemType) CartItemType {
	if itemType == "" {
		return CartItemTypeList
	}
	return itemType
}

func makeItemKey(inventoryID, listID, modelID string) string {
	// IDs are assumed not to contain this delimiter. If that assumption changes,
	// switch to encoding/escaping (e.g., base64 or url.PathEscape for each part).
	return inventoryID + "__" + listID + "__" + modelID
}

func makeResaleItemKey(resaleID, productID string) string {
	return "resale__" + resaleID + "__" + productID
}

func cloneItems(src map[string]CartItem) map[string]CartItem {
	dst := map[string]CartItem{}
	if len(src) == 0 {
		return dst
	}

	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		normalizedKey, normalized, err := normalizeItem(src[k])
		if err != nil {
			continue
		}

		if exist, ok := dst[normalizedKey]; ok {
			dst[normalizedKey] = mergeItems(exist, normalized)
		} else {
			dst[normalizedKey] = normalized
		}
	}

	return dst
}
