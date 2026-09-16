// backend/internal/domain/refund/errors.go
package refund

import "errors"

var (
	ErrInvalidID = errors.New(
		"refund: invalid id",
	)
	ErrInvalidInquiryID = errors.New(
		"refund: invalid inquiryId",
	)
	ErrInvalidOrderID = errors.New(
		"refund: invalid orderId",
	)
	ErrInvalidPaymentID = errors.New(
		"refund: invalid paymentId",
	)
	ErrPaymentOrderMismatch = errors.New(
		"refund: paymentId does not match orderId",
	)
	ErrInvalidOrderItemIndex = errors.New(
		"refund: invalid orderItemIndex",
	)
	ErrInvalidSellerType = errors.New(
		"refund: invalid sellerType",
	)
	ErrInvalidSellerIdentity = errors.New(
		"refund: invalid seller identity",
	)
	ErrInvalidCompanyID = errors.New(
		"refund: invalid companyId",
	)
	ErrInvalidAccountID = errors.New(
		"refund: invalid accountId",
	)
	ErrInvalidAvatarID = errors.New(
		"refund: invalid avatarId",
	)
	ErrInvalidUserID = errors.New(
		"refund: invalid userId",
	)
	ErrInvalidPayoutAccountID = errors.New(
		"refund: invalid payoutAccountId",
	)
	ErrInvalidStripeAccountID = errors.New(
		"refund: invalid stripeAccountId",
	)
	ErrInvalidSettlementID = errors.New(
		"refund: invalid settlementId",
	)
	ErrInvalidSalesReceivableID = errors.New(
		"refund: invalid salesReceivableId",
	)
	ErrInvalidReturnRefundAmount = errors.New(
		"refund: invalid return refund amount",
	)
	ErrInvalidReturnRefundAmounts = errors.New(
		"refund: invalid return refund amounts",
	)
	ErrInvalidMerchandiseAmount = errors.New(
		"refund: invalid merchandiseAmount",
	)
	ErrInvalidMerchandiseTaxAmount = errors.New(
		"refund: invalid merchandiseTaxAmount",
	)
	ErrInvalidOutboundShippingAmount = errors.New(
		"refund: invalid outboundShippingAmount",
	)
	ErrInvalidOutboundShippingTaxAmount = errors.New(
		"refund: invalid outboundShippingTaxAmount",
	)
	ErrInvalidReturnShippingAmount = errors.New(
		"refund: invalid returnShippingAmount",
	)
	ErrInvalidReturnShippingTaxAmount = errors.New(
		"refund: invalid returnShippingTaxAmount",
	)
	ErrInvalidRefundAmount = errors.New(
		"refund: invalid refundAmount",
	)
	ErrRefundAmountMismatch = errors.New(
		"refund: refundAmount does not equal refundable merchandise and outbound shipping amounts",
	)
	ErrInvalidCurrency = errors.New(
		"refund: invalid currency",
	)
	ErrInvalidStripeRefundID = errors.New(
		"refund: invalid stripeRefundId",
	)
	ErrInvalidStatus = errors.New(
		"refund: invalid status",
	)
	ErrInvalidStatusTransition = errors.New(
		"refund: invalid status transition",
	)
	ErrInvalidRefundedAt = errors.New(
		"refund: invalid refundedAt",
	)
	ErrInvalidTransferReversalAmount = errors.New(
		"refund: invalid transferReversalAmount",
	)
	ErrInvalidStripeTransferReversalID = errors.New(
		"refund: invalid stripeTransferReversalId",
	)
	ErrInvalidTransferReversalStatus = errors.New(
		"refund: invalid transfer reversal status",
	)
	ErrInvalidTransferReversalStatusTransition = errors.New(
		"refund: invalid transfer reversal status transition",
	)
	ErrInvalidTransferReversedAt = errors.New(
		"refund: invalid transferReversedAt",
	)
	ErrTransferReversalRequiresSucceededRefund = errors.New(
		"refund: transfer reversal requires succeeded refund",
	)
	ErrInvalidCreatedAt = errors.New(
		"refund: invalid createdAt",
	)
	ErrInvalidUpdatedAt = errors.New(
		"refund: invalid updatedAt",
	)
)
