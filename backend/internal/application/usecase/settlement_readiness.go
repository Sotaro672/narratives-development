// backend/internal/application/usecase/settlement_readiness.go
package usecase

import (
	"context"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Settlement readiness
// ============================================================

// MarkReadyByPaymentAndAccount marks one Account-level Settlement as ready
// after the corresponding seller's Order items have been dispatched.
//
// Payment success alone must not make a Settlement ready because
// DispatchDue treats ready as eligible for Stripe Transfer.
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
func (u *SettlementUsecase) MarkReadyByPaymentAndAccount(
	ctx context.Context,
	paymentID string,
	accountID string,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{},
			ErrSettlementRepositoryMissing
	}

	if paymentID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidPaymentID
	}

	if accountID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidAccountID
	}

	settlementID, err :=
		settlementdom.NewID(
			paymentID,
			accountID,
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	current, err :=
		u.repo.GetByID(
			ctx,
			settlementID,
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	if current.PaymentID != paymentID ||
		current.AccountID != accountID {
		return settlementdom.Settlement{},
			settlementdom.ErrConflict
	}

	switch current.Status {
	case settlementdom.StatusPending:
		readyStatus :=
			settlementdom.StatusReady

		return u.repo.UpdateByID(
			ctx,
			settlementID,
			settlementdom.UpdateSettlementInput{
				Status: &readyStatus,
			},
		)

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
