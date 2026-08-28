// backend/internal/application/usecase/settlement_reconciliation.go
package usecase

import (
	"context"
	"fmt"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Settlement reconciliation
// ============================================================

// DispatchDue enqueues Settlement transfer candidates that may have lost their
// original Cloud Task or require recovery.
//
// Candidates are selected by the repository:
//
// - ready
// - failed_retryable
// - stale transferring
//
// A stale transferring Settlement is safe to enqueue again because
// TransferByID uses both an atomic Firestore claim and a deterministic Stripe
// Idempotency-Key.
//
// Individual enqueue failures do not stop the remaining candidates. The first
// error is returned after all candidates have been attempted.
func (u *SettlementUsecase) DispatchDue(
	ctx context.Context,
	queue SettlementTransferQueue,
	limit int,
) (int, error) {
	if u == nil || u.repo == nil {
		return 0,
			ErrSettlementRepositoryMissing
	}

	if queue == nil {
		return 0,
			ErrSettlementTransferQueueMissing
	}

	limit =
		normalizeSettlementTransferDispatchLimit(
			limit,
		)

	now := u.now().UTC()

	transferLease := u.transferLease
	if transferLease <= 0 {
		transferLease =
			defaultSettlementTransferLease
	}

	staleBefore :=
		now.Add(
			-transferLease,
		)

	candidates, err :=
		u.repo.ListTransferCandidates(
			ctx,
			settlementdom.ListTransferCandidatesInput{
				StaleBefore: staleBefore,
				Limit:       limit,
			},
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"settlement: list transfer candidates: %w",
				err,
			)
	}

	enqueuedCount := 0
	var firstErr error

	for _, candidate := range candidates {
		if candidate.ID == "" {
			if firstErr == nil {
				firstErr =
					settlementdom.ErrInvalidID
			}

			continue
		}

		switch candidate.Status {
		case settlementdom.StatusReady,
			settlementdom.StatusFailedRetryable:

		case settlementdom.StatusTransferring:
			if candidate.UpdatedAt.After(
				staleBefore,
			) {
				continue
			}

		default:
			continue
		}

		if err :=
			queue.EnqueueSettlementTransfer(
				ctx,
				candidate.ID,
			); err != nil {
			if firstErr == nil {
				firstErr =
					fmt.Errorf(
						"settlement: enqueue transfer candidate %q: %w",
						candidate.ID,
						err,
					)
			}

			continue
		}

		enqueuedCount++
	}

	return enqueuedCount,
		firstErr
}

func normalizeSettlementTransferDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultSettlementTransferDispatchLimit
	}

	if limit >
		maxSettlementTransferDispatchLimit {
		return maxSettlementTransferDispatchLimit
	}

	return limit
}
