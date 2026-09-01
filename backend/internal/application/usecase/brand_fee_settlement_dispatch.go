// backend/internal/application/usecase/brand_fee_settlement_dispatch.go
package usecase

import (
	"context"
	"errors"
	"fmt"

	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
)

// ============================================================
// Transfer Queue Port
// ============================================================

// BrandFeeSettlementTransferQueue is the outbound queue contract used to
// dispatch productBlueprint Brand fee Stripe Transfers.
//
// The queue payload must contain only BrandFeeSettlementID. Brand identity,
// BrandFeeAmount, Stripe destination, Charge ID, TransferGroup, and all other
// financial fields must be loaded again from the authoritative
// BrandFeeSettlement document by the worker.
type BrandFeeSettlementTransferQueue interface {
	EnqueueBrandFeeSettlementTransfer(
		ctx context.Context,
		brandFeeSettlementID string,
	) error
}

// ============================================================
// Errors
// ============================================================

var (
	ErrBrandFeeSettlementTransferQueueMissing = errors.New("brandFeeSettlement: transfer queue is not configured")
)

// ============================================================
// Dispatch configuration
// ============================================================

const (
	defaultBrandFeeSettlementTransferDispatchLimit = 50
	maxBrandFeeSettlementTransferDispatchLimit     = 200
)

// ============================================================
// Brand fee reconciliation
// ============================================================

// DispatchDue enqueues BrandFeeSettlement transfer candidates that may require
// initial execution or recovery.
//
// Candidates are selected by BrandFeeSettlementRepository:
//
//   - ready
//   - failed_retryable
//   - stale transferring
//
// pending records are intentionally excluded. A Brand fee must cross the
// resale fulfillment boundary and become ready before Stripe Transfer may be
// attempted.
//
// A stale transferring BrandFeeSettlement is safe to enqueue again because
// TransferByID uses both an atomic repository claim and a deterministic Stripe
// Idempotency-Key:
//
//	brand_fee_settlement:{BrandFeeSettlementID}
//
// Individual enqueue failures do not stop processing of the remaining
// candidates. The first error is returned after every candidate has been
// attempted.
func (u *BrandFeeSettlementTransferUsecase) DispatchDue(
	ctx context.Context,
	queue BrandFeeSettlementTransferQueue,
	limit int,
) (int, error) {
	if u == nil || u.repo == nil {
		return 0, ErrBrandFeeSettlementRepositoryMissing
	}
	if queue == nil {
		return 0, ErrBrandFeeSettlementTransferQueueMissing
	}
	if u.now == nil {
		return 0, ErrBrandFeeSettlementClockMissing
	}

	limit = normalizeBrandFeeSettlementTransferDispatchLimit(limit)
	now := u.now().UTC()

	transferLease := u.transferLease
	if transferLease <= 0 {
		transferLease = defaultBrandFeeSettlementTransferLease
	}
	staleBefore := now.Add(-transferLease)

	candidates, err := u.repo.ListTransferCandidates(
		ctx,
		brandfeesettlementdom.ListTransferCandidatesInput{
			StaleBefore: staleBefore,
			Limit:       limit,
		},
	)
	if err != nil {
		return 0, fmt.Errorf(
			"brandFeeSettlement: list transfer candidates: %w",
			err,
		)
	}

	enqueuedCount := 0
	var firstErr error

	for _, candidate := range candidates {
		if candidate.ID == "" {
			if firstErr == nil {
				firstErr = brandfeesettlementdom.ErrInvalidID
			}
			continue
		}

		if err := candidate.Validate(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"brandFeeSettlement: invalid transfer candidate %q: %w",
					candidate.ID,
					err,
				)
			}
			continue
		}

		switch candidate.Status {
		case brandfeesettlementdom.StatusReady,
			brandfeesettlementdom.StatusFailedRetryable:

		case brandfeesettlementdom.StatusTransferring:
			if candidate.UpdatedAt.After(staleBefore) {
				continue
			}

		default:
			continue
		}

		if err := queue.EnqueueBrandFeeSettlementTransfer(
			ctx,
			candidate.ID,
		); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"brandFeeSettlement: enqueue transfer candidate %q: %w",
					candidate.ID,
					err,
				)
			}
			continue
		}

		enqueuedCount++
	}

	return enqueuedCount, firstErr
}

func normalizeBrandFeeSettlementTransferDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultBrandFeeSettlementTransferDispatchLimit
	}
	if limit > maxBrandFeeSettlementTransferDispatchLimit {
		return maxBrandFeeSettlementTransferDispatchLimit
	}

	return limit
}
