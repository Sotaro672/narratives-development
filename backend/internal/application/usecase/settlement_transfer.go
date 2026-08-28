// backend/internal/application/usecase/settlement_transfer.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Stripe Transfer
// ============================================================

// TransferByPaymentAndAccount executes one Account-level Stripe Transfer.
//
// This is useful after dispatch because an Order item already stores
// SellerSnapshot.AccountID.
func (u *SettlementUsecase) TransferByPaymentAndAccount(
	ctx context.Context,
	paymentID string,
	accountID string,
) (settlementdom.Settlement, error) {
	if paymentID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidPaymentID
	}

	if accountID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidAccountID
	}

	settlementID, err := settlementdom.NewID(
		paymentID,
		accountID,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	return u.TransferByID(
		ctx,
		settlementID,
	)
}

// TransferByID executes a Stripe Connect Transfer for one Settlement.
//
// Execution order:
//  1. Atomically claim the Settlement.
//  2. Call Stripe with a deterministic Idempotency-Key.
//  3. Persist StripeTransferID and transferred state.
//  4. If Stripe fails, persist failed_retryable or failed.
//
// The atomic claim prevents two workers from intentionally starting the same
// Settlement at the same time.
//
// The Stripe idempotency key protects against an uncertain network result where
// Stripe may have accepted the request before the worker received the response.
func (u *SettlementUsecase) TransferByID(
	ctx context.Context,
	settlementID string,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{},
			ErrSettlementRepositoryMissing
	}

	if u.stripeTransferGateway == nil {
		return settlementdom.Settlement{},
			ErrSettlementStripeTransferGatewayMissing
	}

	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

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

	claim, err := u.repo.ClaimForTransfer(
		ctx,
		settlementID,
		now,
		staleBefore,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	if !claim.Claimed {
		if claim.Settlement.Status ==
			settlementdom.StatusTransferred {
			return claim.Settlement, nil
		}

		return claim.Settlement,
			ErrSettlementTransferNotReady
	}

	settlement := claim.Settlement

	if settlement.Status !=
		settlementdom.StatusTransferring {
		return settlementdom.Settlement{},
			ErrSettlementTransferNotReady
	}

	if settlement.StripeChargeID == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			false,
			nil,
			nil,
			ErrSettlementStripeChargeIDMissing,
		)
	}

	if settlement.TransferGroup == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			false,
			nil,
			nil,
			ErrSettlementTransferGroupMissing,
		)
	}

	idempotencyKey := fmt.Sprintf(
		"settlement:%s:%s",
		settlement.PaymentID,
		settlement.AccountID,
	)

	stripeResult, stripeErr :=
		u.stripeTransferGateway.CreateTransfer(
			ctx,
			CreateStripeSettlementTransferInput{
				Amount: settlement.TransferAmount,

				Currency: strings.ToLower(
					settlement.Currency,
				),

				DestinationStripeAccountID: settlement.StripeAccountID,
				SourceTransaction:          settlement.StripeChargeID,
				TransferGroup:              settlement.TransferGroup,
				IdempotencyKey:             idempotencyKey,

				OrderID:      settlement.OrderID,
				PaymentID:    settlement.PaymentID,
				SettlementID: settlement.ID,
				CompanyID:    settlement.CompanyID,
				AccountID:    settlement.AccountID,
			},
		)

	if stripeErr != nil {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			isSettlementTransferErrorRetryable(
				stripeErr,
			),
			settlementErrorType(
				stripeErr,
			),
			settlementErrorCode(
				stripeErr,
			),
			stripeErr,
		)
	}

	if stripeResult == nil {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			true,
			nil,
			nil,
			ErrSettlementStripeTransferResultEmpty,
		)
	}

	stripeTransferID :=
		stripeResult.StripeTransferID
	if stripeTransferID == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			true,
			nil,
			nil,
			ErrSettlementStripeTransferIDEmpty,
		)
	}

	completedAt := u.now().UTC()

	completed, err := u.repo.CompleteTransfer(
		ctx,
		settlement.ID,
		stripeTransferID,
		completedAt,
	)
	if err != nil {
		// Stripe may already have completed the Transfer here.
		// Do not attempt another transfer with a different idempotency key.
		// The deterministic key allows reconciliation/retry to recover the same
		// Stripe Transfer.
		return settlementdom.Settlement{},
			fmt.Errorf(
				"settlement: persist completed Stripe Transfer %q: %w",
				stripeTransferID,
				err,
			)
	}

	return completed, nil
}

func (u *SettlementUsecase) failClaimedSettlement(
	ctx context.Context,
	settlement settlementdom.Settlement,
	retryable bool,
	errorType *string,
	errorCode *string,
	cause error,
) (settlementdom.Settlement, error) {
	if cause == nil {
		cause = errors.New(
			"settlement: Stripe transfer failed",
		)
	}

	errorMessage := cause.Error()

	nextStatus := settlementdom.StatusFailed
	if retryable {
		nextStatus = settlementdom.StatusFailedRetryable
	}

	failed, persistErr := u.repo.FailTransfer(
		ctx,
		settlement.ID,
		nextStatus,
		normalizeSettlementErrorString(
			errorType,
		),
		normalizeSettlementErrorString(
			errorCode,
		),
		&errorMessage,
		u.now().UTC(),
	)
	if persistErr != nil {
		return settlementdom.Settlement{},
			fmt.Errorf(
				"settlement: Stripe transfer failed and failure state could not be persisted: transferErr=%v persistErr=%w",
				cause,
				persistErr,
			)
	}

	return failed, cause
}
