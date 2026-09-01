// backend/internal/application/usecase/settlement_create.go
package usecase

import (
	"context"
	"errors"
	"time"

	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	settlementdom "narratives/internal/domain/settlement"
)

var (
	ErrSettlementCombinedCalculatorMissing        = errors.New("settlement: combined settlement and receivable calculator is not configured")
	ErrSettlementSalesReceivableUsecaseMissing    = errors.New("settlement: sales receivable usecase is not configured")
	ErrSettlementBrandFeeSettlementUsecaseMissing = errors.New("settlement: brand fee settlement usecase is not configured")
	ErrSettlementClockMissing                     = errors.New("settlement: clock is not configured")
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
//
// Each consumer resale Order item creates:
//   - one SalesReceivable for the resale seller
//   - one BrandFeeSettlement for the productBlueprint Brand
//
// Resale allocation:
//
//	MerchandiseAmount - ShippingAmount = GrossAmount
//	GrossAmount = PlatformFeeAmount + BrandFeeAmount + ReceivableAmount
//
// PlatformFeeAmount is AMOL's share.
// BrandFeeAmount is the productBlueprint Brand's share.
// ReceivableAmount is the amount owed to the resale seller.
//
// BrandRevenue identifies the immutable productBlueprint Brand Account and
// Stripe destination captured when the resale Order item was created.
//
// All persistence paths use deterministic IDs and are safe to retry after a
// partial failure or duplicate Stripe webhook.
//
// Only primary-sale Settlement records are returned. Resale seller and Brand
// financial state is persisted through SalesReceivableUsecase and
// BrandFeeSettlementUsecase.
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
	if err := validateSellerFinancialCalculation(order, payment, calculation); err != nil {
		return nil, err
	}
	if len(calculation.Receivables) > 0 {
		if u.salesReceivableUC == nil {
			return nil, ErrSettlementSalesReceivableUsecaseMissing
		}
		if u.brandFeeSettlementUC == nil {
			return nil, ErrSettlementBrandFeeSettlementUsecaseMissing
		}
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
				OrderItemIndex:    allocation.OrderItemIndex,
				ResaleID:          allocation.ResaleID,
				AvatarID:          allocation.Seller.AvatarID,
				UserID:            allocation.Seller.UserID,
				PayoutAccountID:   allocation.Seller.PayoutAccountID,
				MerchandiseAmount: allocation.MerchandiseAmount,
				ShippingAmount:    allocation.ShippingAmount,
				GrossAmount:       allocation.GrossAmount,
				PlatformFeeAmount: allocation.PlatformFeeAmount,
				BrandFeeAmount:    allocation.BrandFeeAmount,
				ReceivableAmount:  allocation.ReceivableAmount,
				Currency:          salesreceivabledom.CurrencyJPY,
			},
		)
		if err != nil {
			return nil, err
		}

		_, err = u.brandFeeSettlementUC.EnsurePending(
			ctx,
			EnsureBrandFeeSettlementInput{
				OrderID:        order.ID,
				PaymentID:      payment.PaymentID,
				OrderItemIndex: allocation.OrderItemIndex,
				ResaleID:       allocation.ResaleID,
				Brand: brandfeesettlementdom.BrandIdentity{
					BrandID:         allocation.BrandRevenue.BrandID,
					CompanyID:       allocation.BrandRevenue.CompanyID,
					AccountID:       allocation.BrandRevenue.AccountID,
					StripeAccountID: allocation.BrandRevenue.StripeAccountID,
				},
				StripePaymentIntentID: payment.StripePaymentIntentID,
				StripeChargeID:        payment.StripeChargeID,
				TransferGroup:         payment.TransferGroup,
				BrandFeeAmount:        allocation.BrandFeeAmount,
				Currency:              brandfeesettlementdom.CurrencyJPY,
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

	settlementID, err := settlementdom.NewID(payment.PaymentID, allocation.Seller)
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
			if err := validateResaleBrandRevenueSnapshot(item.BrandRevenueSnapshot, item.BrandID); err != nil {
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

func validateResaleBrandRevenueSnapshot(
	snapshot orderdom.BrandRevenueSnapshot,
	expectedBrandID string,
) error {
	if expectedBrandID == "" ||
		snapshot.BrandID == "" ||
		snapshot.BrandID != expectedBrandID ||
		snapshot.CompanyID == "" ||
		snapshot.AccountID == "" ||
		snapshot.StripeAccountID == "" {
		return orderdom.ErrInvalidItemSnapshot
	}

	resolved := settlementdom.BrandRevenueIdentity{
		BrandID:         snapshot.BrandID,
		CompanyID:       snapshot.CompanyID,
		AccountID:       snapshot.AccountID,
		StripeAccountID: snapshot.StripeAccountID,
	}
	if err := resolved.Validate(); err != nil {
		return orderdom.ErrInvalidItemSnapshot
	}

	return nil
}

func validateSellerFinancialCalculation(
	order orderdom.Order,
	payment paymentdom.Payment,
	calculation settlementdom.Calculation,
) error {
	if len(calculation.Settlements) == 0 && len(calculation.Receivables) == 0 {
		return ErrSettlementAllocationEmpty
	}

	maxInt := int(^uint(0) >> 1)
	seenSettlements := make(map[string]struct{}, len(calculation.Settlements))
	seenReceivableItems := make(map[int]struct{}, len(calculation.Receivables))
	seenResaleIDs := make(map[string]struct{}, len(calculation.Receivables))
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
		if _, exists := seenSettlements[sellerKey]; exists {
			return ErrSettlementDuplicateSeller
		}
		seenSettlements[sellerKey] = struct{}{}

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
		if allocation.OrderItemIndex < 0 ||
			allocation.OrderItemIndex >= len(order.Items) ||
			allocation.ResaleID == "" ||
			allocation.BrandID == "" {
			return ErrSettlementAllocationInvalid
		}
		if err := allocation.Seller.Validate(); err != nil {
			return ErrSettlementAllocationInvalid
		}
		if err := allocation.BrandRevenue.Validate(); err != nil {
			return ErrSettlementAllocationInvalid
		}
		if allocation.BrandRevenue.BrandID != allocation.BrandID {
			return ErrSettlementAllocationInvalid
		}

		item := order.Items[allocation.OrderItemIndex]
		if item.IsCancelled ||
			item.Type != orderdom.OrderItemTypeResale ||
			item.ResaleID != allocation.ResaleID ||
			item.BrandID == "" ||
			item.BrandID != allocation.BrandID ||
			item.Qty != 1 ||
			item.Price <= 0 {
			return ErrSettlementAllocationInvalid
		}

		if item.SellerSnapshot.AvatarID != allocation.Seller.AvatarID ||
			item.SellerSnapshot.UserID != allocation.Seller.UserID ||
			item.SellerSnapshot.PayoutAccountID != allocation.Seller.PayoutAccountID ||
			item.SellerSnapshot.PayoutAccountID != item.SellerSnapshot.UserID ||
			item.SellerSnapshot.BrandID != "" ||
			item.SellerSnapshot.CompanyID != "" ||
			item.SellerSnapshot.AccountID != "" ||
			item.SellerSnapshot.StripeAccountID != "" {
			return ErrSettlementAllocationInvalid
		}

		if item.BrandRevenueSnapshot.BrandID != allocation.BrandRevenue.BrandID ||
			item.BrandRevenueSnapshot.CompanyID != allocation.BrandRevenue.CompanyID ||
			item.BrandRevenueSnapshot.AccountID != allocation.BrandRevenue.AccountID ||
			item.BrandRevenueSnapshot.StripeAccountID != allocation.BrandRevenue.StripeAccountID ||
			item.BrandRevenueSnapshot.BrandID != item.BrandID {
			return ErrSettlementAllocationInvalid
		}

		if allocation.MerchandiseAmount <= 0 ||
			allocation.ShippingAmount <= 0 ||
			allocation.ShippingAmount >= allocation.MerchandiseAmount ||
			allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.BrandFeeAmount <= 0 ||
			allocation.ReceivableAmount <= 0 ||
			allocation.PlatformFeeAmount > allocation.GrossAmount ||
			allocation.BrandFeeAmount > allocation.GrossAmount ||
			allocation.ReceivableAmount > allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if allocation.MerchandiseAmount != item.Price {
			return ErrSettlementAllocationInvalid
		}
		if allocation.MerchandiseAmount-allocation.ShippingAmount != allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}
		if allocation.PlatformFeeAmount > allocation.GrossAmount-allocation.BrandFeeAmount {
			return ErrSettlementAllocationInvalid
		}
		if allocation.GrossAmount-allocation.PlatformFeeAmount-allocation.BrandFeeAmount != allocation.ReceivableAmount {
			return ErrSettlementAllocationInvalid
		}

		if _, exists := seenReceivableItems[allocation.OrderItemIndex]; exists {
			return ErrSettlementAllocationInvalid
		}
		seenReceivableItems[allocation.OrderItemIndex] = struct{}{}

		if _, exists := seenResaleIDs[allocation.ResaleID]; exists {
			return ErrSettlementAllocationInvalid
		}
		seenResaleIDs[allocation.ResaleID] = struct{}{}

		expectedReceivableID, err := salesreceivabledom.NewID(
			payment.PaymentID,
			allocation.OrderItemIndex,
		)
		if err != nil || expectedReceivableID == "" {
			return ErrSettlementAllocationInvalid
		}

		expectedBrandFeeSettlementID, err := brandfeesettlementdom.NewID(
			payment.PaymentID,
			allocation.OrderItemIndex,
		)
		if err != nil || expectedBrandFeeSettlementID == "" {
			return ErrSettlementAllocationInvalid
		}

		brandIdentity := brandfeesettlementdom.BrandIdentity{
			BrandID:         allocation.BrandRevenue.BrandID,
			CompanyID:       allocation.BrandRevenue.CompanyID,
			AccountID:       allocation.BrandRevenue.AccountID,
			StripeAccountID: allocation.BrandRevenue.StripeAccountID,
		}
		if err := brandIdentity.Validate(); err != nil {
			return ErrSettlementAllocationInvalid
		}

		// Resale shipping is not charged to the buyer.
		// Payment.Amount therefore contains the original resale merchandise
		// amount rather than GrossAmount after shipping deduction.
		if total > maxInt-allocation.MerchandiseAmount {
			return ErrSettlementAllocationAmountMismatch
		}
		total += allocation.MerchandiseAmount
	}

	if total != payment.Amount {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}
