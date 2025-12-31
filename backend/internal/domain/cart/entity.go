// backend/internal/domain/cart/entity.go
package cart

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidCart        = errors.New("cart: invalid")
	ErrCartAlreadyOrdered = errors.New("cart: already ordered")
)

// DefaultCartTTL is the inactivity window after which the cart becomes eligible for auto deletion
// (Firestore TTL should be configured on expiresAt).
// You can change this later to fit product policy.
const DefaultCartTTL = 7 * 24 * time.Hour

// Cart represents "avatar's cart".
// - AvatarID: who owns this cart
// - Items: modelId -> quantity
// - Ordered: true once converted to an order (locked)
// - ExpiresAt: for Firestore TTL (auto deletion), updated on each cart mutation
type Cart struct {
	AvatarID string

	// Items is modelId -> qty
	Items map[string]int

	CreatedAt time.Time
	UpdatedAt time.Time

	// ExpiresAt is used for Firestore TTL.
	// This should be set to a future timestamp and refreshed on each update.
	ExpiresAt time.Time

	Ordered bool
}

// NewCart creates a new cart for an avatar.
// items can be nil (treated as empty). ordered is always false at creation.
func NewCart(avatarID string, items map[string]int, now time.Time) (*Cart, error) {
	c := &Cart{
		AvatarID:  strings.TrimSpace(avatarID),
		Items:     cloneItems(items),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(DefaultCartTTL),
		Ordered:   false,
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Add increases quantity for a modelId.
// qty must be >= 1.
func (c *Cart) Add(modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}
	if c.Ordered {
		return ErrCartAlreadyOrdered
	}
	mid := strings.TrimSpace(modelID)
	if mid == "" || qty <= 0 {
		return ErrInvalidCart
	}
	if c.Items == nil {
		c.Items = map[string]int{}
	}
	c.Items[mid] = c.Items[mid] + qty
	c.touch(now)
	return c.validate()
}

// SetQty sets quantity for a modelId.
// If qty <= 0, it removes the modelId from the cart.
func (c *Cart) SetQty(modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}
	if c.Ordered {
		return ErrCartAlreadyOrdered
	}
	mid := strings.TrimSpace(modelID)
	if mid == "" {
		return ErrInvalidCart
	}
	if c.Items == nil {
		c.Items = map[string]int{}
	}
	if qty <= 0 {
		delete(c.Items, mid)
	} else {
		c.Items[mid] = qty
	}
	c.touch(now)
	return c.validate()
}

// Remove removes a modelId from the cart.
func (c *Cart) Remove(modelID string, now time.Time) error {
	return c.SetQty(modelID, 0, now)
}

// MarkOrdered locks the cart.
// (Usually you will delete the cart after creating an order, but TTL is still refreshed here.)
func (c *Cart) MarkOrdered(now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}
	if c.Ordered {
		return ErrCartAlreadyOrdered
	}
	c.Ordered = true
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
	if strings.TrimSpace(c.AvatarID) == "" {
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
			mid := strings.TrimSpace(k)
			qty := c.Items[k]
			if mid == "" || qty <= 0 {
				return ErrInvalidCart
			}
			// normalize key if it had spaces
			if mid != k {
				delete(c.Items, k)
				// merge if normalized key already exists
				c.Items[mid] = c.Items[mid] + qty
			}
		}
	}

	return nil
}

func cloneItems(src map[string]int) map[string]int {
	if src == nil {
		return map[string]int{}
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		k2 := strings.TrimSpace(k)
		if k2 == "" || v <= 0 {
			continue
		}
		dst[k2] = dst[k2] + v
	}
	return dst
}
