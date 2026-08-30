// backend/internal/application/usecase/settlement_readiness.go
package usecase

import (
	"context"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Settlement readiness
// ============================================================

// MarkReadyByPaymentAndSeller marks one seller-level Settlement as ready after
// the corresponding seller's Order items have crossed the dispatch boundary.
//
// Primary List sales use SellerTypeAccount.
// Resale transactions use SellerTypeAvatar.
//
// Payment success alone must not make a Settlement ready because DispatchDue
// treats ready as eligible for Stripe Transfer.
//
// Idempotency:
//
// - pending -> ready
// - ready -> no-op
// - transferring -> no-op
// - transferred -> no-op
// - failed_retryable -> no-op
// - failed -> no-op
//
// canceled and reversed are terminal refund/reversal states and cannot become
// ready again.
func (u *SettlementUsecase) MarkReadyByPaymentAndSeller(
	ctx context.Context,
	paymentID string,
	seller settlementdom.SellerIdentity,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{}, ErrSettlementRepositoryMissing
	}
	if paymentID == "" {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidPaymentID
	}
	if err := seller.Validate(); err != nil {
		return settlementdom.Settlement{}, err
	}

	settlementID, err := settlementdom.NewID(paymentID, seller)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	current, err := u.repo.GetByID(ctx, settlementID)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	if current.PaymentID != paymentID {
		return settlementdom.Settlement{}, settlementdom.ErrConflict
	}

	currentSeller := current.SellerIdentity()
	if err := currentSeller.Validate(); err != nil {
		return settlementdom.Settlement{}, err
	}
	if currentSeller != seller {
		return settlementdom.Settlement{}, settlementdom.ErrConflict
	}

	switch current.Status {
	case settlementdom.StatusPending:
		readyStatus := settlementdom.StatusReady

		updated, err := u.repo.UpdateByID(
			ctx,
			settlementID,
			settlementdom.UpdateSettlementInput{
				Status: &readyStatus,
			},
		)
		if err != nil {
			return settlementdom.Settlement{}, err
		}

		if updated.PaymentID != paymentID ||
			updated.SellerIdentity() != seller {
			return settlementdom.Settlement{}, settlementdom.ErrConflict
		}

		return updated, nil

	case settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusTransferred,
		settlementdom.StatusFailedRetryable,
		settlementdom.StatusFailed:
		return current, nil

	case settlementdom.StatusCanceled,
		settlementdom.StatusReversed:
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatusTransition

	default:
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatus
	}
}
