// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"fmt"
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
}

// ========================================
// Entity (mirror TS Order)
// ========================================

type Order struct {
	ID     string
	UserID string
	CartID string

	// ✅ Snapshot (replace AddressID reference)
	ShippingSnapshot ShippingSnapshot
	BillingSnapshot  BillingSnapshot // last4 + cardHolderName only

	ListID         string
	Items          []string `json:"items"`
	InvoiceID      string
	PaymentID      string
	TransferedDate *time.Time // note: mirrors TS: transferedDate
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdatedBy      *string
}

// OrderPatch represents partial updates to Order fields.
// A nil field means "no change".
type OrderPatch struct {
	UserID *string
	CartID *string

	ShippingSnapshot *ShippingSnapshot
	BillingSnapshot  *BillingSnapshot

	ListID         *string
	Items          *[]string
	InvoiceID      *string
	PaymentID      *string
	TransferedDate *time.Time
	UpdatedBy      *string
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID              = errors.New("order: invalid id")
	ErrInvalidUserID          = errors.New("order: invalid userId")
	ErrInvalidCartID          = errors.New("order: invalid cartId")
	ErrInvalidShippingAddress = errors.New("order: invalid shippingSnapshot") // keep name for easier migration
	ErrInvalidBillingAddress  = errors.New("order: invalid billingSnapshot")  // keep name for easier migration
	ErrInvalidListID          = errors.New("order: invalid listId")
	ErrInvalidItems           = errors.New("order: invalid items")
	ErrInvalidInvoiceID       = errors.New("order: invalid invoiceId")
	ErrInvalidPaymentID       = errors.New("order: invalid paymentId")
	ErrInvalidTransferredDate = errors.New("order: invalid transferredDate")
	ErrInvalidCreatedAt       = errors.New("order: invalid createdAt")
	ErrInvalidUpdatedAt       = errors.New("order: invalid updatedAt")
	ErrInvalidUpdatedBy       = errors.New("order: invalid updatedBy")
	ErrInvalidItemID          = errors.New("order: invalid item id")
)

// ========================================
// Policy (align with orderConstants.ts if any)
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
	cartID string,
	shippingSnapshot ShippingSnapshot,
	billingSnapshot BillingSnapshot,
	listID string,
	items []string,
	invoiceID string,
	paymentID string,
	transferedDate *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
	updatedBy *string,
) (Order, error) {
	o := Order{
		ID:     strings.TrimSpace(id),
		UserID: strings.TrimSpace(userID),
		CartID: strings.TrimSpace(cartID),

		ShippingSnapshot: normalizeShippingSnapshot(shippingSnapshot),
		BillingSnapshot:  normalizeBillingSnapshot(billingSnapshot),

		ListID:         strings.TrimSpace(listID),
		Items:          dedupTrim(items),
		InvoiceID:      strings.TrimSpace(invoiceID),
		PaymentID:      strings.TrimSpace(paymentID),
		TransferedDate: normalizeTimePtr(transferedDate),
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
		UpdatedBy:      normalizePtr(updatedBy),
	}
	if err := o.validate(); err != nil {
		return Order{}, err
	}
	return o, nil
}

func NewFromStringTimes(
	id string,
	userID string,
	cartID string,
	shippingSnapshot ShippingSnapshot,
	billingSnapshot BillingSnapshot,
	listID string,
	items []string,
	invoiceID string,
	paymentID string,
	transferedDateStr *string,
	createdAtStr string,
	updatedAtStr string,
	updatedBy *string,
) (Order, error) {
	ca, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Order{}, err
	}
	ua, err := parseTime(updatedAtStr, ErrInvalidUpdatedAt)
	if err != nil {
		return Order{}, err
	}

	var td *time.Time
	if transferedDateStr != nil && strings.TrimSpace(*transferedDateStr) != "" {
		t, err := parseTime(*transferedDateStr, ErrInvalidTransferredDate)
		if err != nil {
			return Order{}, err
		}
		td = &t
	}

	return New(
		id,
		userID,
		cartID,
		shippingSnapshot,
		billingSnapshot,
		listID,
		items,
		invoiceID,
		paymentID,
		td,
		ca,
		ua,
		updatedBy,
	)
}

// ========================================
// Behavior (mutators)
// ========================================

func (o *Order) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	o.UpdatedAt = now.UTC()
	return nil
}

