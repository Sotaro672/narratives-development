// backend/internal/application/usecase/opened_return_receipt_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrOpenedReturnReceiptUsecaseNotConfigured = errors.New(
		"opened return receipt usecase: not configured",
	)

	ErrOpenedReturnReceiptInvalidCompanyID = errors.New(
		"opened return receipt usecase: invalid companyId",
	)

	ErrOpenedReturnReceiptInvalidMemberID = errors.New(
		"opened return receipt usecase: invalid memberId",
	)

	ErrOpenedReturnReceiptInvalidInquiryType = errors.New(
		"opened return receipt usecase: inquiry is not return_opened",
	)

	ErrOpenedReturnReceiptInquiryNotOpen = errors.New(
		"opened return receipt usecase: inquiry is not open",
	)

	ErrOpenedReturnReceiptInquiryClosed = errors.New(
		"opened return receipt usecase: inquiry is closed",
	)

	ErrOpenedReturnReceiptOrderMismatch = errors.New(
		"opened return receipt usecase: inquiry does not match order",
	)

	ErrOpenedReturnReceiptCompanyMismatch = errors.New(
		"opened return receipt usecase: order item does not belong to company",
	)

	ErrOpenedReturnReceiptOrderNotPaid = errors.New(
		"opened return receipt usecase: order is not paid",
	)

	ErrOpenedReturnReceiptReturnNotRequested = errors.New(
		"opened return receipt usecase: return is not requested",
	)

	ErrOpenedReturnReceiptReturnNotOpened = errors.New(
		"opened return receipt usecase: return item is not opened",
	)

	ErrOpenedReturnReceiptRefundMismatch = errors.New(
		"opened return receipt usecase: refund does not match return target",
	)

	ErrOpenedReturnReceiptOrderCompletionMismatch = errors.New(
		"opened return receipt usecase: order return completion mismatch",
	)

	ErrOpenedReturnReceiptInquiryResolutionMismatch = errors.New(
		"opened return receipt usecase: inquiry resolution mismatch",
	)
)

// ============================================================
// Order Port
// ============================================================

// OpenedReturnReceiptOrderService is the minimum Order application service
// required by OpenedReturnReceiptUsecase.
//
// Order remains authoritative for:
//
// - purchaser / avatar ownership
// - item index
// - seller Company / Account
// - payment state
// - return-request state
// - return-request kind
// - return-completion state
// - merchandise and shipping amounts
type OpenedReturnReceiptOrderService interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)

	CompleteReturnItem(
		ctx context.Context,
		in CompleteReturnOrderItemInput,
	) (orderdom.Order, error)
}

// ============================================================
// Inquiry Port
// ============================================================

// OpenedReturnReceiptInquiryResolver handles the purchaser-visible refund
// completion reply and resolves the Inquiry after every required financial
// operation and Order return completion have succeeded.
//
// EnsureReplyByMember must be idempotent for a deterministic reply ID so a
// retry never creates duplicate refund-completion replies.
type OpenedReturnReceiptInquiryResolver interface {
	EnsureReplyByMember(
		ctx context.Context,
		replyID string,
		inquiryID string,
		memberID string,
		content string,
		images []inquirydom.ImageFile,
	) (inquirydom.Reply, error)

	ResolveByMember(
		ctx context.Context,
		in ResolveInquiryInput,
	) (inquirydom.Inquiry, error)
}

// ============================================================
// Item Refund Port
// ============================================================

// OpenedReturnReceiptItemRefundService executes or resumes the financial flow
// for one opened return.
//
// RefundOpenedReturnOrderItem must calculate every amount from the authoritative
// Order snapshot. The caller supplies only the selected refund Policy.
type OpenedReturnReceiptItemRefundService interface {
	RefundOpenedReturnOrderItem(
		ctx context.Context,
		in RefundOpenedReturnItemInput,
	) (refunddom.Refund, error)
}

// ============================================================
// Input / Result
// ============================================================

// ReceiveOpenedReturnInput identifies one Console opened-return receipt
// operation.
//
// CompanyID and MemberID must come from authenticated Console context.
//
// Policy is the only client-selected financial parameter. Monetary amounts must
// never be accepted from the HTTP request.
type ReceiveOpenedReturnInput struct {
	InquiryID string

	CompanyID string
	MemberID  string

	Policy refunddom.OpenedReturnRefundPolicy
}

