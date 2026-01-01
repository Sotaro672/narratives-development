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
// You can change this later to fit product policy.
const DefaultCartTTL = 7 * 24 * time.Hour

// Cart represents "a cart document".
// - docId = avatarId (Firestore)
// - Items: modelId -> quantity
// - ExpiresAt: for Firestore TTL (auto deletion), updated on each cart mutation
//
// NOTE:
// - ordered フラグは持たない
// - order テーブル作成（注文確定）に合わせて、items から modelId を削除（消費）する
type Cart struct {
	// ID is Firestore docId (= avatarId).
	// （docId と同値をフィールドにも保持しておくことで、repo 側で docId 要求があっても整合できる）
	ID string `json:"id" firestore:"id"`

	// Items is modelId -> qty
	Items map[string]int `json:"items" firestore:"items"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`

	// ExpiresAt is used for Firestore TTL.
	// This should be set to a future timestamp and refreshed on each update.
	ExpiresAt time.Time `json:"expiresAt" firestore:"expiresAt"`
}

// NewCart creates a new cart doc.
// id is the Firestore docId (avatarId).
// items can be nil (treated as empty).
func NewCart(id string, items map[string]int, now time.Time) (*Cart, error) {
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

// Add increases quantity for a modelId.
// qty must be >= 1.
func (c *Cart) Add(modelID string, qty int, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
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

// ConsumeAll clears items for order creation and returns a snapshot of items.
// 想定ユースケース:
// 1) cart.Items を元に order を作成（order テーブルに保存）
// 2) 同トランザクション/同リクエスト内で cart.ConsumeAll() を呼び、items を空にする
//
// これにより「order が作成されることで items から modelId が削除される」を実現する。
func (c *Cart) ConsumeAll(now time.Time) (map[string]int, error) {
	if c == nil {
		return nil, ErrInvalidCart
	}
	// snapshot
	snap := cloneItems(c.Items)

	// clear
	if c.Items == nil {
		c.Items = map[string]int{}
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

// Consume removes specific modelIDs from the cart (partial consumption).
// order テーブル側が「一部の items を注文確定」する設計の場合に使う。
func (c *Cart) Consume(modelIDs []string, now time.Time) error {
	if c == nil {
		return ErrInvalidCart
	}
	if len(modelIDs) == 0 {
		return nil
	}
	if c.Items == nil {
		c.Items = map[string]int{}
		return nil
	}
	for _, id := range modelIDs {
		mid := strings.TrimSpace(id)
		if mid == "" {
			continue
		}
		delete(c.Items, mid)
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
