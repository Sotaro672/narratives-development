// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"strings"
	"time"
)

// ========================================
// Snapshot structs (stored in Order)
// ========================================

type ShippingSnapshot struct {
	ZipCode string
	State   string
	City    string
	Street  string
	Street2 string
	Country string
}

type BillingSnapshot struct {
	Last4          string
	CardHolderName string
	// NOTE: cardId は保持しない（この構造体に元々存在しないため対応不要）
}

// OrderItemSnapshot is stored inside Order.Items.
// Expectation: items are NOT split by listId, and each item is
// [modelId, inventoryId, qty, price].
//
// ✅ NEW:
// - transferred / transferredAt を item 単位で保持する（複数商品の部分移転に対応）
type OrderItemSnapshot struct {
	ModelID     string `json:"modelId"`
	InventoryID string `json:"inventoryId"`
	Qty         int    `json:"qty"`
	Price       int    `json:"price"`

	// ✅ NEW
	Transferred   bool       `json:"transferred"`
	TransferredAt *time.Time `json:"transferredAt,omitempty"`
}

// ========================================
// Entity
// ========================================

type Order struct {
	ID       string
	UserID   string
	AvatarID string
	CartID   string

	ShippingSnapshot ShippingSnapshot
	BillingSnapshot  BillingSnapshot

	// ✅ paid は Order 全体で保持してOK（支払いは注文単位）
	Paid bool `json:"paid"`

	Items     []OrderItemSnapshot `json:"items"`
	CreatedAt time.Time
}

// OrderPatch represents partial updates to Order fields.
// A nil field means "no change".
type OrderPatch struct {
	UserID   *string
	AvatarID *string // ✅ NEW
	CartID   *string

	ShippingSnapshot *ShippingSnapshot
	BillingSnapshot  *BillingSnapshot

	// ✅ paid は注文単位のまま
	Paid *bool

	Items *[]OrderItemSnapshot
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID              = errors.New("order: invalid id")
	ErrInvalidUserID          = errors.New("order: invalid userId")
	ErrInvalidAvatarID        = errors.New("order: invalid avatarId") // ✅ NEW
	ErrInvalidCartID          = errors.New("order: invalid cartId")
	ErrInvalidShippingAddress = errors.New("order: invalid shippingSnapshot")
	ErrInvalidBillingAddress  = errors.New("order: invalid billingSnapshot")
	ErrInvalidItems           = errors.New("order: invalid items")
	ErrInvalidCreatedAt       = errors.New("order: invalid createdAt")

	ErrInvalidItemSnapshot = errors.New("order: invalid item snapshot")
)

// ========================================
// Policy
// ========================================

var (
	MinItemsRequired = 1
)

// ========================================
// Constructors
// ========================================

func New(
	id string,
	userID string,
	avatarID string, // ✅ NEW
	cartID string,
	shippingSnapshot ShippingSnapshot,
	billingSnapshot BillingSnapshot,
	items []OrderItemSnapshot,
	createdAt time.Time,
) (Order, error) {
	o := Order{
		ID:       strings.TrimSpace(id),
		UserID:   strings.TrimSpace(userID),
		AvatarID: strings.TrimSpace(avatarID), // ✅ NEW
		CartID:   strings.TrimSpace(cartID),

		ShippingSnapshot: normalizeShippingSnapshot(shippingSnapshot),
		BillingSnapshot:  normalizeBillingSnapshot(billingSnapshot),

		// ✅ paid は起票時 false
		Paid: false,

		Items:     normalizeItems(items),
		CreatedAt: createdAt.UTC(),
	}
	if err := o.validate(); err != nil {
		return Order{}, err
	}
	return o, nil
}

// ========================================
// Behavior (mutators)
// ========================================

func (o *Order) ReplaceItems(items []OrderItemSnapshot) error {
	ns := normalizeItems(items)
	if err := validateItems(ns); err != nil {
		return err
	}
	o.Items = ns
	return nil
}

// ✅ Replace AddressID update with Snapshot update
func (o *Order) UpdateShippingSnapshot(s ShippingSnapshot) error {
	s = normalizeShippingSnapshot(s)
	if err := validateShippingSnapshot(s); err != nil {
		return err
	}
	o.ShippingSnapshot = s
	return nil
}

func (o *Order) UpdateBillingSnapshot(b BillingSnapshot) error {
	b = normalizeBillingSnapshot(b)
	if err := validateBillingSnapshot(b); err != nil {
		return err
	}
	o.BillingSnapshot = b
	return nil
}

