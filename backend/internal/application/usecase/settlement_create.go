// backend/internal/application/usecase/settlement_create.go
package usecase

import (
	"context"
	"errors"
	"time"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	settlementdom "narratives/internal/domain/settlement"
)

var (
	ErrSettlementCombinedCalculatorMissing     = errors.New("settlement: combined settlement and receivable calculator is not configured")
	ErrSettlementSalesReceivableUsecaseMissing = errors.New("settlement: sales receivable usecase is not configured")
	ErrSettlementClockMissing                  = errors.New("settlement: clock is not configured")
)

type sellerFinancialAllocationCalculator interface {
	CalculateAll(
		ctx context.Context,
		order orderdom.Order,
		payment paymentdom.Payment,
	) (settlementdom.Calculation, error)
}

// EnsureForSucceededPayment guarantees the seller-side financial records for
// one successfully completed Payment.
//
// Primary List sales create Stripe Settlements.
// Consumer resale transactions create SalesReceivables.
//
// Both persistence paths use deterministic IDs and are safe to retry after a
// partial failure or duplicate Stripe webhook.
//
// Only primary-sale Settlement records are returned. Resale financial state is
// persisted through SalesReceivableUsecase.
func (u *SettlementUsecase) EnsureForSucceededPayment(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}
	if u.calculator == nil {
		return nil, ErrSettlementCalculatorMissing
	}
	if u.now == nil {
		return nil, ErrSettlementClockMissing
	}
	if err := validateSettlementSource(order, payment); err != nil {
		return nil, err
	}
	if err := validateSucceededPaymentSellerSnapshots(order); err != nil {
		return nil, err
	}

	calculator, ok := u.calculator.(sellerFinancialAllocationCalculator)
	if !ok {
		return nil, ErrSettlementCombinedCalculatorMissing
	}

	calculation, err := calculator.CalculateAll(ctx, order, payment)
	if err != nil {
		return nil, err
	}
	if err := validateSellerFinancialCalculation(payment, calculation); err != nil {
		return nil, err
	}
	if len(calculation.Receivables) > 0 && u.salesReceivableUC == nil {
		return nil, ErrSettlementSalesReceivableUsecaseMissing
	}

	now := u.now().UTC()
	result := make([]settlementdom.Settlement, 0, len(calculation.Settlements))

	for _, allocation := range calculation.Settlements {
		settlement, err := u.ensurePrimarySettlement(ctx, order, payment, allocation, now)
		if err != nil {
			return nil, err
		}
		result = append(result, settlement)
	}

	for _, allocation := range calculation.Receivables {
		_, err := u.salesReceivableUC.EnsurePending(
			ctx,
			EnsureSalesReceivableInput{
				OrderID:           order.ID,
				PaymentID:         payment.PaymentID,
				AvatarID:          allocation.Seller.AvatarID,
				UserID:            allocation.Seller.UserID,
				PayoutAccountID:   allocation.Seller.PayoutAccountID,
				GrossAmount:       allocation.GrossAmount,
				PlatformFeeAmount: allocation.PlatformFeeAmount,
				ReceivableAmount:  allocation.ReceivableAmount,
				Currency:          salesreceivabledom.CurrencyJPY,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (u *SettlementUsecase) ensurePrimarySettlement(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
	allocation settlementdom.Allocation,
	now time.Time,
) (settlementdom.Settlement, error) {
	if err := allocation.Seller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrSettlementAllocationInvalid
	}
	if allocation.Seller.Type != settlementdom.SellerTypeAccount {
		return settlementdom.Settlement{}, ErrSettlementAllocationInvalid
	}

	settlementID, err := settlementdom.NewID(
		payment.PaymentID,
		allocation.Seller,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	entity, err := settlementdom.New(
		settlementID,
		order.ID,
		payment.PaymentID,
		allocation.Seller,
		payment.StripePaymentIntentID,
		payment.StripeChargeID,
		payment.TransferGroup,
		allocation.GrossAmount,
		allocation.PlatformFeeAmount,
		allocation.TransferAmount,
		settlementdom.CurrencyJPY,
		settlementdom.StatusPending,
		now,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	created, err := u.repo.Create(
		ctx,
		settlementdom.CreateSettlementInput{
			SettlementID:          entity.ID,
			OrderID:               entity.OrderID,
			PaymentID:             entity.PaymentID,
			Seller:                entity.SellerIdentity(),
			StripePaymentIntentID: entity.StripePaymentIntentID,
			StripeChargeID:        entity.StripeChargeID,
			TransferGroup:         entity.TransferGroup,
			GrossAmount:           entity.GrossAmount,
			PlatformFeeAmount:     entity.PlatformFeeAmount,
			TransferAmount:        entity.TransferAmount,
			Currency:              entity.Currency,
			Status:                entity.Status,
		},
	)
	if err == nil {
		if err := validateExistingSettlement(created, entity); err != nil {
			return settlementdom.Settlement{}, err
		}
		return created, nil
	}
	if !errors.Is(err, settlementdom.ErrConflict) {
		return settlementdom.Settlement{}, err
	}

	existing, getErr := u.repo.GetByID(ctx, settlementID)
	if getErr != nil {
		return settlementdom.Settlement{}, getErr
	}
	if err := validateExistingSettlement(existing, entity); err != nil {
		return settlementdom.Settlement{}, err
	}

	return existing, nil
}

func validateSucceededPaymentSellerSnapshots(order orderdom.Order) error {
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
			if err := validateListSettlementSeller(item.SellerSnapshot); err != nil {
				return err
			}

		case orderdom.OrderItemTypeResale:
			if err := validateResaleReceivableSeller(item.SellerSnapshot); err != nil {
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

func validateResaleReceivableSeller(
	seller orderdom.SellerSnapshot,
) error {
	if seller.AvatarID == "" ||
		seller.UserID == "" ||
		seller.PayoutAccountID == "" ||
		seller.PayoutAccountID != seller.UserID {
		return orderdom.ErrInvalidSellerSnapshot
	}
	if seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" ||
		seller.StripeAccountID != "" {
		return orderdom.ErrInvalidSellerSnapshot
	}

	resolved := settlementdom.ResaleSellerIdentity{
		AvatarID:        seller.AvatarID,
		UserID:          seller.UserID,
		PayoutAccountID: seller.PayoutAccountID,
	}
	if err := resolved.Validate(); err != nil {
		return orderdom.ErrInvalidSellerSnapshot
	}

	return nil
}

func validateSellerFinancialCalculation(
	payment paymentdom.Payment,
	calculation settlementdom.Calculation,
) error {
	if len(calculation.Settlements) == 0 &&
		len(calculation.Receivables) == 0 {
		return ErrSettlementAllocationEmpty
	}

	maxInt := int(^uint(0) >> 1)
	seen := make(map[string]struct{}, len(calculation.Settlements)+len(calculation.Receivables))
	total := 0

	for _, allocation := range calculation.Settlements {
		if err := allocation.Seller.Validate(); err != nil ||
			allocation.Seller.Type != settlementdom.SellerTypeAccount {
			return ErrSettlementAllocationInvalid
		}

		sellerID, err := allocation.Seller.Key()
		if err != nil || sellerID == "" {
			return ErrSettlementAllocationInvalid
		}

		sellerKey := "account:" + sellerID
		if _, exists := seen[sellerKey]; exists {
			return ErrSettlementDuplicateSeller
		}
		seen[sellerKey] = struct{}{}

		if allocation.MerchandiseAmount < 0 ||
			allocation.MerchandiseTaxAmount < 0 ||
			allocation.ShippingAmount < 0 ||
			allocation.ShippingTaxAmount < 0 ||
			allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.TransferAmount <= 0 ||
			allocation.PlatformFeeAmount >= allocation.GrossAmount ||
			allocation.TransferAmount > allocation.GrossAmount ||
			allocation.GrossAmount-allocation.PlatformFeeAmount != allocation.TransferAmount {
			return ErrSettlementAllocationInvalid
		}

		calculatedGross := allocation.MerchandiseAmount
		if calculatedGross > maxInt-allocation.MerchandiseTaxAmount {
			return ErrSettlementAllocationInvalid
		}
		calculatedGross += allocation.MerchandiseTaxAmount
		if calculatedGross > maxInt-allocation.ShippingAmount {
			return ErrSettlementAllocationInvalid
		}
		calculatedGross += allocation.ShippingAmount
		if calculatedGross > maxInt-allocation.ShippingTaxAmount {
			return ErrSettlementAllocationInvalid
		}
		calculatedGross += allocation.ShippingTaxAmount
		if calculatedGross != allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if total > maxInt-allocation.GrossAmount {
			return ErrSettlementAllocationAmountMismatch
		}
		total += allocation.GrossAmount
	}

	for _, allocation := range calculation.Receivables {
		if err := allocation.Seller.Validate(); err != nil {
			return ErrSettlementAllocationInvalid
		}
		if allocation.MerchandiseAmount < 0 ||
			allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.ReceivableAmount <= 0 ||
			allocation.PlatformFeeAmount >= allocation.GrossAmount ||
			allocation.ReceivableAmount > allocation.GrossAmount ||
			allocation.GrossAmount-allocation.PlatformFeeAmount != allocation.ReceivableAmount ||
			allocation.MerchandiseAmount != allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		sellerKey := "resale:" + allocation.Seller.PayoutAccountID
		if _, exists := seen[sellerKey]; exists {
			return ErrSettlementDuplicateSeller
		}
		seen[sellerKey] = struct{}{}

		if total > maxInt-allocation.GrossAmount {
			return ErrSettlementAllocationAmountMismatch
		}
		total += allocation.GrossAmount
	}

	if total != payment.Amount {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}
