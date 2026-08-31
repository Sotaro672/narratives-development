// backend/internal/domain/order/snapshot.go
package order

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
	Type OrderItemType `json:"type,omitempty"`

	// List item identifiers.
	ListID      string `json:"listId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`

	// Resale item identifier.
	ResaleID string `json:"resaleId,omitempty"`

	OriginShippingAddressID      string `json:"originShippingAddressId,omitempty"`
	DestinationShippingAddressID string `json:"destinationShippingAddressId"`

	Carrier string `json:"carrier,omitempty"`

	TransportationID string `json:"transportationId,omitempty"`

	Size int `json:"size,omitempty"`

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

// SellerSnapshot fixes the seller identity and Stripe Connect destination at
// the time the Order is created.
//
// List sales use BrandID / CompanyID / AccountID.
// Resale sales use AvatarID / UserID / PayoutAccountID.
//
// StripeAccountID is common to both seller types and is the immutable Stripe
// Connect destination captured when the Order is created.
type SellerSnapshot struct {
	// List seller identifiers.
	BrandID   string `json:"brandId,omitempty"`
	CompanyID string `json:"companyId,omitempty"`
	AccountID string `json:"accountId,omitempty"`

	// Resale seller identifiers.
	AvatarID        string `json:"avatarId,omitempty"`
	UserID          string `json:"userId,omitempty"`
	PayoutAccountID string `json:"payoutAccountId,omitempty"`

	StripeAccountID string `json:"stripeAccountId"`
}
