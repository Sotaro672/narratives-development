// backend/internal/application/usecase/settlement_query.go
package usecase

import (
	"context"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Queries
// ============================================================

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

	return u.repo.GetByID(
		ctx,
		settlementID,
	)
}

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

	return u.repo.ListByPaymentID(
		ctx,
		paymentID,
	)
}

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

	return u.repo.ListByOrderID(
		ctx,
		orderID,
	)
}

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

	return u.repo.ListByCompanyID(
		ctx,
		companyID,
	)
}

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

	return u.repo.ListByAccountID(
		ctx,
		accountID,
	)
}

func (u *SettlementUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	if avatarID == "" {
		return nil, settlementdom.ErrInvalidAvatarID
	}

	return u.repo.ListByAvatarID(
		ctx,
		avatarID,
	)
}

func (u *SettlementUsecase) ListByPayoutAccountID(
	ctx context.Context,
	payoutAccountID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	if payoutAccountID == "" {
		return nil, settlementdom.ErrInvalidPayoutAccountID
	}

	return u.repo.ListByPayoutAccountID(
		ctx,
		payoutAccountID,
	)
}
