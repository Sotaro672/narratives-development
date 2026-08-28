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

func validateSettlementOrderItems(
	order orderdom.Order,
) error {
	if len(order.Items) == 0 {
		return ErrSettlementUnsupportedOrderItem
	}

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		// Current settlement implementation supports primary List sales only.
		//
		// Resale.BrandID identifies the product brand, not necessarily the
		// resale seller. A separate consumer payout identity is required before
		// resale proceeds can safely use Stripe Connect.
		if item.Type != orderdom.OrderItemTypeList {
			return ErrSettlementUnsupportedOrderItem
		}

		if item.SellerSnapshot.BrandID == "" ||
			item.SellerSnapshot.CompanyID == "" ||
			item.SellerSnapshot.AccountID == "" ||
			item.SellerSnapshot.StripeAccountID == "" {
			return orderdom.ErrInvalidSellerSnapshot
		}
	}

	return nil
}

func validateSettlementAllocations(
	payment paymentdom.Payment,
	allocations []settlementdom.Allocation,
) error {
	if len(allocations) == 0 {
		return ErrSettlementAllocationEmpty
	}

	maxInt := int(^uint(0) >> 1)

	seenAccounts := make(
		map[string]struct{},
		len(allocations),
	)

	total := 0

	for _, allocation := range allocations {
		if allocation.CompanyID == "" ||
			allocation.AccountID == "" ||
			allocation.StripeAccountID == "" {
			return ErrSettlementAllocationInvalid
		}

		if _, exists := seenAccounts[allocation.AccountID]; exists {
			return ErrSettlementDuplicateAccount
		}

		seenAccounts[allocation.AccountID] = struct{}{}

		if allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.TransferAmount <= 0 {
			return ErrSettlementAllocationInvalid
		}

		if allocation.PlatformFeeAmount >
			allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if allocation.TransferAmount >
			allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if allocation.GrossAmount-
			allocation.PlatformFeeAmount !=
			allocation.TransferAmount {
			return ErrSettlementAllocationInvalid
		}

		if total >
			maxInt-allocation.GrossAmount {
			return ErrSettlementAllocationAmountMismatch
		}

		total += allocation.GrossAmount
	}

	if total != payment.Amount {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}

func validateExistingSettlement(
	existing settlementdom.Settlement,
	expected settlementdom.Settlement,
) error {
	if existing.ID != expected.ID ||
		existing.OrderID != expected.OrderID ||
		existing.PaymentID != expected.PaymentID ||
		existing.CompanyID != expected.CompanyID ||
		existing.AccountID != expected.AccountID ||
		existing.StripeAccountID != expected.StripeAccountID ||
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
