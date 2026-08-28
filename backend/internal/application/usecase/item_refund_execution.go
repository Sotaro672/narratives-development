// backend/internal/application/usecase/item_refund_execution.go
package usecase

import (
	"context"
	"errors"
	"strings"

	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

func (u *ItemRefundUsecase) refundOrderItem(
	ctx context.Context,
	in itemRefundRequest,
) (refunddom.Refund, error) {
	if err := u.validateConfigured(); err != nil {
		return refunddom.Refund{}, err
	}

	if in.InquiryID == "" {
		return refunddom.Refund{}, ErrItemRefundInvalidInquiryID
	}

	if in.OrderID == "" || in.ItemIndex < 0 {
		return refunddom.Refund{}, ErrItemRefundOrderMismatch
	}

	if in.CompanyID == "" {
		return refunddom.Refund{}, ErrItemRefundInvalidCompanyID
	}

	if in.Policy != "" {
		if err := refunddom.ValidateOpenedReturnRefundPolicy(in.Policy); err != nil {
			return refunddom.Refund{}, err
		}
	}

	order, err := u.orderReader.GetByID(ctx, in.OrderID)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if order.ID != in.OrderID || in.ItemIndex >= len(order.Items) {
		return refunddom.Refund{}, ErrItemRefundOrderMismatch
	}

	targetItem := order.Items[in.ItemIndex]

	if targetItem.SellerSnapshot.CompanyID != in.CompanyID {
		return refunddom.Refund{}, ErrItemRefundCompanyMismatch
	}

	if targetItem.SellerSnapshot.AccountID == "" {
		return refunddom.Refund{}, ErrItemRefundAccountMissing
	}

	amountSummary, err := calculateItemRefundAmount(
		order,
		in.ItemIndex,
		in.Policy,
	)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if amountSummary.RefundAmount <= 0 {
		return refunddom.Refund{}, refunddom.ErrInvalidRefundAmount
	}

	payment, err := u.paymentReader.GetByPaymentID(ctx, order.ID)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if payment == nil || payment.PaymentID != order.ID {
		return refunddom.Refund{}, paymentdom.ErrNotFound
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return refunddom.Refund{}, ErrItemRefundPaymentNotSucceeded
	}

	if payment.StripeChargeID == "" ||
		!strings.HasPrefix(payment.StripeChargeID, "ch_") {
		return refunddom.Refund{}, ErrItemRefundStripeChargeMissing
	}

	settlement, err := u.resolveSettlement(
		ctx,
		*payment,
		targetItem,
	)
	if err != nil {
		return refunddom.Refund{}, err
	}

	refundID, err := refunddom.NewID(
		order.ID,
		in.ItemIndex,
	)
	if err != nil {
		return refunddom.Refund{}, err
	}

	existingRefund, err := u.refundRepo.GetByID(
		ctx,
		refundID,
	)
	if err == nil {
		if err := validateExistingItemRefund(
			*existingRefund,
			in,
			order,
			targetItem,
			settlement,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(
			ctx,
			*payment,
			settlement,
			*existingRefund,
		)
	}

	if !errors.Is(err, refunddom.ErrNotFound) {
		return refunddom.Refund{}, err
	}

	if payment.RefundStatus != paymentdom.RefundStatusNone {
		return refunddom.Refund{}, ErrItemRefundPaymentAlreadyRefunding
	}

	if err := u.validatePaymentRefundCapacity(
		ctx,
		payment.PaymentID,
		refundID,
		amountSummary.RefundAmount,
		payment.Amount,
	); err != nil {
		return refunddom.Refund{}, err
	}

	transferReversalAmount, err := u.resolveTransferReversalAmount(
		ctx,
		settlement,
		targetItem,
		amountSummary,
	)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if err := u.validateTransferReversalCapacity(
		ctx,
		settlement,
		refundID,
		transferReversalAmount,
	); err != nil {
		return refunddom.Refund{}, err
	}

	createdRefund, err := u.refundRepo.Create(
		ctx,
		refunddom.CreateRefundInput{
			RefundID:                  refundID,
			InquiryID:                 in.InquiryID,
			OrderID:                   order.ID,
			PaymentID:                 payment.PaymentID,
			OrderItemIndex:            in.ItemIndex,
			CompanyID:                 targetItem.SellerSnapshot.CompanyID,
			AccountID:                 targetItem.SellerSnapshot.AccountID,
			SettlementID:              settlement.ID,
			Policy:                    amountSummary.Policy,
			MerchandiseAmount:         amountSummary.MerchandiseAmount,
			MerchandiseTaxAmount:      amountSummary.MerchandiseTaxAmount,
			OutboundShippingAmount:    amountSummary.OutboundShippingAmount,
			OutboundShippingTaxAmount: amountSummary.OutboundShippingTaxAmount,
			ReturnShippingAmount:      amountSummary.ReturnShippingAmount,
			ReturnShippingTaxAmount:   amountSummary.ReturnShippingTaxAmount,
			TransferReversalAmount:    transferReversalAmount,
			Currency:                  refunddom.CurrencyJPY,
			CreatedAt:                 u.nowUTC(),
		},
	)
	if err != nil {
		if !isRefundCreateConflict(err) {
			return refunddom.Refund{}, err
		}

		existing, getErr := u.refundRepo.GetByID(
			ctx,
			refundID,
		)
		if getErr != nil {
			return refunddom.Refund{}, err
		}

		if err := validateExistingItemRefund(
			*existing,
			in,
			order,
			targetItem,
			settlement,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(
			ctx,
			*payment,
			settlement,
			*existing,
		)
	}

	if createdRefund == nil {
		return refunddom.Refund{}, refunddom.ErrConflict
	}

	return u.resumeRefund(
		ctx,
		*payment,
		settlement,
		*createdRefund,
	)
}

// ============================================================
// Resume
// ============================================================

func (u *ItemRefundUsecase) resumeRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	settlement settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if err := refund.Validate(); err != nil {
		return refunddom.Refund{}, err
	}

	switch refund.Status {
	case refunddom.StatusCreated:
		updated, err := u.createPurchaserRefund(
			ctx,
			payment,
			refund,
		)
		if err != nil {
			return updated, err
		}

		refund = updated

	case refunddom.StatusPending,
		refunddom.StatusRequiresAction:
		// Stripe accepted the Refund, but purchaser-side completion has not
		// been confirmed yet.
		//
		// Do not execute seller Transfer Reversal.
		return refund, nil

	case refunddom.StatusSucceeded:
		// Continue below.

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return refund, ErrItemRefundStripeRefundTerminal

	default:
		return refund, refunddom.ErrInvalidStatus
	}

	if refund.Status != refunddom.StatusSucceeded {
		return refund, nil
	}

	return u.completeTransferReversal(
		ctx,
		payment,
		settlement,
		refund,
	)
}
