// backend/internal/application/usecase/item_refund_capacity.go
package usecase

import (
	"context"

	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Capacity Validation
// ============================================================

func (u *ItemRefundUsecase) validatePaymentRefundCapacity(
	ctx context.Context,
	paymentID string,
	currentRefundID string,
	newRefundAmount int,
	paymentAmount int,
) error {
	if newRefundAmount <= 0 || paymentAmount <= 0 {
		return refunddom.ErrInvalidRefundAmount
	}

	refunds, err := u.refundRepo.ListByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return err
	}

	total := 0

	for _, refund := range refunds {
		if refund.ID == currentRefundID {
			continue
		}

		if !reservesPurchaserRefundAmount(refund) {
			continue
		}

		if refund.RefundAmount > paymentAmount-total {
			return ErrItemRefundPaymentAmountExceeded
		}

		total += refund.RefundAmount
	}

	if newRefundAmount > paymentAmount-total {
		return ErrItemRefundPaymentAmountExceeded
	}

	return nil
}

func (u *ItemRefundUsecase) validateTransferReversalCapacity(
	ctx context.Context,
	settlement settlementdom.Settlement,
	currentRefundID string,
	newReversalAmount int,
) error {
	if newReversalAmount < 0 {
		return refunddom.ErrInvalidTransferReversalAmount
	}

	if newReversalAmount > settlement.TransferAmount {
		return ErrItemRefundTransferReversalAmountExceeded
	}

	refunds, err := u.refundRepo.ListBySettlementID(
		ctx,
		settlement.ID,
	)
	if err != nil {
		return err
	}

	total := 0

	for _, refund := range refunds {
		if refund.ID == currentRefundID {
			continue
		}

		if !reservesTransferReversalAmount(refund) {
			continue
		}

		if refund.TransferReversalAmount >
			settlement.TransferAmount-total {
			return ErrItemRefundTransferReversalAmountExceeded
		}

		total += refund.TransferReversalAmount
	}

	if newReversalAmount > settlement.TransferAmount-total {
		return ErrItemRefundTransferReversalAmountExceeded
	}

	return nil
}

func reservesPurchaserRefundAmount(
	refund refunddom.Refund,
) bool {
	switch refund.Status {
	case refunddom.StatusCreated,
		refunddom.StatusPending,
		refunddom.StatusRequiresAction,
		refunddom.StatusSucceeded:
		return true

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return false

	default:
		return true
	}
}

func reservesTransferReversalAmount(
	refund refunddom.Refund,
) bool {
	if refund.TransferReversalAmount <= 0 {
		return false
	}

	switch refund.Status {
	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return false

	default:
		return true
	}
}
