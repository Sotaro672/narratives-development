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
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	Country string `json:"country"`
}

type ShippingQuoteItemSnapshot struct {
	ListID      string `json:"listId"`
	InventoryID string `json:"inventoryId"`
	ModelID     string `json:"modelId"`

	OriginShippingAddressID      string `json:"originShippingAddressId"`
	DestinationShippingAddressID string `json:"destinationShippingAddressId"`

	Carrier string `json:"carrier"`

	TransportationID string `json:"transportationId,omitempty"`

	Size int `json:"size"`

	Qty int `json:"qty"`

	UnitAmount int `json:"unitAmount"`
	Amount     int `json:"amount"`

	Currency string `json:"currency"`
}

type ShippingQuoteSnapshot struct {
	Items []ShippingQuoteItemSnapshot `json:"items"`

	Amount int `json:"amount"`

	Currency string `json:"currency"`
}

type PaymentMethodSnapshot struct {
	PaymentMethodID       string `json:"paymentMethodId"`
	CustomerID            string `json:"customerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`
	Brand                 string `json:"brand"`
	Last4                 string `json:"last4"`
	ExpMonth              int    `json:"expMonth"`
	ExpYear               int    `json:"expYear"`
	CardholderName        string `json:"cardholderName"`
	IsDefault             bool   `json:"isDefault"`
}

// SellerSnapshot fixes the seller and Stripe Connect destination at the
// time the Order is created.
//
// It is intentionally stored inside each Order item so later Brand or Account
// changes cannot silently change the settlement destination of an existing
// Order.
type SellerSnapshot struct {
	BrandID         string `json:"brandId"`
	CompanyID       string `json:"companyId"`
	AccountID       string `json:"accountId"`
	StripeAccountID string `json:"stripeAccountId"`
}

// OrderItemType identifies what kind of item is stored in Order.Items.
type OrderItemType string
type ReturnRequestKind string

const (
	OrderItemTypeList   OrderItemType = "list"
	OrderItemTypeResale OrderItemType = "resale"
)

const (
	ReturnRequestKindUnopened ReturnRequestKind = "unopened"
	ReturnRequestKindOpened   ReturnRequestKind = "opened"
)

