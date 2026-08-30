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

	activeItemCount := 0

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		activeItemCount++

		switch item.Type {
		case orderdom.OrderItemTypeList:
			if err := validateListSettlementSeller(
				item.SellerSnapshot,
			); err != nil {
				return err
			}

		case orderdom.OrderItemTypeResale:
			if err := validateResaleSettlementSeller(
				item.SellerSnapshot,
			); err != nil {
				return err
			}

		default:
			return ErrSettlementUnsupportedOrderItem
		}
	}

	if activeItemCount == 0 {
		return ErrSettlementUnsupportedOrderItem
	}

	return nil
}

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

func validateResaleSettlementSeller(
	seller orderdom.SellerSnapshot,
) error {
	if seller.AvatarID == "" ||
		seller.UserID == "" ||
		seller.PayoutAccountID == "" ||
		seller.StripeAccountID == "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	if seller.PayoutAccountID != seller.UserID {
		return orderdom.ErrInvalidSellerSnapshot
	}

	if seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	settlementSeller := settlementdom.SellerIdentity{
		Type:            settlementdom.SellerTypeAvatar,
		AvatarID:        seller.AvatarID,
		UserID:          seller.UserID,
		PayoutAccountID: seller.PayoutAccountID,
		StripeAccountID: seller.StripeAccountID,
	}

	if err := settlementSeller.Validate(); err != nil {
		return orderdom.ErrInvalidSellerSnapshot
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

	seenSellers := make(
		map[string]struct{},
		len(allocations),
	)

	total := 0

	for _, allocation := range allocations {
		if err := allocation.Seller.Validate(); err != nil {
			return ErrSettlementAllocationInvalid
		}

		sellerID, err := allocation.Seller.Key()
		if err != nil || sellerID == "" {
			return ErrSettlementAllocationInvalid
		}

		sellerKey :=
			string(allocation.Seller.Type) +
				":" +
				sellerID

		if _, exists := seenSellers[sellerKey]; exists {
			return ErrSettlementDuplicateSeller
		}

		seenSellers[sellerKey] = struct{}{}

		if allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.TransferAmount <= 0 {
			return ErrSettlementAllocationInvalid
		}

		if allocation.MerchandiseAmount < 0 ||
			allocation.MerchandiseTaxAmount < 0 ||
			allocation.ShippingAmount < 0 ||
			allocation.ShippingTaxAmount < 0 {
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

		calculatedGrossAmount := allocation.MerchandiseAmount

		if calculatedGrossAmount >
			maxInt-allocation.MerchandiseTaxAmount {
			return ErrSettlementAllocationInvalid
		}

		calculatedGrossAmount +=
			allocation.MerchandiseTaxAmount

		if calculatedGrossAmount >
			maxInt-allocation.ShippingAmount {
			return ErrSettlementAllocationInvalid
		}

		calculatedGrossAmount +=
			allocation.ShippingAmount

		if calculatedGrossAmount >
			maxInt-allocation.ShippingTaxAmount {
			return ErrSettlementAllocationInvalid
		}

		calculatedGrossAmount +=
			allocation.ShippingTaxAmount

		if calculatedGrossAmount !=
			allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		switch allocation.Seller.Type {
		case settlementdom.SellerTypeAccount:
			// Primary List settlement may contain merchandise tax and
			// shipping amounts according to the Order snapshot.

		case settlementdom.SellerTypeAvatar:
			// Resale is non-taxable for the purchaser and has no
			// purchaser-side shipping charge.
			if allocation.MerchandiseTaxAmount != 0 ||
				allocation.ShippingAmount != 0 ||
				allocation.ShippingTaxAmount != 0 {
				return ErrSettlementAllocationInvalid
			}

		default:
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