// OpenedReturnReceiptResult represents one opened-return receipt attempt.
//
// FinanciallyCompleted:
//
//	true only after the purchaser Stripe Refund and every required seller-side
//	Transfer Reversal have completed.
//
// OrderCompleted:
//
//	true only after Order item IsReturnCompleted becomes true.
//
// InquiryResolved:
//
//	true only after Inquiry becomes resolved.
//
// AlreadyCompleted:
//
//	true when the Order item had already completed before this invocation.
//
// If Stripe reports pending / requires_action, this result is returned with
// FinanciallyCompleted=false and Order / Inquiry are left incomplete.
type OpenedReturnReceiptResult struct {
	Inquiry inquirydom.Inquiry
	Order   orderdom.Order
	Refund  refunddom.Refund

	FinanciallyCompleted bool
	OrderCompleted       bool
	InquiryResolved      bool

	AlreadyCompleted bool
}

// ============================================================
// Usecase
// ============================================================

// OpenedReturnReceiptUsecase coordinates receipt of one opened returned item.
//
// Responsibilities:
//
//  1. Load authoritative Inquiry.
//  2. Require inquiryType=return_opened.
//  3. Validate selected refund Policy.
//  4. Resolve authoritative Order and Order item.
//  5. Validate company boundary.
//  6. Require Order.ReturnRequestKind=opened.
//  7. Execute or resume opened-return financial Refund.
//  8. Require Refund.IsFinanciallyCompleted().
//  9. Mark the Order item return as completed.
//  10. Ensure the refund amount / breakdown reply from the company member.
//  11. Resolve the Inquiry.
//  12. Ensure the purchaser refund-completion notification delivery.
//
// Execution order:
//
//	Stripe purchaser Refund
//	-> seller Transfer Reversal
//	-> Order IsReturnCompleted
//	-> refund completion reply
//	-> Inquiry resolved
//	-> refund completion notification delivery
//
// Order and Inquiry must never be marked complete before the financial Refund
// aggregate is complete.
//
// Stripe and Firestore cannot participate in one transaction. Every step that
// can be retried must therefore remain idempotent.
type OpenedReturnReceiptUsecase struct {
	orderService OpenedReturnReceiptOrderService

	inquiryRepo     inquirydom.Repository
	inquiryResolver OpenedReturnReceiptInquiryResolver

	itemRefundService OpenedReturnReceiptItemRefundService

	refundCompletionNotifier ReturnReceiptRefundCompletionNotifier
}

func NewOpenedReturnReceiptUsecase(
	orderService OpenedReturnReceiptOrderService,
	inquiryRepo inquirydom.Repository,
	inquiryResolver OpenedReturnReceiptInquiryResolver,
	itemRefundService OpenedReturnReceiptItemRefundService,
) *OpenedReturnReceiptUsecase {
	return &OpenedReturnReceiptUsecase{
		orderService:      orderService,
		inquiryRepo:       inquiryRepo,
		inquiryResolver:   inquiryResolver,
		itemRefundService: itemRefundService,
	}
}

func (uc *OpenedReturnReceiptUsecase) WithRefundCompletionNotifier(
	notifier ReturnReceiptRefundCompletionNotifier,
) *OpenedReturnReceiptUsecase {
	if uc == nil {
		return nil
	}

	uc.refundCompletionNotifier = notifier
	return uc
}

// ============================================================
// Receive Opened Return
// ============================================================