// ✅ NEW: avatarId update
func (o *Order) UpdateAvatarID(avatarID string) error {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return ErrInvalidAvatarID
	}
	o.AvatarID = avatarID
	return nil
}

// ✅ paid update（注文単位）
func (o *Order) UpdatePaid(paid bool) {
	o.Paid = paid
}

// ✅ NEW: item 単位 transferred update
// - 指定 index の item.Transfered を更新し、TransferredAt も整合する
func (o *Order) UpdateItemTransferred(index int, transferred bool, at time.Time) error {
	if o == nil {
		return ErrInvalidItems
	}
	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	o.Items[index].Transferred = transferred
	if transferred {
		t := at.UTC()
		o.Items[index].TransferredAt = &t
		return nil
	}

	// transferred=false に戻すなら日時も消す
	o.Items[index].TransferredAt = nil
	return nil
}

// ========================================
// Validation
// ========================================

func (o Order) validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.UserID == "" {
		return ErrInvalidUserID
	}
	if o.AvatarID == "" { // ✅ NEW
		return ErrInvalidAvatarID
	}
	if o.CartID == "" {
		return ErrInvalidCartID
	}
	if err := validateShippingSnapshot(o.ShippingSnapshot); err != nil {
		return err
	}
	if err := validateBillingSnapshot(o.BillingSnapshot); err != nil {
		return err
	}
	if err := validateItems(o.Items); err != nil {
		return err
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}

func validateShippingSnapshot(s ShippingSnapshot) error {
	if strings.TrimSpace(s.State) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.City) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.Street) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.Country) == "" {
		return ErrInvalidShippingAddress
	}
	return nil
}

func validateBillingSnapshot(b BillingSnapshot) error {
	last4 := strings.TrimSpace(b.Last4)
	if last4 == "" {
		return ErrInvalidBillingAddress
	}
	// cardHolderName は任意（空でもOK）
	return nil
}

func validateItems(items []OrderItemSnapshot) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}
	for _, it := range items {
		if strings.TrimSpace(it.ModelID) == "" {
			return ErrInvalidItemSnapshot
		}
		if strings.TrimSpace(it.InventoryID) == "" {
			return ErrInvalidItemSnapshot
		}
		if it.Qty <= 0 {
			return ErrInvalidItemSnapshot
		}
		if it.Price < 0 {
			return ErrInvalidItemSnapshot
		}

		// ✅ NEW: transferredAt の整合性
		// - transferred=true なら transferredAt があるべき（ただし過去データ互換のため「必須」にはしない）
		// - transferred=false なら transferredAt は nil が望ましい（こちらも必須にはしない）
		// 厳密にしたい場合はここでエラーにしてOK
	}
	return nil
}

// ========================================
// Helpers
// ========================================

func normalizeShippingSnapshot(s ShippingSnapshot) ShippingSnapshot {
	s.ZipCode = strings.TrimSpace(s.ZipCode)
	s.State = strings.TrimSpace(s.State)
	s.City = strings.TrimSpace(s.City)
	s.Street = strings.TrimSpace(s.Street)
	s.Street2 = strings.TrimSpace(s.Street2)
	s.Country = strings.TrimSpace(s.Country)
	return s
}

func normalizeBillingSnapshot(b BillingSnapshot) BillingSnapshot {
	b.Last4 = strings.TrimSpace(b.Last4)
	b.CardHolderName = strings.TrimSpace(b.CardHolderName)
	return b
}

func normalizeItems(items []OrderItemSnapshot) []OrderItemSnapshot {
	out := make([]OrderItemSnapshot, 0, len(items))
	for _, it := range items {
		n := OrderItemSnapshot{
			ModelID:     strings.TrimSpace(it.ModelID),
			InventoryID: strings.TrimSpace(it.InventoryID),
			Qty:         it.Qty,
			Price:       it.Price,

			// ✅ NEW: 起票時の item transfer 状態は false / nil
			Transferred:   false,
			TransferredAt: nil,
		}

		// 既存入力に transferred が入っている場合は尊重（再構築やimport時）
		// ただし transferredAt は transferred と整合するように正規化
		if it.Transferred {
			n.Transferred = true
			if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
				t := it.TransferredAt.UTC()
				n.TransferredAt = &t
			} else {
				// transferred=true で時刻が無い場合は nil のまま（repository/update 側で補完してもOK）
				n.TransferredAt = nil
			}
		} else {
			// transferred=false の場合、TransferredAt は基本 nil に寄せる
			n.TransferredAt = nil
		}

		// 空は validateItems で弾くのでここでは落とさない
		out = append(out, n)
	}
	return out
}