// New name to mirror TS field "transferedDate"
func (o *Order) SetTransfered(at time.Time, now time.Time) error {
	if at.IsZero() {
		return ErrInvalidTransferredDate
	}
	utc := at.UTC()
	o.TransferedDate = &utc
	return o.Touch(now)
}

// Backward-compat: keep old method name delegating to new one
func (o *Order) SetTransferred(at time.Time, now time.Time) error {
	return o.SetTransfered(at, now)
}

func (o *Order) ReplaceItems(items []string, now time.Time) error {
	ds := dedupTrim(items)
	if err := validateItems(ds); err != nil {
		return err
	}
	o.Items = ds
	return o.Touch(now)
}

func (o *Order) AddItem(itemID string, now time.Time) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidItems
	}
	if !contains(o.Items, itemID) {
		o.Items = append(o.Items, itemID)
	}
	return o.Touch(now)
}

func (o *Order) RemoveItem(itemID string, now time.Time) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidItems
	}
	out := o.Items[:0]
	for _, it := range o.Items {
		if it != itemID {
			out = append(out, it)
		}
	}
	o.Items = out
	return o.Touch(now)
}

// ✅ Replace AddressID update with Snapshot update
func (o *Order) UpdateShippingSnapshot(s ShippingSnapshot, now time.Time) error {
	s = normalizeShippingSnapshot(s)
	if err := validateShippingSnapshot(s); err != nil {
		return err
	}
	o.ShippingSnapshot = s
	return o.Touch(now)
}

func (o *Order) UpdateBillingSnapshot(b BillingSnapshot, now time.Time) error {
	b = normalizeBillingSnapshot(b)
	if err := validateBillingSnapshot(b); err != nil {
		return err
	}
	o.BillingSnapshot = b
	return o.Touch(now)
}

func (o *Order) UpdateInvoice(invoiceID string, now time.Time) error {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return ErrInvalidInvoiceID
	}
	o.InvoiceID = invoiceID
	return o.Touch(now)
}

func (o *Order) UpdatePayment(paymentID string, now time.Time) error {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return ErrInvalidPaymentID
	}
	o.PaymentID = paymentID
	return o.Touch(now)
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
	if o.CartID == "" {
		return ErrInvalidCartID
	}
	if err := validateShippingSnapshot(o.ShippingSnapshot); err != nil {
		return err
	}
	if err := validateBillingSnapshot(o.BillingSnapshot); err != nil {
		return err
	}
	if o.ListID == "" {
		return ErrInvalidListID
	}
	if err := validateItems(o.Items); err != nil {
		return err
	}
	if o.InvoiceID == "" {
		return ErrInvalidInvoiceID
	}
	if o.PaymentID == "" {
		return ErrInvalidPaymentID
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if o.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	// Time order
	if o.UpdatedAt.Before(o.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if o.TransferedDate != nil && (o.TransferedDate.IsZero() || o.TransferedDate.Before(o.CreatedAt)) {
		return ErrInvalidTransferredDate
	}
	// UpdatedBy optional but if set must be non-empty
	if o.UpdatedBy != nil && strings.TrimSpace(*o.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	// validate で Items が orderItem の主キー(ID)として妥当か検証します（空文字は不可）。
	for _, id := range o.Items {
		if strings.TrimSpace(id) == "" {
			return ErrInvalidItemID
		}
	}
	return nil
}

func validateShippingSnapshot(s ShippingSnapshot) error {
	// 必須（あなたのUI/運用前提に合わせて最小限を必須化）
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
	// ZipCode/Street2 は任意（国によってはZipがない/運用で省略あり得る）
	return nil
}

func validateBillingSnapshot(b BillingSnapshot) error {
	// ✅ last4 は必須（billingAddressID 必須だったのと同等の強さ）
	// cardHolderName は任意（空なら空で良い）
	last4 := strings.TrimSpace(b.Last4)
	if last4 == "" {
		return ErrInvalidBillingAddress
	}
	// "4桁" 以外を弾きたいならここで追加
	// if len(last4) != 4 || !isDigits(last4) { return ErrInvalidBillingAddress }
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

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	utc := p.UTC()
	return &utc
}

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func validateItems(items []string) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}
	for _, it := range items {
		if strings.TrimSpace(it) == "" {
			return ErrInvalidItems
		}
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if _, ok := seen[it]; ok {
			return ErrInvalidItems
		}
		seen[it] = struct{}{}
	}
	return nil
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}