// OrderItemSnapshot is stored inside Order.Items.
//
// List item:
//   - type: "list"
//   - modelId, inventoryId, listId
//   - productBlueprintId, tokenBlueprintId
//   - sellerSnapshot
//   - productBlueprintCategoryPath, consumptionTaxRate
//   - qty, price
//
// Resale item:
//   - type: "resale"
//   - resaleId, productId
//   - productBlueprintId, tokenBlueprintId, brandId
//   - sellerSnapshot
//   - productBlueprintCategoryPath, consumptionTaxRate
//   - qty=1, price
//
// Token transfer, cancellation, dispatch, and return state is maintained per item.
// Stripe Connect settlement state is maintained separately from this snapshot.
type OrderItemSnapshot struct {
	Type OrderItemType `json:"type"`

	// List item identifiers.
	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	// Resale item identifier.
	ResaleID string `json:"resaleId,omitempty"`

	// Product identifiers.
	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
	BrandID            string `json:"brandId,omitempty"`

	SellerSnapshot SellerSnapshot `json:"sellerSnapshot"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`

	ConsumptionTaxRate int `json:"consumptionTaxRate"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	IsReturnRequested bool              `json:"isReturnRequested"`
	ReturnRequestKind ReturnRequestKind `json:"returnRequestKind,omitempty"`
	ReturnRequestedAt *time.Time        `json:"returnRequestedAt,omitempty"`

	IsReturnCompleted bool       `json:"isReturnCompleted"`
	ReturnCompletedAt *time.Time `json:"returnCompletedAt,omitempty"`

	TokenTransferVerifiedAt *time.Time `json:"tokenTransferVerifiedAt,omitempty"`

	Transferred   bool       `json:"transferred"`
	TransferredAt *time.Time `json:"transferredAt,omitempty"`
}

// ========================================
// Entity
// ========================================

type Order struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot ShippingSnapshot `json:"shippingSnapshot"`

	ShippingQuoteSnapshot ShippingQuoteSnapshot `json:"shippingQuoteSnapshot"`

	PaymentMethodSnapshot PaymentMethodSnapshot `json:"paymentMethodSnapshot"`

	// Paid is maintained at the Order aggregate level.
	Paid bool `json:"paid"`

	Items     []OrderItemSnapshot `json:"items"`
	CreatedAt time.Time           `json:"createdAt"`
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID                = errors.New("order: invalid id")
	ErrInvalidUserID            = errors.New("order: invalid userId")
	ErrInvalidAvatarID          = errors.New("order: invalid avatarId")
	ErrInvalidCartID            = errors.New("order: invalid cartId")
	ErrInvalidShippingSnapshot  = errors.New("order: invalid shippingSnapshot")
	ErrInvalidShippingQuote     = errors.New("order: invalid shippingQuoteSnapshot")
	ErrInvalidShippingQuoteItem = errors.New(
		"order: invalid shippingQuoteItemSnapshot",
	)
	ErrInvalidPaymentMethod = errors.New(
		"order: invalid paymentMethodSnapshot",
	)
	ErrInvalidItems     = errors.New("order: invalid items")
	ErrInvalidCreatedAt = errors.New("order: invalid createdAt")

	ErrInvalidItemSnapshot   = errors.New("order: invalid item snapshot")
	ErrInvalidSellerSnapshot = errors.New("order: invalid sellerSnapshot")
)

// ========================================
// Policy
// ========================================

const (
	ShippingQuoteCurrencyJPY = "JPY"

	ConsumptionTaxRateReduced  = 8
	ConsumptionTaxRateStandard = 10
)

var (
	MinItemsRequired = 1
)

// ========================================
// Constructors
// ========================================

func New(
	id string,
	userID string,
	avatarID string,
	cartID string,
	shippingSnapshot ShippingSnapshot,
	shippingQuoteSnapshot ShippingQuoteSnapshot,
	paymentMethodSnapshot PaymentMethodSnapshot,
	items []OrderItemSnapshot,
	createdAt time.Time,
) (Order, error) {
	o := Order{
		ID:                    id,
		UserID:                userID,
		AvatarID:              avatarID,
		CartID:                cartID,
		ShippingSnapshot:      shippingSnapshot,
		ShippingQuoteSnapshot: shippingQuoteSnapshot,
		PaymentMethodSnapshot: paymentMethodSnapshot,
		Paid:                  false,
		Items:                 items,
		CreatedAt:             createdAt.UTC(),
	}

	if err := o.Validate(); err != nil {
		return Order{}, err
	}

	return o, nil
}

// ========================================
// Behavior (mutators)
// ========================================

func (o *Order) ReplaceItems(items []OrderItemSnapshot) error {
	if err := validateItems(items); err != nil {
		return err
	}

	o.Items = items
	return nil
}

func (o *Order) UpdateShippingSnapshot(
	s ShippingSnapshot,
) error {
	if err := validateShippingSnapshot(s); err != nil {
		return err
	}

	o.ShippingSnapshot = s
	return nil
}

func (o *Order) UpdateShippingQuoteSnapshot(
	s ShippingQuoteSnapshot,
) error {
	if err := validateShippingQuoteSnapshot(s); err != nil {
		return err
	}

	o.ShippingQuoteSnapshot =
		ShippingQuoteSnapshot{
			Items: append(
				[]ShippingQuoteItemSnapshot(nil),
				s.Items...,
			),
			Amount:   s.Amount,
			Currency: s.Currency,
		}

	return nil
}

func (o *Order) UpdatePaymentMethodSnapshot(
	p PaymentMethodSnapshot,
) error {
	if err := validatePaymentMethodSnapshot(p); err != nil {
		return err
	}

	o.PaymentMethodSnapshot = p
	return nil
}

func (o *Order) UpdateAvatarID(avatarID string) error {
	if avatarID == "" {
		return ErrInvalidAvatarID
	}

	o.AvatarID = avatarID
	return nil
}

func (o *Order) UpdatePaid(paid bool) {
	o.Paid = paid
}

func (o *Order) CancelItem(
	index int,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	item := &o.Items[index]

	if item.IsCancelled {
		return nil
	}

	if item.IsDispatched ||
		item.Transferred ||
		item.TokenTransferVerifiedAt != nil {
		return ErrConflict
	}

	item.IsCancelled = true
	return nil
}

func (o *Order) RequestItemReturn(
	index int,
	kind ReturnRequestKind,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if !isValidReturnRequestKind(kind) {
		return ErrInvalidItemSnapshot
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.IsReturnCompleted {
		return ErrConflict
	}

	switch kind {
	case ReturnRequestKindUnopened:
		if item.Transferred ||
			item.TokenTransferVerifiedAt != nil {
			return ErrConflict
		}

	case ReturnRequestKindOpened:
	}

	if item.IsReturnRequested {
		switch {
		case item.ReturnRequestKind == kind:
			return nil

		case item.ReturnRequestKind == "":
			item.ReturnRequestKind = kind
			return nil

		case item.ReturnRequestKind == ReturnRequestKindUnopened &&
			kind == ReturnRequestKindOpened:
			item.ReturnRequestKind = ReturnRequestKindOpened
			return nil

		case item.ReturnRequestKind == ReturnRequestKindOpened &&
			kind == ReturnRequestKindUnopened:
			return ErrConflict

		default:
			return ErrInvalidItemSnapshot
		}
	}

	returnRequestedAt := at.UTC()

	item.IsReturnRequested = true
	item.ReturnRequestKind = kind
	item.ReturnRequestedAt = &returnRequestedAt

	return nil
}

func (o *Order) CompleteItemReturn(
	index int,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsReturnCompleted {
		return nil
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		!item.IsReturnRequested ||
		item.ReturnRequestedAt == nil ||
		item.ReturnRequestedAt.IsZero() {
		return ErrConflict
	}

	returnCompletedAt := at.UTC()

	if returnCompletedAt.Before(
		item.ReturnRequestedAt.UTC(),
	) {
		return ErrInvalidItemSnapshot
	}

	item.IsReturnCompleted = true
	item.ReturnCompletedAt = &returnCompletedAt

	return nil
}

func (o *Order) MarkItemTokenTransferVerified(
	index int,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsCancelled ||
		item.IsReturnCompleted {
		return ErrConflict
	}

	if item.TokenTransferVerifiedAt != nil {
		return nil
	}

	verifiedAt := at.UTC()
	item.TokenTransferVerifiedAt = &verifiedAt

	return nil
}

func (o *Order) UpdateItemDispatched(
	index int,
	isDispatched bool,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if isDispatched &&
		o.Items[index].IsCancelled {
		return ErrConflict
	}

	if !isDispatched &&
		o.Items[index].IsReturnRequested {
		return ErrConflict
	}

	o.Items[index].IsDispatched = isDispatched
	return nil
}

// UpdateItemTransferred updates item-level transfer state.
// transferred=true requires a non-zero transferredAt value.
func (o *Order) UpdateItemTransferred(
	index int,
	transferred bool,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if transferred {
		if o.Items[index].IsCancelled ||
			o.Items[index].IsReturnRequested {
			return ErrConflict
		}

		if at.IsZero() {
			return ErrInvalidItemSnapshot
		}

		transferredAt := at.UTC()

		if o.Items[index].TokenTransferVerifiedAt == nil {
			verifiedAt := transferredAt
			o.Items[index].TokenTransferVerifiedAt = &verifiedAt
		}

		o.Items[index].Transferred = true
		o.Items[index].TransferredAt = &transferredAt
		return nil
	}

	o.Items[index].Transferred = false
	o.Items[index].TransferredAt = nil
	return nil
}

// ========================================
// Validation
// ========================================

// Validate verifies all invariants required for persisting an Order.
//
// Repository implementations must call Validate immediately before Create or
// Update so callers cannot bypass domain invariants by invoking the Repository
// directly.
func (o Order) Validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}

	if o.UserID == "" {
		return ErrInvalidUserID
	}

	if o.AvatarID == "" {
		return ErrInvalidAvatarID
	}

	if o.CartID == "" {
		return ErrInvalidCartID
	}

	if err := validateShippingSnapshot(
		o.ShippingSnapshot,
	); err != nil {
		return err
	}

	if err :=
		validateShippingQuoteSnapshot(
			o.ShippingQuoteSnapshot,
		); err != nil {
		return err
	}

	if err := validatePaymentMethodSnapshot(
		o.PaymentMethodSnapshot,
	); err != nil {
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

func validateShippingSnapshot(
	s ShippingSnapshot,
) error {
	if s.State == "" {
		return ErrInvalidShippingSnapshot
	}

	if s.City == "" {
		return ErrInvalidShippingSnapshot
	}

	if s.Street == "" {
		return ErrInvalidShippingSnapshot
	}

	if s.Country == "" {
		return ErrInvalidShippingSnapshot
	}

	return nil
}

func validateShippingQuoteSnapshot(
	s ShippingQuoteSnapshot,
) error {
	if len(s.Items) == 0 {
		return ErrInvalidShippingQuote
	}

	if s.Amount < 0 {
		return ErrInvalidShippingQuote
	}

	if s.Currency !=
		ShippingQuoteCurrencyJPY {
		return ErrInvalidShippingQuote
	}

	maxInt :=
		int(^uint(0) >> 1)

	total := 0

	for _, item := range s.Items {
		if err :=
			validateShippingQuoteItemSnapshot(
				item,
			); err != nil {
			return err
		}

		if total >
			maxInt-item.Amount {
			return ErrInvalidShippingQuote
		}

		total +=
			item.Amount
	}

	if total != s.Amount {
		return ErrInvalidShippingQuote
	}

	return nil
}

func validateShippingQuoteItemSnapshot(
	item ShippingQuoteItemSnapshot,
) error {
	if item.ListID == "" {
		return ErrInvalidShippingQuoteItem
	}

	if item.InventoryID == "" {
		return ErrInvalidShippingQuoteItem
	}

	if item.ModelID == "" {
		return ErrInvalidShippingQuoteItem
	}

	if item.OriginShippingAddressID == "" {
		return ErrInvalidShippingQuoteItem
	}

	if item.DestinationShippingAddressID == "" {
		return ErrInvalidShippingQuoteItem
	}

	if !isValidShippingQuoteCarrier(
		item.Carrier,
	) {
		return ErrInvalidShippingQuoteItem
	}

	if item.Carrier == "custom" {
		if item.TransportationID == "" {
			return ErrInvalidShippingQuoteItem
		}

		if item.Size != 0 {
			return ErrInvalidShippingQuoteItem
		}
	} else {
		if item.Size <= 0 {
			return ErrInvalidShippingQuoteItem
		}
	}

	if item.Qty <= 0 {
		return ErrInvalidShippingQuoteItem
	}

	if item.UnitAmount < 0 {
		return ErrInvalidShippingQuoteItem
	}

	if item.Amount < 0 {
		return ErrInvalidShippingQuoteItem
	}

	if item.Currency !=
		ShippingQuoteCurrencyJPY {
		return ErrInvalidShippingQuoteItem
	}

	maxInt :=
		int(^uint(0) >> 1)

	if item.UnitAmount > 0 &&
		item.Qty >
			maxInt/item.UnitAmount {
		return ErrInvalidShippingQuoteItem
	}

	if item.UnitAmount*
		item.Qty !=
		item.Amount {
		return ErrInvalidShippingQuoteItem
	}

	return nil
}

func isValidShippingQuoteCarrier(
	carrier string,
) bool {
	switch carrier {
	case "yamato",
		"sagawa",
		"post",
		"custom":
		return true

	default:
		return false
	}
}

func validatePaymentMethodSnapshot(
	p PaymentMethodSnapshot,
) error {
	if p.PaymentMethodID == "" {
		return ErrInvalidPaymentMethod
	}

	if p.CustomerID == "" {
		return ErrInvalidPaymentMethod
	}

	if p.StripePaymentMethodID == "" {
		return ErrInvalidPaymentMethod
	}

	if p.Brand == "" {
		return ErrInvalidPaymentMethod
	}

	if p.Last4 == "" {
		return ErrInvalidPaymentMethod
	}

	if p.ExpMonth < 1 || p.ExpMonth > 12 {
		return ErrInvalidPaymentMethod
	}

	if p.ExpYear < 2000 || p.ExpYear > 9999 {
		return ErrInvalidPaymentMethod
	}

	if p.CardholderName == "" {
		return ErrInvalidPaymentMethod
	}

	return nil
}

func validateItems(items []OrderItemSnapshot) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}

	for _, item := range items {
		if err := validateItemSnapshot(item); err != nil {
			return err
		}
	}

	return nil
}

func validateItemSnapshot(
	item OrderItemSnapshot,
) error {
	if err := validateSellerSnapshot(
		item.SellerSnapshot,
	); err != nil {
		return err
	}

	if err :=
		validateProductBlueprintCategorySnapshot(
			item,
		); err != nil {
		return err
	}

	switch item.Type {
	case OrderItemTypeList:
		return validateListItemSnapshot(item)

	case OrderItemTypeResale:
		return validateResaleItemSnapshot(item)

	default:
		return ErrInvalidItemSnapshot
	}
}

func validateSellerSnapshot(
	seller SellerSnapshot,
) error {
	if seller.BrandID == "" {
		return ErrInvalidSellerSnapshot
	}

	if seller.CompanyID == "" {
		return ErrInvalidSellerSnapshot
	}

	if seller.AccountID == "" {
		return ErrInvalidSellerSnapshot
	}

	if seller.StripeAccountID == "" ||
		!strings.HasPrefix(
			seller.StripeAccountID,
			"acct_",
		) {
		return ErrInvalidSellerSnapshot
	}

	return nil
}

func validateProductBlueprintCategorySnapshot(
	item OrderItemSnapshot,
) error {
	if len(
		item.ProductBlueprintCategoryPath,
	) == 0 {
		return ErrInvalidItemSnapshot
	}

	for _, segment := range item.ProductBlueprintCategoryPath {
		if segment == "" {
			return ErrInvalidItemSnapshot
		}
	}

	switch item.ConsumptionTaxRate {
	case ConsumptionTaxRateReduced,
		ConsumptionTaxRateStandard:
		return nil

	default:
		return ErrInvalidItemSnapshot
	}
}

func validateListItemSnapshot(
	item OrderItemSnapshot,
) error {
	if item.ModelID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.InventoryID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.ListID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.ProductBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.TokenBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}

	// Resale-only identifiers must not be mixed into a list item.
	if item.ResaleID != "" ||
		item.ProductID != "" ||
		item.BrandID != "" {
		return ErrInvalidItemSnapshot
	}

	if item.Qty <= 0 {
		return ErrInvalidItemSnapshot
	}

	if item.Price < 0 {
		return ErrInvalidItemSnapshot
	}

	return validateItemTransferState(item)
}

func validateResaleItemSnapshot(
	item OrderItemSnapshot,
) error {
	if item.ResaleID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.ProductID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.ProductBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.TokenBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.BrandID == "" {
		return ErrInvalidItemSnapshot
	}

	if item.BrandID !=
		item.SellerSnapshot.BrandID {
		return ErrInvalidItemSnapshot
	}

	// List-only identifiers must not be mixed into a resale item.
	if item.ModelID != "" ||
		item.InventoryID != "" ||
		item.ListID != "" {
		return ErrInvalidItemSnapshot
	}

	if item.Qty != 1 {
		return ErrInvalidItemSnapshot
	}

	if item.Price < 0 {
		return ErrInvalidItemSnapshot
	}

	return validateItemTransferState(item)
}

func validateItemTransferState(
	item OrderItemSnapshot,
) error {
	if item.IsCancelled {
		if item.IsDispatched ||
			item.IsReturnRequested ||
			item.ReturnRequestKind != "" ||
			item.ReturnRequestedAt != nil ||
			item.IsReturnCompleted ||
			item.ReturnCompletedAt != nil ||
			item.TokenTransferVerifiedAt != nil ||
			item.Transferred ||
			item.TransferredAt != nil {
			return ErrInvalidItemSnapshot
		}
	}

	if item.TokenTransferVerifiedAt != nil &&
		item.TokenTransferVerifiedAt.IsZero() {
		return ErrInvalidItemSnapshot
	}

	if item.IsReturnRequested {
		if item.IsCancelled ||
			!item.IsDispatched ||
			item.ReturnRequestedAt == nil ||
			item.ReturnRequestedAt.IsZero() {
			return ErrInvalidItemSnapshot
		}

		// Legacy Orders created before ReturnRequestKind was persisted may have
		// an empty value here. New return requests always persist the kind, but
		// the empty value remains valid for backward compatibility.
		if item.ReturnRequestKind != "" &&
			!isValidReturnRequestKind(item.ReturnRequestKind) {
			return ErrInvalidItemSnapshot
		}
	} else {
		if item.ReturnRequestKind != "" ||
			item.ReturnRequestedAt != nil ||
			item.IsReturnCompleted ||
			item.ReturnCompletedAt != nil {
			return ErrInvalidItemSnapshot
		}
	}

	if item.IsReturnCompleted {
		if item.ReturnCompletedAt == nil ||
			item.ReturnCompletedAt.IsZero() ||
			item.ReturnCompletedAt.Before(
				item.ReturnRequestedAt.UTC(),
			) {
			return ErrInvalidItemSnapshot
		}
	} else if item.ReturnCompletedAt != nil {
		return ErrInvalidItemSnapshot
	}

	if item.Transferred {
		if item.TransferredAt == nil ||
			item.TransferredAt.IsZero() ||
			item.TokenTransferVerifiedAt == nil ||
			item.TokenTransferVerifiedAt.IsZero() {
			return ErrInvalidItemSnapshot
		}

		if item.TokenTransferVerifiedAt.After(
			*item.TransferredAt,
		) {
			return ErrInvalidItemSnapshot
		}

		return nil
	}

	if item.TransferredAt != nil {
		return ErrInvalidItemSnapshot
	}

	return nil
}

func isValidReturnRequestKind(
	kind ReturnRequestKind,
) bool {
	switch kind {
	case ReturnRequestKindUnopened,
		ReturnRequestKindOpened:
		return true

	default:
		return false
	}
}
