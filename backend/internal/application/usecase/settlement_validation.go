// backend/internal/application/usecase/settlement_validation.go
package usecase

import (
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Validation
// ============================================================

func validateSettlementSource(
	order orderdom.Order,
	payment paymentdom.Payment,
) error {
	if order.ID == "" {
		return ErrSettlementOrderIDInvalid
	}

	if payment.PaymentID == "" {
		return ErrSettlementPaymentIDInvalid
	}

	if payment.PaymentID != order.ID {
		return ErrSettlementPaymentOrderMismatch
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return ErrSettlementPaymentNotSucceeded
	}

	if payment.StripePaymentIntentID == "" {
		return ErrSettlementStripePaymentIntentIDMissing
	}

	if payment.StripeChargeID == "" {
		return ErrSettlementStripeChargeIDMissing
	}

	if payment.TransferGroup == "" {
		return ErrSettlementTransferGroupMissing
	}

	if payment.Amount <= 0 {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}

// validateListSettlementSeller validates the immutable primary-sale seller
// snapshot used to create a Stripe Settlement.
//
// List seller:
//
// - BrandID is required.
// - CompanyID is required.
// - AccountID is required.
// - StripeAccountID is required.
// - AvatarID, UserID and PayoutAccountID must be empty.
//
// Consumer resale seller snapshots are validated separately by the
// SalesReceivable creation flow and must never be converted into a Settlement
// SellerIdentity.
func validateListSettlementSeller(
	seller orderdom.SellerSnapshot,
) error {
	if seller.BrandID == "" ||
		seller.CompanyID == "" ||
		seller.AccountID == "" ||
		seller.StripeAccountID == "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	if seller.AvatarID != "" ||
		seller.UserID != "" ||
		seller.PayoutAccountID != "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	settlementSeller := settlementdom.SellerIdentity{
		Type:            settlementdom.SellerTypeAccount,
		CompanyID:       seller.CompanyID,
		AccountID:       seller.AccountID,
		StripeAccountID: seller.StripeAccountID,
	}

	if err := settlementSeller.Validate(); err != nil {
		return orderdom.ErrInvalidSellerSnapshot
	}

	return nil
}

// validateExistingSettlement verifies that an idempotent Create conflict points
// to the exact same immutable primary-sale Settlement.
//
// Seller identity and all financial/source fields are immutable. Runtime state
// fields such as Status, StripeTransferID and UpdatedAt are intentionally not
// compared because an existing Settlement may already have progressed after it
// was initially created.
func validateExistingSettlement(
	existing settlementdom.Settlement,
	expected settlementdom.Settlement,
) error {
	existingSeller := existing.SellerIdentity()
	expectedSeller := expected.SellerIdentity()

	if existingSeller != expectedSeller {
		return settlementdom.ErrConflict
	}

	if existing.ID != expected.ID ||
		existing.OrderID != expected.OrderID ||
		existing.PaymentID != expected.PaymentID ||
		existing.StripePaymentIntentID != expected.StripePaymentIntentID ||
		existing.StripeChargeID != expected.StripeChargeID ||
		existing.TransferGroup != expected.TransferGroup ||
		existing.GrossAmount != expected.GrossAmount ||
		existing.PlatformFeeAmount != expected.PlatformFeeAmount ||
		existing.TransferAmount != expected.TransferAmount ||
		existing.Currency != expected.Currency {
		return settlementdom.ErrConflict
	}

	return nil
}
