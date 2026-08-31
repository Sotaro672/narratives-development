// backend/internal/application/usecase/settlement_query.go
package usecase

import (
	"context"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Queries
// ============================================================

// GetByID returns one primary-sale Stripe Settlement.
func (u *SettlementUsecase) GetByID(
	ctx context.Context,
	settlementID string,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{}, ErrSettlementRepositoryMissing
	}
	if settlementID == "" {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidID
	}

	return u.repo.GetByID(ctx, settlementID)
}

// ListByPaymentID returns every primary-sale Settlement belonging to one
// succeeded buyer Payment.
func (u *SettlementUsecase) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}
	if paymentID == "" {
		return nil, settlementdom.ErrInvalidPaymentID
	}

	return u.repo.ListByPaymentID(ctx, paymentID)
}

// ListByOrderID returns every primary-sale Settlement belonging to one Order.
//
// PaymentID currently equals OrderID, but the Order boundary remains explicit
// for application and query use.
func (u *SettlementUsecase) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}
	if orderID == "" {
		return nil, settlementdom.ErrInvalidOrderID
	}

	return u.repo.ListByOrderID(ctx, orderID)
}

// ListByCompanyID returns primary-sale Settlements attributable to one Company.
func (u *SettlementUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}
	if companyID == "" {
		return nil, settlementdom.ErrInvalidCompanyID
	}

	return u.repo.ListByCompanyID(ctx, companyID)
}

// ListByAccountID returns primary-sale Settlements attributable to one Stripe
// Connected Account payout identity.
func (u *SettlementUsecase) ListByAccountID(
	ctx context.Context,
	accountID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}
	if accountID == "" {
		return nil, settlementdom.ErrInvalidAccountID
	}

	return u.repo.ListByAccountID(ctx, accountID)
}
