// backend/internal/domain/refund/seller_identity.go
package refund

import "strings"

// SellerType identifies the seller payout identity associated with a Refund.
//
// account:
// Primary List sale paid through a Stripe Settlement.
//
// resale:
// Consumer resale paid through SalesReceivable and a future BankPayout.
type SellerType string

const (
	SellerTypeAccount SellerType = "account"
	SellerTypeResale  SellerType = "resale"
)

var AllowedSellerTypes = map[SellerType]struct{}{
	SellerTypeAccount: {},
	SellerTypeResale:  {},
}

// SellerIdentity is the immutable seller payout identity captured by a Refund.
//
// Account seller:
//   - CompanyID, AccountID and StripeAccountID are required.
//   - AvatarID, UserID and PayoutAccountID must be empty.
//
// Resale seller:
//   - AvatarID, UserID and PayoutAccountID are required.
//   - PayoutAccountID must equal UserID.
//   - CompanyID, AccountID and StripeAccountID must be empty.
type SellerIdentity struct {
	Type SellerType

	CompanyID string
	AccountID string

	AvatarID        string
	UserID          string
	PayoutAccountID string

	StripeAccountID string
}

func IsValidSellerType(
	sellerType SellerType,
) bool {
	if sellerType == "" {
		return false
	}

	_, ok := AllowedSellerTypes[sellerType]
	return ok
}

func (s SellerIdentity) Validate() error {
	if !IsValidSellerType(s.Type) {
		return ErrInvalidSellerType
	}

	switch s.Type {
	case SellerTypeAccount:
		if s.CompanyID == "" {
			return ErrInvalidCompanyID
		}
		if s.AccountID == "" {
			return ErrInvalidAccountID
		}
		if !isStripeAccountID(s.StripeAccountID) {
			return ErrInvalidStripeAccountID
		}
		if s.AvatarID != "" ||
			s.UserID != "" ||
			s.PayoutAccountID != "" {
			return ErrInvalidSellerIdentity
		}

	case SellerTypeResale:
		if s.AvatarID == "" {
			return ErrInvalidAvatarID
		}
		if s.UserID == "" {
			return ErrInvalidUserID
		}
		if s.PayoutAccountID == "" {
			return ErrInvalidPayoutAccountID
		}
		if s.PayoutAccountID != s.UserID {
			return ErrInvalidSellerIdentity
		}
		if s.CompanyID != "" ||
			s.AccountID != "" ||
			s.StripeAccountID != "" {
			return ErrInvalidSellerIdentity
		}

	default:
		return ErrInvalidSellerType
	}

	return nil
}

// SellerIdentity returns the immutable seller payout identity stored by this
// Refund.
func (r Refund) SellerIdentity() SellerIdentity {
	return SellerIdentity{
		Type:            r.SellerType,
		CompanyID:       r.CompanyID,
		AccountID:       r.AccountID,
		AvatarID:        r.AvatarID,
		UserID:          r.UserID,
		PayoutAccountID: r.PayoutAccountID,
		StripeAccountID: r.StripeAccountID,
	}
}

// SellerFinancialReferenceID returns the seller-side financial record associated
// with this Refund.
func (r Refund) SellerFinancialReferenceID() string {
	switch r.SellerType {
	case SellerTypeAccount:
		return r.SettlementID
	case SellerTypeResale:
		return r.SalesReceivableID
	default:
		return ""
	}
}

func (r Refund) validateSellerFinancialReference() error {
	switch r.SellerType {
	case SellerTypeAccount:
		if r.SettlementID == "" ||
			strings.Contains(r.SettlementID, "/") {
			return ErrInvalidSettlementID
		}
		if r.SalesReceivableID != "" {
			return ErrInvalidSalesReceivableID
		}

	case SellerTypeResale:
		if r.SettlementID != "" {
			return ErrInvalidSettlementID
		}
		if r.SalesReceivableID == "" ||
			strings.Contains(r.SalesReceivableID, "/") {
			return ErrInvalidSalesReceivableID
		}

	default:
		return ErrInvalidSellerType
	}

	return nil
}