// ReceiveOpenedReturn receives one opened returned item.
//
// The frontend may submit only:
//
// - Inquiry ID through the route
// - one OpenedReturnRefundPolicy
//
// CompanyID and MemberID come from authenticated Console context.
//
// The frontend must never submit:
//
// - refund amount
// - merchandise amount
// - tax amount
// - shipping amount
// - Order item index
// - Account ID
// - Settlement ID
//
// A successful completed flow ends with:
//
//	Refund.IsFinanciallyCompleted() == true
//	Order.Items[itemIndex].IsReturnCompleted == true
//	refund completion reply exists
//	Inquiry.Status == resolved
func (uc *OpenedReturnReceiptUsecase) ReceiveOpenedReturn(
	ctx context.Context,
	in ReceiveOpenedReturnInput,
) (OpenedReturnReceiptResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	inquiryID := strings.TrimSpace(in.InquiryID)
	if inquiryID == "" {
		return OpenedReturnReceiptResult{},
			inquirydom.ErrInvalidID
	}

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return OpenedReturnReceiptResult{},
			ErrOpenedReturnReceiptInvalidCompanyID
	}

	memberID := strings.TrimSpace(in.MemberID)
	if memberID == "" {
		return OpenedReturnReceiptResult{},
			ErrOpenedReturnReceiptInvalidMemberID
	}

	if err := refunddom.ValidateOpenedReturnRefundPolicy(in.Policy); err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	inquiry, err := uc.inquiryRepo.GetByID(
		ctx,
		inquiryID,
	)
	if err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	if inquiry.ID != inquiryID ||
		inquiry.DeletedAt != nil {
		return OpenedReturnReceiptResult{},
			inquirydom.ErrNotFound
	}

	if inquiry.InquiryType !=
		inquirydom.InquiryTypeReturnOpened {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrOpenedReturnReceiptInvalidInquiryType
	}

	if inquiry.Status ==
		inquirydom.InquiryStatusClosed {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrOpenedReturnReceiptInquiryClosed
	}

	if inquiry.OrderID == "" ||
		inquiry.OrderItemIndex == nil ||
		*inquiry.OrderItemIndex < 0 ||
		inquiry.AvatarID == "" {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrOpenedReturnReceiptOrderMismatch
	}

	orderID := inquiry.OrderID
	itemIndex := *inquiry.OrderItemIndex

	order, err := uc.orderService.GetByID(
		ctx,
		orderID,
	)
	if err != nil {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, err
	}

	if order.ID != orderID ||
		order.AvatarID != inquiry.AvatarID {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptOrderMismatch
	}

	if itemIndex >= len(order.Items) {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptOrderMismatch
	}

	targetItem := order.Items[itemIndex]

	if targetItem.SellerSnapshot.CompanyID !=
		companyID {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptCompanyMismatch
	}

	if !order.Paid {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptOrderNotPaid
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		!targetItem.IsReturnRequested ||
		targetItem.ReturnRequestedAt == nil ||
		targetItem.ReturnRequestedAt.IsZero() {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptReturnNotRequested
	}

	if targetItem.ReturnRequestKind !=
		orderdom.ReturnRequestKindOpened {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptReturnNotOpened
	}

	// A resolved return_opened Inquiry is valid here only when the Order item
	// has already completed.
	//
	// This prevents a generic Inquiry resolve operation from bypassing the
	// selected refund policy and financial workflow.
	if inquiry.Status ==
		inquirydom.InquiryStatusResolved &&
		!targetItem.IsReturnCompleted {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptInquiryNotOpen
	}

	refund, err := uc.itemRefundService.RefundOpenedReturnOrderItem(
		ctx,
		RefundOpenedReturnItemInput{
			InquiryID: inquiry.ID,
			OrderID:   order.ID,
			ItemIndex: itemIndex,
			CompanyID: companyID,
			Policy:    in.Policy,
		},
	)
	if err != nil {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
			Refund:  refund,
		}, err
	}

	result := OpenedReturnReceiptResult{
		Inquiry: inquiry,
		Order:   order,
		Refund:  refund,
	}

	if err := validateOpenedReturnReceiptRefund(
		inquiry,
		order,
		itemIndex,
		in.Policy,
		refund,
	); err != nil {
		return result, err
	}

	result.FinanciallyCompleted =
		refund.IsFinanciallyCompleted()

	if !result.FinanciallyCompleted {
		// Stripe may have accepted the purchaser Refund while still reporting
		// pending / requires_action.
		//
		// Do not complete Order or Inquiry until the Refund aggregate confirms
		// purchaser refund and required seller-side Transfer Reversal completion.
		return result, nil
	}

	alreadyOrderCompleted :=
		targetItem.IsReturnCompleted

	if !alreadyOrderCompleted {
		completedOrder, err :=
			uc.orderService.CompleteReturnItem(
				ctx,
				CompleteReturnOrderItemInput{
					ID:        order.ID,
					ItemIndex: itemIndex,
				},
			)
		if err != nil {
			return result, err
		}

		order = completedOrder
		result.Order = completedOrder
	}

	if itemIndex >= len(order.Items) ||
		!order.Items[itemIndex].IsReturnCompleted ||
		order.Items[itemIndex].ReturnCompletedAt == nil ||
		order.Items[itemIndex].ReturnCompletedAt.IsZero() {
		return result,
			ErrOpenedReturnReceiptOrderCompletionMismatch
	}

	result.OrderCompleted = true

	// Ensure the purchaser-visible refund completion reply before resolving the
	// Inquiry.
	//
	// The reply uses only authoritative values persisted in Refund. In
	// particular, ReturnShippingAmount / ReturnShippingTaxAmount are company-side
	// burden and are not included in the purchaser Stripe refund breakdown.
	replyContent, err :=
		buildRefundCompletionReplyContent(
			refund,
		)
	if err != nil {
		return result, err
	}

	// The deterministic reply ID makes this operation idempotent.
	//
	// If the reply was created by a previous attempt and a later operation
	// failed, EnsureReplyByMember returns the existing matching reply rather than
	// creating another purchaser-visible message.
	_, err =
		uc.inquiryResolver.EnsureReplyByMember(
			ctx,
			refundCompletionReplyID(
				refund.ID,
			),
			inquiry.ID,
			memberID,
			replyContent,
			nil,
		)
	if err != nil {
		return result, err
	}

	// Inquiry resolution follows the refund completion reply.
	//
	// If this step fails, retrying resumes from the deterministic Refund and
	// already-completed Order item, ensures the same deterministic reply, and
	// retries Inquiry resolution.
	if inquiry.Status !=
		inquirydom.InquiryStatusResolved {
		resolvedInquiry, err :=
			uc.inquiryResolver.ResolveByMember(
				ctx,
				ResolveInquiryInput{
					InquiryID: inquiry.ID,
					MemberID:  memberID,
				},
			)
		if err != nil {
			return result, err
		}

		inquiry = resolvedInquiry
		result.Inquiry = resolvedInquiry
	}

	if inquiry.Status !=
		inquirydom.InquiryStatusResolved ||
		inquiry.ResolvedAt == nil ||
		inquiry.ResolvedAt.IsZero() ||
		inquiry.ResolvedBy == nil ||
		*inquiry.ResolvedBy == "" {
		return result,
			ErrOpenedReturnReceiptInquiryResolutionMismatch
	}

	result.InquiryResolved = true

	// EnsureDelivery is intentionally executed even when the Order and Inquiry
	// were already completed by a previous attempt.
	//
	// This allows a retry to repair the partial-failure case where the financial
	// refund, Order completion, refund completion reply, and Inquiry resolution
	// succeeded but delivery creation or Cloud Tasks enqueue did not complete.
	_, err = uc.refundCompletionNotifier.EnsureDelivery(
		ctx,
		EnsureRefundCompletionNotificationInput{
			PaymentID:      refund.PaymentID,
			OrderID:        refund.OrderID,
			UserID:         order.UserID,
			StripeRefundID: refund.StripeRefundID,
			RefundedAmount: refund.RefundAmount,
		},
	)
	if err != nil {
		return result, err
	}

	result.AlreadyCompleted =
		alreadyOrderCompleted

	return result, nil
}

