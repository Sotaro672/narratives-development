// backend/internal/application/usecase/settlement_create.go
package usecase

import (
	"context"
	"errors"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Settlement creation
// ============================================================

// EnsureForSucceededPayment creates Account-level Settlements for one
// successfully completed Payment.
//
// This method is idempotent because each Settlement ID is deterministic:
//
//	paymentID + "_" + accountID
//
// Existing Settlement records are loaded and verified instead of overwritten.
//
// This method creates Settlements as pending and does not send money.
//
// A Settlement becomes ready only after the corresponding seller Account's
// Order items have been dispatched through MarkReadyByPaymentAndAccount.
//
// Keeping payment success and payout readiness separate prevents reconciliation
// from sending funds for merchandise that has not been dispatched yet.
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

	if err := validateSettlementSource(
		order,
		payment,
	); err != nil {
		return nil, err
	}

	if err := validateSettlementOrderItems(order); err != nil {
		return nil, err
	}

	allocations, err := u.calculator.Calculate(
		ctx,
		order,
		payment,
	)
	if err != nil {
		return nil, err
	}

	if err := validateSettlementAllocations(
		payment,
		allocations,
	); err != nil {
		return nil, err
	}

	now := u.now().UTC()

	result := make(
		[]settlementdom.Settlement,
		0,
		len(allocations),
	)

	for _, allocation := range allocations {
		settlementID, err := settlementdom.NewID(
			payment.PaymentID,
			allocation.AccountID,
		)
		if err != nil {
			return nil, err
		}

		entity, err := settlementdom.New(
			settlementID,
			order.ID,
			payment.PaymentID,
			allocation.CompanyID,
			allocation.AccountID,
			allocation.StripeAccountID,
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
			return nil, err
		}

		created, err := u.repo.Create(
			ctx,
			settlementdom.CreateSettlementInput{
				SettlementID: entity.ID,
				OrderID:      entity.OrderID,
				PaymentID:    entity.PaymentID,

				CompanyID: entity.CompanyID,
				AccountID: entity.AccountID,

				StripeAccountID: entity.StripeAccountID,

				StripePaymentIntentID: entity.StripePaymentIntentID,
				StripeChargeID:        entity.StripeChargeID,

				TransferGroup: entity.TransferGroup,

				GrossAmount:       entity.GrossAmount,
				PlatformFeeAmount: entity.PlatformFeeAmount,
				TransferAmount:    entity.TransferAmount,

				Currency: entity.Currency,
				Status:   entity.Status,
			},
		)
		if err != nil {
			if !errors.Is(
				err,
				settlementdom.ErrConflict,
			) {
				return nil, err
			}

			existing, getErr := u.repo.GetByID(
				ctx,
				settlementID,
			)
			if getErr != nil {
				return nil, getErr
			}

			if err := validateExistingSettlement(
				existing,
				entity,
			); err != nil {
				return nil, err
			}

			result = append(
				result,
				existing,
			)
			continue
		}

		result = append(
			result,
			created,
		)
	}

	return result, nil
}
