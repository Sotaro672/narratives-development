// backend/internal/domain/order/validation.go
package order

import transportationdom "narratives/internal/domain/transportation"

// ========================================
// Order validation
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
	if err := validateShippingSnapshot(o.ShippingSnapshot); err != nil {
		return err
	}
	if err := validateShippingQuoteSnapshot(o.ShippingQuoteSnapshot); err != nil {
		return err
	}
	if err := validatePaymentMethodSnapshot(o.PaymentMethodSnapshot); err != nil {
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

// ========================================
// Shipping validation
// ========================================

func validateShippingSnapshot(s ShippingSnapshot) error {
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

func validateShippingQuoteSnapshot(s ShippingQuoteSnapshot) error {
	if len(s.Items) == 0 {
		return ErrInvalidShippingQuote
	}
	if s.Amount < 0 {
		return ErrInvalidShippingQuote
	}
	if s.Currency != ShippingQuoteCurrencyJPY {
		return ErrInvalidShippingQuote
	}

	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, item := range s.Items {
		if err := validateShippingQuoteItemSnapshot(item); err != nil {
			return err
		}
		if total > maxInt-item.Amount {
			return ErrInvalidShippingQuote
		}

		total += item.Amount
	}

	if total != s.Amount {
		return ErrInvalidShippingQuote
	}

	return nil
}

func validateShippingQuoteItemSnapshot(item ShippingQuoteItemSnapshot) error {
	switch item.Type {
	case "", OrderItemTypeList:
		return validateListShippingQuoteItemSnapshot(item)
	case OrderItemTypeResale:
		return validateResaleShippingQuoteItemSnapshot(item)
	default:
		return ErrInvalidShippingQuoteItem
	}
}

func validateListShippingQuoteItemSnapshot(item ShippingQuoteItemSnapshot) error {
	if item.ListID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.InventoryID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.ModelID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.ResaleID != "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.OriginShippingAddressID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.DestinationShippingAddressID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if !isValidShippingQuoteCarrier(item.Carrier) {
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

	return validateShippingQuoteAmountSnapshot(item)
}

func validateResaleShippingQuoteItemSnapshot(item ShippingQuoteItemSnapshot) error {
	if item.ResaleID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.ListID != "" || item.InventoryID != "" || item.ModelID != "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.OriginShippingAddressID != "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.DestinationShippingAddressID == "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.TransportationID != "" {
		return ErrInvalidShippingQuoteItem
	}
	if item.Qty != 1 {
		return ErrInvalidShippingQuoteItem
	}
	if item.Currency != ShippingQuoteCurrencyJPY {
		return ErrInvalidShippingQuoteItem
	}

	if item.Carrier == "" {
		if item.Size != 0 || item.UnitAmount != 0 || item.Amount != 0 {
			return ErrInvalidShippingQuoteItem
		}
		return nil
	}

	carrier := transportationdom.Carrier(item.Carrier)
	if !transportationdom.IsValidResaleShippingCarrier(carrier) {
		return ErrInvalidShippingQuoteItem
	}
	if !transportationdom.IsValidResaleBoxSize(item.Size) {
		return ErrInvalidShippingQuoteItem
	}

	quote, err := transportationdom.CalculateResaleFlatRate(carrier, item.Size)
	if err != nil {
		return ErrInvalidShippingQuoteItem
	}
	if quote.Amount <= 0 {
		return ErrInvalidShippingQuoteItem
	}

	maxInt := int64(^uint(0) >> 1)
	if quote.Amount > maxInt {
		return ErrInvalidShippingQuoteItem
	}

	expectedAmount := int(quote.Amount)
	if item.UnitAmount != expectedAmount || item.Amount != expectedAmount {
		return ErrInvalidShippingQuoteItem
	}

	return validateShippingQuoteAmountSnapshot(item)
}

func validateShippingQuoteAmountSnapshot(item ShippingQuoteItemSnapshot) error {
	if item.Qty <= 0 {
		return ErrInvalidShippingQuoteItem
	}
	if item.UnitAmount < 0 {
		return ErrInvalidShippingQuoteItem
	}
	if item.Amount < 0 {
		return ErrInvalidShippingQuoteItem
	}
	if item.Currency != ShippingQuoteCurrencyJPY {
		return ErrInvalidShippingQuoteItem
	}

	maxInt := int(^uint(0) >> 1)
	if item.UnitAmount > 0 && item.Qty > maxInt/item.UnitAmount {
		return ErrInvalidShippingQuoteItem
	}
	if item.UnitAmount*item.Qty != item.Amount {
		return ErrInvalidShippingQuoteItem
	}

	return nil
}

func isValidShippingQuoteCarrier(carrier string) bool {
	switch carrier {
	case "yamato", "sagawa", "post", "custom":
		return true
	default:
		return false
	}
}

// ========================================
// Payment method validation
// ========================================

func validatePaymentMethodSnapshot(p PaymentMethodSnapshot) error {
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

// ========================================
// Item validation
// ========================================

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

func validateItemSnapshot(item OrderItemSnapshot) error {
	if err := validateProductBlueprintCategorySnapshot(item); err != nil {
		return err
	}

	switch item.Type {
	case OrderItemTypeList:
		if err := validateListSellerSnapshot(item.SellerSnapshot); err != nil {
			return err
		}
		if err := validateEmptyBrandRevenueSnapshot(item.BrandRevenueSnapshot); err != nil {
			return err
		}

		return validateListItemSnapshot(item)

	case OrderItemTypeResale:
		if err := validateResaleSellerSnapshot(item.SellerSnapshot); err != nil {
			return err
		}
		if err := validateResaleBrandRevenueSnapshot(item.BrandRevenueSnapshot, item.BrandID); err != nil {
			return err
		}

		return validateResaleItemSnapshot(item)

	default:
		return ErrInvalidItemSnapshot
	}
}

// ========================================
// Seller snapshot validation
// ========================================

func validateListSellerSnapshot(seller SellerSnapshot) error {
	if seller.BrandID == "" || seller.CompanyID == "" || seller.AccountID == "" {
		return ErrInvalidSellerSnapshot
	}
	if seller.AvatarID != "" || seller.UserID != "" || seller.PayoutAccountID != "" {
		return ErrInvalidSellerSnapshot
	}

	return validateListSellerStripeAccountID(seller.StripeAccountID)
}

func validateResaleSellerSnapshot(seller SellerSnapshot) error {
	if seller.AvatarID == "" || seller.UserID == "" || seller.PayoutAccountID == "" {
		return ErrInvalidSellerSnapshot
	}
	if seller.PayoutAccountID != seller.UserID {
		return ErrInvalidSellerSnapshot
	}
	if seller.BrandID != "" || seller.CompanyID != "" || seller.AccountID != "" {
		return ErrInvalidSellerSnapshot
	}
	if seller.StripeAccountID != "" {
		return ErrInvalidSellerSnapshot
	}

	return nil
}

func validateListSellerStripeAccountID(stripeAccountID string) error {
	if len(stripeAccountID) < len("acct_") || stripeAccountID[:len("acct_")] != "acct_" {
		return ErrInvalidSellerSnapshot
	}

	return nil
}

// ========================================
// Brand revenue snapshot validation
// ========================================

func validateEmptyBrandRevenueSnapshot(snapshot BrandRevenueSnapshot) error {
	if snapshot.BrandID != "" ||
		snapshot.CompanyID != "" ||
		snapshot.AccountID != "" ||
		snapshot.StripeAccountID != "" {
		return ErrInvalidItemSnapshot
	}

	return nil
}

func validateResaleBrandRevenueSnapshot(
	snapshot BrandRevenueSnapshot,
	itemBrandID string,
) error {
	if itemBrandID == "" ||
		snapshot.BrandID == "" ||
		snapshot.CompanyID == "" ||
		snapshot.AccountID == "" ||
		snapshot.StripeAccountID == "" {
		return ErrInvalidItemSnapshot
	}
	if snapshot.BrandID != itemBrandID {
		return ErrInvalidItemSnapshot
	}
	if len(snapshot.StripeAccountID) < len("acct_") ||
		snapshot.StripeAccountID[:len("acct_")] != "acct_" {
		return ErrInvalidItemSnapshot
	}

	return nil
}

// ========================================
// Product / item validation
// ========================================

func validateProductBlueprintCategorySnapshot(item OrderItemSnapshot) error {
	if len(item.ProductBlueprintCategoryPath) == 0 {
		return ErrInvalidItemSnapshot
	}

	for _, segment := range item.ProductBlueprintCategoryPath {
		if segment == "" {
			return ErrInvalidItemSnapshot
		}
	}

	switch item.ConsumptionTaxRate {
	case ConsumptionTaxRateReduced, ConsumptionTaxRateStandard:
		return nil
	default:
		return ErrInvalidItemSnapshot
	}
}

func validateListItemSnapshot(item OrderItemSnapshot) error {
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
	if item.ResaleID != "" || item.ProductID != "" || item.BrandID != "" {
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

func validateResaleItemSnapshot(item OrderItemSnapshot) error {
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

	// BrandID identifies the product brand. It must not be treated as the
	// consumer resale seller or compared with SellerSnapshot.BrandID.
	// BrandRevenueSnapshot separately identifies that Brand's immutable
	// Account and Stripe payout destination captured at Order creation.
	//
	// List-only identifiers must not be mixed into a resale item.
	if item.ModelID != "" || item.InventoryID != "" || item.ListID != "" {
		return ErrInvalidItemSnapshot
	}
	if item.Qty != 1 {
		return ErrInvalidItemSnapshot
	}
	if item.Price <= 0 {
		return ErrInvalidItemSnapshot
	}

	return validateItemTransferState(item)
}

// ========================================
// Item lifecycle validation
// ========================================

func validateItemTransferState(item OrderItemSnapshot) error {
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

	if item.TokenTransferVerifiedAt != nil && item.TokenTransferVerifiedAt.IsZero() {
		return ErrInvalidItemSnapshot
	}

	if item.IsReturnRequested {
		if item.IsCancelled ||
			!item.IsDispatched ||
			item.ReturnRequestedAt == nil ||
			item.ReturnRequestedAt.IsZero() {
			return ErrInvalidItemSnapshot
		}

		if !isValidReturnRequestKind(item.ReturnRequestKind) {
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
			item.ReturnCompletedAt.Before(item.ReturnRequestedAt.UTC()) {
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

		if item.TokenTransferVerifiedAt.After(*item.TransferredAt) {
			return ErrInvalidItemSnapshot
		}

		return nil
	}

	if item.TransferredAt != nil {
		return ErrInvalidItemSnapshot
	}

	return nil
}

func isValidReturnRequestKind(kind ReturnRequestKind) bool {
	switch kind {
	case ReturnRequestKindUnopened, ReturnRequestKindOpened:
		return true
	default:
		return false
	}
}
