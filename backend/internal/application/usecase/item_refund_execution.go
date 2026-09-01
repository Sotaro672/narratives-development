// backend/internal/application/usecase/item_refund_execution.go
package usecase

import (
	"context"
	"errors"
	"strings"

	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
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
	amountSummary, err := calculateItemRefundAmount(order, in.ItemIndex, in.Policy)
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
	if payment.StripeChargeID == "" || !strings.HasPrefix(payment.StripeChargeID, "ch_") {
		return refunddom.Refund{}, ErrItemRefundStripeChargeMissing
	}

	refundID, err := refunddom.NewID(order.ID, in.ItemIndex)
	if err != nil {
		return refunddom.Refund{}, err
	}

	var refundSeller refunddom.SellerIdentity
	var settlement *settlementdom.Settlement
	var salesReceivable *salesreceivabledom.SalesReceivable
	settlementID := ""
	salesReceivableID := ""

	switch targetItem.Type {
	case orderdom.OrderItemTypeList:
		if in.CompanyID == "" {
			return refunddom.Refund{}, ErrItemRefundInvalidCompanyID
		}
		if targetItem.SellerSnapshot.CompanyID != in.CompanyID {
			return refunddom.Refund{}, ErrItemRefundCompanyMismatch
		}
		if targetItem.SellerSnapshot.AccountID == "" {
			return refunddom.Refund{}, ErrItemRefundAccountMissing
		}

		refundSeller = refunddom.SellerIdentity{
			Type:            refunddom.SellerTypeAccount,
			CompanyID:       targetItem.SellerSnapshot.CompanyID,
			AccountID:       targetItem.SellerSnapshot.AccountID,
			StripeAccountID: targetItem.SellerSnapshot.StripeAccountID,
		}
		if err := refundSeller.Validate(); err != nil {
			return refunddom.Refund{}, err
		}

		resolvedSettlement, err := u.resolveSettlement(ctx, *payment, targetItem)
		if err != nil {
			return refunddom.Refund{}, err
		}
		settlement = &resolvedSettlement
		settlementID = resolvedSettlement.ID

	case orderdom.OrderItemTypeResale:
		refundSeller, err = itemRefundResaleSellerIdentity(targetItem)
		if err != nil {
			return refunddom.Refund{}, err
		}

		resolvedReceivable, err := u.resolveAndCancelResaleReceivable(
			ctx,
			*payment,
			targetItem,
			in.ItemIndex,
		)
		if err != nil {
			return refunddom.Refund{}, err
		}
		salesReceivable = resolvedReceivable
		salesReceivableID = resolvedReceivable.ID

		if err := u.prepareResaleBrandFeeSettlementRefund(ctx, *payment, targetItem, in.ItemIndex); err != nil {
			return refunddom.Refund{}, err
		}

	default:
		return refunddom.Refund{}, ErrItemRefundOrderMismatch
	}

	existingRefund, err := u.refundRepo.GetByID(ctx, refundID)
	if err == nil {
		if existingRefund == nil {
			return refunddom.Refund{}, refunddom.ErrConflict
		}

		if err := validateExistingItemRefund(
			*existingRefund,
			in,
			order,
			targetItem,
			settlement,
			salesReceivable,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(ctx, *payment, targetItem, settlement, *existingRefund)
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

	transferReversalAmount := 0
	if settlement != nil {
		transferReversalAmount, err = u.resolveTransferReversalAmount(
			ctx,
			*settlement,
			targetItem,
			amountSummary,
		)
		if err != nil {
			return refunddom.Refund{}, err
		}

		if err := u.validateTransferReversalCapacity(
			ctx,
			*settlement,
			refundID,
			transferReversalAmount,
		); err != nil {
			return refunddom.Refund{}, err
		}
	}

	createdRefund, err := u.refundRepo.Create(
		ctx,
		refunddom.CreateRefundInput{
			RefundID:                  refundID,
			InquiryID:                 in.InquiryID,
			OrderID:                   order.ID,
			PaymentID:                 payment.PaymentID,
			OrderItemIndex:            in.ItemIndex,
			Seller:                    refundSeller,
			SettlementID:              settlementID,
			SalesReceivableID:         salesReceivableID,
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

		existing, getErr := u.refundRepo.GetByID(ctx, refundID)
		if getErr != nil {
			return refunddom.Refund{}, getErr
		}
		if existing == nil {
			return refunddom.Refund{}, refunddom.ErrConflict
		}

		if err := validateExistingItemRefund(
			*existing,
			in,
			order,
			targetItem,
			settlement,
			salesReceivable,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(ctx, *payment, targetItem, settlement, *existing)
	}

	if createdRefund == nil {
		return refunddom.Refund{}, refunddom.ErrConflict
	}

	return u.resumeRefund(ctx, *payment, targetItem, settlement, *createdRefund)
}

// ============================================================
// Resale Financial State
// ============================================================

func itemRefundResaleSellerIdentity(
	targetItem orderdom.OrderItemSnapshot,
) (refunddom.SellerIdentity, error) {
	if targetItem.Type != orderdom.OrderItemTypeResale ||
		targetItem.ResaleID == "" ||
		targetItem.Qty != 1 ||
		targetItem.Price <= 0 {
		return refunddom.SellerIdentity{}, ErrItemRefundSalesReceivableMismatch
	}

	snapshot := targetItem.SellerSnapshot
	if snapshot.AvatarID == "" ||
		snapshot.UserID == "" ||
		snapshot.PayoutAccountID == "" ||
		snapshot.PayoutAccountID != snapshot.UserID ||
		snapshot.BrandID != "" ||
		snapshot.CompanyID != "" ||
		snapshot.AccountID != "" ||
		snapshot.StripeAccountID != "" {
		return refunddom.SellerIdentity{}, ErrItemRefundSalesReceivableMismatch
	}

	seller := refunddom.SellerIdentity{
		Type:            refunddom.SellerTypeResale,
		AvatarID:        snapshot.AvatarID,
		UserID:          snapshot.UserID,
		PayoutAccountID: snapshot.PayoutAccountID,
	}
	if err := seller.Validate(); err != nil {
		return refunddom.SellerIdentity{}, ErrItemRefundSalesReceivableMismatch
	}

	return seller, nil
}

// resolveAndCancelResaleReceivable resolves the exact item-level receivable and
// ensures seller proceeds can no longer be paid before the purchaser Stripe
// Refund is created.
//
// Cancellation is deliberately performed before Refund creation / Stripe Refund.
// If a later step fails, retrying this operation sees StatusCanceled and safely
// continues.
//
// reserved and paid are rejected. They require BankPayout coordination and,
// after payment, a separate recovery/adjustment model.
func (u *ItemRefundUsecase) resolveAndCancelResaleReceivable(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) (*salesreceivabledom.SalesReceivable, error) {
	if u == nil || u.salesReceivableService == nil {
		return nil, ErrItemRefundSalesReceivableNotConfigured
	}
	if payment.PaymentID == "" || itemIndex < 0 {
		return nil, ErrItemRefundSalesReceivableMismatch
	}
	if _, err := itemRefundResaleSellerIdentity(targetItem); err != nil {
		return nil, err
	}

	receivableID, err := salesreceivabledom.NewID(payment.PaymentID, itemIndex)
	if err != nil {
		return nil, ErrItemRefundSalesReceivableMismatch
	}

	current, err := u.salesReceivableService.GetByID(ctx, receivableID)
	if err != nil {
		if errors.Is(err, salesreceivabledom.ErrNotFound) {
			return nil, ErrItemRefundSalesReceivableNotFound
		}
		return nil, err
	}
	if current == nil {
		return nil, ErrItemRefundSalesReceivableNotFound
	}
	if err := validateResaleReceivableRefundTarget(
		*current,
		payment,
		targetItem,
		itemIndex,
	); err != nil {
		return nil, err
	}

	switch current.Status {
	case salesreceivabledom.StatusPending,
		salesreceivabledom.StatusAvailable:
		canceled, err := u.salesReceivableService.Cancel(ctx, receivableID)
		if err != nil {
			if errors.Is(err, ErrSalesReceivableCannotCancel) {
				return u.resolveResaleReceivableCancellationRace(
					ctx,
					receivableID,
					payment,
					targetItem,
					itemIndex,
				)
			}
			return nil, err
		}
		if canceled == nil {
			return nil, ErrItemRefundSalesReceivableUnavailable
		}
		if err := validateResaleReceivableRefundTarget(
			*canceled,
			payment,
			targetItem,
			itemIndex,
		); err != nil {
			return nil, err
		}
		if canceled.Status != salesreceivabledom.StatusCanceled ||
			canceled.BankPayoutID != "" {
			return nil, ErrItemRefundSalesReceivableUnavailable
		}

		return canceled, nil

	case salesreceivabledom.StatusCanceled:
		if current.BankPayoutID != "" {
			return nil, ErrItemRefundSalesReceivableMismatch
		}

		return current, nil

	case salesreceivabledom.StatusReserved:
		return nil, ErrItemRefundSalesReceivableReserved

	case salesreceivabledom.StatusPaid:
		return nil, ErrItemRefundSalesReceivablePaid

	default:
		return nil, ErrItemRefundSalesReceivableUnavailable
	}
}

func (u *ItemRefundUsecase) resolveResaleReceivableCancellationRace(
	ctx context.Context,
	receivableID string,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) (*salesreceivabledom.SalesReceivable, error) {
	current, err := u.salesReceivableService.GetByID(ctx, receivableID)
	if err != nil {
		if errors.Is(err, salesreceivabledom.ErrNotFound) {
			return nil, ErrItemRefundSalesReceivableNotFound
		}
		return nil, err
	}
	if current == nil {
		return nil, ErrItemRefundSalesReceivableNotFound
	}
	if err := validateResaleReceivableRefundTarget(
		*current,
		payment,
		targetItem,
		itemIndex,
	); err != nil {
		return nil, err
	}

	switch current.Status {
	case salesreceivabledom.StatusCanceled:
		return current, nil

	case salesreceivabledom.StatusReserved:
		return nil, ErrItemRefundSalesReceivableReserved

	case salesreceivabledom.StatusPaid:
		return nil, ErrItemRefundSalesReceivablePaid

	default:
		return nil, ErrItemRefundSalesReceivableUnavailable
	}
}

func validateResaleReceivableRefundTarget(
	receivable salesreceivabledom.SalesReceivable,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) error {
	if err := receivable.Validate(); err != nil {
		return ErrItemRefundSalesReceivableMismatch
	}

	expectedID, err := salesreceivabledom.NewID(payment.PaymentID, itemIndex)
	if err != nil {
		return ErrItemRefundSalesReceivableMismatch
	}

	snapshot := targetItem.SellerSnapshot
	if targetItem.Type != orderdom.OrderItemTypeResale ||
		targetItem.ResaleID == "" ||
		targetItem.Qty != 1 ||
		targetItem.Price <= 0 ||
		receivable.ID != expectedID ||
		receivable.OrderID != payment.PaymentID ||
		receivable.PaymentID != payment.PaymentID ||
		receivable.OrderItemIndex != itemIndex ||
		receivable.ResaleID != targetItem.ResaleID ||
		receivable.AvatarID != snapshot.AvatarID ||
		receivable.UserID != snapshot.UserID ||
		receivable.PayoutAccountID != snapshot.PayoutAccountID ||
		receivable.PayoutAccountID != receivable.UserID ||
		receivable.MerchandiseAmount != targetItem.Price ||
		receivable.Currency != salesreceivabledom.CurrencyJPY {
		return ErrItemRefundSalesReceivableMismatch
	}

	return nil
}

// ============================================================
// Resale Brand Fee Refund
// ============================================================

func (u *ItemRefundUsecase) prepareResaleBrandFeeSettlementRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) error {
	if u == nil || u.brandFeeSettlementRefundService == nil {
		return ErrItemRefundNotConfigured
	}

	result, err := u.brandFeeSettlementRefundService.PrepareByPaymentAndOrderItem(ctx, payment.PaymentID, itemIndex)
	if err != nil {
		return err
	}
	if err := validateResaleBrandFeeSettlementRefundResult(result, payment, targetItem, itemIndex); err != nil {
		return err
	}

	switch result.Status {
	case brandfeesettlementdom.StatusCanceled:
		if result.Action != BrandFeeSettlementRefundActionCanceled &&
			result.Action != BrandFeeSettlementRefundActionAlreadyCanceled {
			return ErrBrandFeeSettlementRefundMismatch
		}

	case brandfeesettlementdom.StatusTransferred:
		if result.Action != BrandFeeSettlementRefundActionReversalRequired {
			return ErrBrandFeeSettlementRefundMismatch
		}

	case brandfeesettlementdom.StatusFailed:
		if result.Action != BrandFeeSettlementRefundActionNoTransfer {
			return ErrBrandFeeSettlementRefundMismatch
		}

	case brandfeesettlementdom.StatusReversed:
		if result.Action != BrandFeeSettlementRefundActionAlreadyReversed {
			return ErrBrandFeeSettlementRefundMismatch
		}

	default:
		return ErrBrandFeeSettlementRefundStatusUnsupported
	}

	return nil
}

func (u *ItemRefundUsecase) completeResaleBrandFeeSettlementRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) error {
	if u == nil || u.brandFeeSettlementRefundService == nil {
		return ErrItemRefundNotConfigured
	}

	result, err := u.brandFeeSettlementRefundService.CompleteByPaymentAndOrderItem(ctx, payment.PaymentID, itemIndex)
	if err != nil {
		return err
	}
	if err := validateResaleBrandFeeSettlementRefundResult(result, payment, targetItem, itemIndex); err != nil {
		return err
	}

	switch result.Status {
	case brandfeesettlementdom.StatusCanceled:
		if result.Action != BrandFeeSettlementRefundActionCanceled &&
			result.Action != BrandFeeSettlementRefundActionAlreadyCanceled {
			return ErrBrandFeeSettlementRefundMismatch
		}

	case brandfeesettlementdom.StatusReversed:
		if result.Action != BrandFeeSettlementRefundActionReversed &&
			result.Action != BrandFeeSettlementRefundActionAlreadyReversed {
			return ErrBrandFeeSettlementRefundMismatch
		}

	case brandfeesettlementdom.StatusFailed:
		if result.Action != BrandFeeSettlementRefundActionNoTransfer {
			return ErrBrandFeeSettlementRefundMismatch
		}

	default:
		return ErrBrandFeeSettlementRefundStatusUnsupported
	}

	return nil
}

func validateResaleBrandFeeSettlementRefundResult(
	result BrandFeeSettlementRefundResult,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	itemIndex int,
) error {
	if payment.PaymentID == "" ||
		itemIndex < 0 ||
		targetItem.Type != orderdom.OrderItemTypeResale ||
		targetItem.ResaleID == "" {
		return ErrBrandFeeSettlementRefundMismatch
	}

	expectedID, err := brandfeesettlementdom.NewID(payment.PaymentID, itemIndex)
	if err != nil {
		return ErrBrandFeeSettlementRefundMismatch
	}

	brand := targetItem.BrandRevenueSnapshot
	if brand.BrandID == "" ||
		brand.CompanyID == "" ||
		brand.AccountID == "" ||
		brand.StripeAccountID == "" {
		return ErrBrandFeeSettlementRefundMismatch
	}
	if targetItem.BrandID != "" && targetItem.BrandID != brand.BrandID {
		return ErrBrandFeeSettlementRefundMismatch
	}

	if result.BrandFeeSettlementID != expectedID ||
		result.OrderID != payment.PaymentID ||
		result.PaymentID != payment.PaymentID ||
		result.OrderItemIndex != itemIndex ||
		result.ResaleID != targetItem.ResaleID ||
		result.BrandID != brand.BrandID ||
		result.CompanyID != brand.CompanyID ||
		result.AccountID != brand.AccountID ||
		result.StripeAccountID != brand.StripeAccountID ||
		result.BrandFeeAmount <= 0 ||
		result.Currency != brandfeesettlementdom.CurrencyJPY {
		return ErrBrandFeeSettlementRefundMismatch
	}

	return nil
}

// ============================================================
// Resume
// ============================================================

func (u *ItemRefundUsecase) resumeRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
	settlement *settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if err := refund.Validate(); err != nil {
		return refunddom.Refund{}, err
	}

	switch refund.Status {
	case refunddom.StatusCreated:
		updated, err := u.createPurchaserRefund(ctx, payment, refund)
		if err != nil {
			return updated, err
		}
		refund = updated

	case refunddom.StatusPending,
		refunddom.StatusRequiresAction:
		// Stripe accepted the purchaser Refund, but completion has not yet been
		// confirmed. Seller-side post-refund processing must not run yet.
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

	switch refund.SellerType {
	case refunddom.SellerTypeResale:
		if settlement != nil ||
			refund.SettlementID != "" ||
			refund.SalesReceivableID == "" ||
			refund.TransferReversalAmount != 0 ||
			refund.TransferReversalStatus != refunddom.TransferReversalStatusNotRequired {
			return refund, ErrItemRefundExistingRefundMismatch
		}

		if err := u.completeResaleBrandFeeSettlementRefund(
			ctx,
			payment,
			targetItem,
			refund.OrderItemIndex,
		); err != nil {
			return refund, err
		}

		return refund, nil

	case refunddom.SellerTypeAccount:
		if settlement == nil ||
			refund.SettlementID == "" ||
			refund.SettlementID != settlement.ID ||
			refund.SalesReceivableID != "" {
			return refund, ErrItemRefundSettlementMismatch
		}

		return u.completeTransferReversal(
			ctx,
			payment,
			*settlement,
			refund,
		)

	default:
		return refund, refunddom.ErrInvalidSellerType
	}
}