// ============================================================
// Validation
// ============================================================

func (uc *OpenedReturnReceiptUsecase) validateConfigured() error {
	if uc == nil ||
		uc.orderService == nil ||
		uc.inquiryRepo == nil ||
		uc.inquiryResolver == nil ||
		uc.itemRefundService == nil ||
		uc.refundCompletionNotifier == nil {
		return ErrOpenedReturnReceiptUsecaseNotConfigured
	}

	return nil
}

func validateOpenedReturnReceiptRefund(
	inquiry inquirydom.Inquiry,
	order orderdom.Order,
	itemIndex int,
	policy refunddom.OpenedReturnRefundPolicy,
	refund refunddom.Refund,
) error {
	if err := refund.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrOpenedReturnReceiptRefundMismatch,
			err,
		)
	}

	if refund.InquiryID != inquiry.ID {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != itemIndex {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	targetItem := order.Items[itemIndex]

	if refund.CompanyID !=
		targetItem.SellerSnapshot.CompanyID {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.AccountID !=
		targetItem.SellerSnapshot.AccountID {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.Policy != policy {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.Currency !=
		refunddom.CurrencyJPY {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	// Recalculate every purchaser-refund and return-shipping amount from the
	// authoritative Order snapshot. The frontend-selected Policy is the only
	// non-Order value involved in this calculation.
	expectedAmount, err :=
		orderdom.CalculateOpenedReturnRefundAmount(
			order,
			itemIndex,
			policy,
		)
	if err != nil {
		return err
	}

	if refund.MerchandiseAmount !=
		expectedAmount.MerchandiseAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.MerchandiseTaxAmount !=
		expectedAmount.MerchandiseTaxAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.OutboundShippingAmount !=
		expectedAmount.OutboundShippingAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.OutboundShippingTaxAmount !=
		expectedAmount.OutboundShippingTaxAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.ReturnShippingAmount !=
		expectedAmount.ReturnShippingAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.ReturnShippingTaxAmount !=
		expectedAmount.ReturnShippingTaxAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.RefundAmount !=
		expectedAmount.StripeRefundAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	totalBrandBurdenAmount, err :=
		refund.TotalBrandBurdenAmount()
	if err != nil {
		return err
	}

	if totalBrandBurdenAmount !=
		expectedAmount.TotalBrandBurdenAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	return nil
}
