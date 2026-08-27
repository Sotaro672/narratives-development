// backend/internal/application/usecase/return_receipt_usecase.go
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
	ErrReturnReceiptUsecaseNotConfigured = errors.New(
		"return receipt usecase: not configured",
	)

	ErrReturnReceiptInvalidCompanyID = errors.New(
		"return receipt usecase: invalid companyId",
	)

	ErrReturnReceiptInvalidMemberID = errors.New(
		"return receipt usecase: invalid memberId",
	)

	ErrReturnReceiptInvalidInquiryType = errors.New(
		"return receipt usecase: inquiry is not return_unopened",
	)

	ErrReturnReceiptInquiryNotOpen = errors.New(
		"return receipt usecase: inquiry is not open",
	)

	ErrReturnReceiptInquiryClosed = errors.New(
		"return receipt usecase: inquiry is closed",
	)

	ErrReturnReceiptOrderMismatch = errors.New(
		"return receipt usecase: inquiry does not match order",
	)

	ErrReturnReceiptCompanyMismatch = errors.New(
		"return receipt usecase: order item does not belong to company",
	)

	ErrReturnReceiptOrderNotPaid = errors.New(
		"return receipt usecase: order is not paid",
	)

	ErrReturnReceiptReturnNotRequested = errors.New(
		"return receipt usecase: return is not requested",
	)

	ErrReturnReceiptReturnNotUnopened = errors.New(
		"return receipt usecase: return item is no longer unopened",
	)

	ErrReturnReceiptRefundMismatch = errors.New(
		"return receipt usecase: refund does not match return target",
	)

	ErrReturnReceiptOrderCompletionMismatch = errors.New(
		"return receipt usecase: order return completion mismatch",
	)

	ErrReturnReceiptInquiryResolutionMismatch = errors.New(
		"return receipt usecase: inquiry resolution mismatch",
	)
)

// ============================================================
// Order Port
// ============================================================

// ReturnReceiptOrderService is the minimum Order application service required
// by ReturnReceiptUsecase.
//
// Order is the authoritative source for:
//
// - purchaser / avatar ownership
// - item index
// - seller Company / Account
// - return-request state
// - unopened state
// - return-completion state
type ReturnReceiptOrderService interface {
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

// ReturnReceiptInquiryResolver is the minimum Inquiry application service
// required after financial completion.
//
// Resolution must go through InquiryUsecase instead of mutating the repository
// directly so Inquiry domain lifecycle rules remain authoritative.
type ReturnReceiptInquiryResolver interface {
	ResolveByMember(
		ctx context.Context,
		in ResolveInquiryInput,
	) (inquirydom.Inquiry, error)
}

// ============================================================
// Item Refund Port
// ============================================================

// RefundOrderItemInput identifies the item-level financial refund that belongs
// to one return Inquiry.
//
// No amount is accepted from the HTTP/frontend layer.
//
// ItemRefundUsecase must resolve the authoritative refund amount from:
//
//	Order
//	-> Order item
//	-> merchandise amount
//	-> allocated merchandise consumption tax
//
// and coordinate the corresponding seller-side partial Transfer Reversal.
type RefundOrderItemInput struct {
	InquiryID string

	OrderID   string
	ItemIndex int

	CompanyID string
}

// ReturnReceiptItemRefundService is the item-level financial service required
// by ReturnReceiptUsecase.
//
// RefundOrderItem must be idempotent.
//
// A deterministic Refund ID and deterministic Stripe idempotency keys should
// ensure that retrying the receive-return endpoint does not create duplicate:
//
// - Stripe Refund objects
// - Stripe Transfer Reversals
// - Refund Firestore documents
type ReturnReceiptItemRefundService interface {
	RefundOrderItem(
		ctx context.Context,
		in RefundOrderItemInput,
	) (refunddom.Refund, error)
}

// ReturnReceiptRefundCompletionNotifier is the minimum notification contract
// required after one item-level return has completed financially.
//
// EnsureDelivery must be idempotent so retrying ReceiveReturn can repair a
// previously completed refund whose notification delivery was not yet queued.
type ReturnReceiptRefundCompletionNotifier interface {
	EnsureDelivery(
		ctx context.Context,
		in EnsureRefundCompletionNotificationInput,
	) (refunddom.CompletionNotificationDelivery, error)
}

// ============================================================
// Input / Result
// ============================================================

// ReceiveReturnInput identifies one Console return receipt operation.
//
// CompanyID and MemberID must come from authenticated Console context.
// They must not be trusted from arbitrary request-body values.
type ReceiveReturnInput struct {
	InquiryID string

	CompanyID string
	MemberID  string
}

// ReturnReceiptResult represents the complete state of one return receipt
// attempt.
//
// FinanciallyCompleted:
//
//	true only after purchaser refund and every required seller-side
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
// If Stripe returns a pending refund state, ReceiveReturn returns a result with
// FinanciallyCompleted=false and does not complete Order or Inquiry.
type ReturnReceiptResult struct {
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

// ReturnReceiptUsecase coordinates receipt of one unopened returned item.
//
// This usecase intentionally acts as an orchestrator.
//
// Responsibilities:
//
//  1. Load the authoritative Inquiry.
//  2. Require inquiryType=return_unopened.
//  3. Resolve the authoritative Order and Order item.
//  4. Validate company boundary.
//  5. Revalidate that the physical item is still unopened.
//  6. Execute or resume the item-level financial Refund.
//  7. Require Refund.IsFinanciallyCompleted().
//  8. Mark the Order item return as completed.
//  9. Resolve the Inquiry.
//  10. Ensure the purchaser refund-completion notification delivery.
//
// The execution order is intentional:
//
//	Stripe purchaser Refund
//	-> seller Transfer Reversal
//	-> Order IsReturnCompleted
//	-> Inquiry resolved
//	-> refund completion notification delivery
//
// Order and Inquiry must never be marked completed before the financial Refund
// aggregate is financially completed.
//
// Stripe and Firestore cannot participate in one transaction. Therefore every
// step before Order / Inquiry completion must be idempotent.
type ReturnReceiptUsecase struct {
	orderService ReturnReceiptOrderService

	inquiryRepo     inquirydom.Repository
	inquiryResolver ReturnReceiptInquiryResolver

	itemRefundService ReturnReceiptItemRefundService

	refundCompletionNotifier ReturnReceiptRefundCompletionNotifier
}

func NewReturnReceiptUsecase(
	orderService ReturnReceiptOrderService,
	inquiryRepo inquirydom.Repository,
	inquiryResolver ReturnReceiptInquiryResolver,
	itemRefundService ReturnReceiptItemRefundService,
) *ReturnReceiptUsecase {
	return &ReturnReceiptUsecase{
		orderService:      orderService,
		inquiryRepo:       inquiryRepo,
		inquiryResolver:   inquiryResolver,
		itemRefundService: itemRefundService,
	}
}

func (uc *ReturnReceiptUsecase) WithRefundCompletionNotifier(
	notifier ReturnReceiptRefundCompletionNotifier,
) *ReturnReceiptUsecase {
	if uc == nil {
		return nil
	}

	uc.refundCompletionNotifier = notifier
	return uc
}

// ============================================================
// Receive Return
// ============================================================

// ReceiveReturn receives one unopened return.
//
// The frontend sends only the Inquiry ID through the route.
//
// CompanyID and MemberID are supplied from authenticated Console context.
//
// Refund amount, Order ID, item index, Account ID, Settlement ID, tax rate,
// merchandise amount, and tax amount are never accepted from the frontend.
//
// A successful completed flow ends with:
//
//	Refund.IsFinanciallyCompleted() == true
//	Order.Items[itemIndex].IsReturnCompleted == true
//	Inquiry.Status == resolved
//
// If Stripe Refund is pending, the Refund is returned but Order / Inquiry remain
// in their current return-processing state.
func (uc *ReturnReceiptUsecase) ReceiveReturn(
	ctx context.Context,
	in ReceiveReturnInput,
) (ReturnReceiptResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnReceiptResult{}, err
	}

	inquiryID := strings.TrimSpace(
		in.InquiryID,
	)
	if inquiryID == "" {
		return ReturnReceiptResult{},
			inquirydom.ErrInvalidID
	}

	companyID := strings.TrimSpace(
		in.CompanyID,
	)
	if companyID == "" {
		return ReturnReceiptResult{},
			ErrReturnReceiptInvalidCompanyID
	}

	memberID := strings.TrimSpace(
		in.MemberID,
	)
	if memberID == "" {
		return ReturnReceiptResult{},
			ErrReturnReceiptInvalidMemberID
	}

	inquiry, err := uc.inquiryRepo.GetByID(
		ctx,
		inquiryID,
	)
	if err != nil {
		return ReturnReceiptResult{}, err
	}

	if inquiry.ID != inquiryID ||
		inquiry.DeletedAt != nil {
		return ReturnReceiptResult{},
			inquirydom.ErrNotFound
	}

	if inquiry.InquiryType !=
		inquirydom.InquiryTypeReturnUnopened {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrReturnReceiptInvalidInquiryType
	}

	if inquiry.Status ==
		inquirydom.InquiryStatusClosed {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrReturnReceiptInquiryClosed
	}

	if inquiry.OrderID == "" ||
		inquiry.OrderItemIndex == nil ||
		*inquiry.OrderItemIndex < 0 ||
		inquiry.AvatarID == "" {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrReturnReceiptOrderMismatch
	}

	orderID := inquiry.OrderID
	itemIndex := *inquiry.OrderItemIndex

	order, err := uc.orderService.GetByID(
		ctx,
		orderID,
	)
	if err != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, err
	}

	if order.ID != orderID ||
		order.AvatarID != inquiry.AvatarID {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptOrderMismatch
	}

	if itemIndex >= len(order.Items) {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptOrderMismatch
	}

	targetItem := order.Items[itemIndex]

	if targetItem.SellerSnapshot.CompanyID !=
		companyID {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptCompanyMismatch
	}

	if !order.Paid {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptOrderNotPaid
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		!targetItem.IsReturnRequested ||
		targetItem.ReturnRequestedAt == nil ||
		targetItem.ReturnRequestedAt.IsZero() {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptReturnNotRequested
	}

	// return_unopened must still represent a physically unopened item.
	//
	// If a valid scan has already occurred, ReturnRequestUsecase should promote
	// the Inquiry to return_opened. Even if that promotion has not yet been
	// persisted because of a concurrent failure, the authoritative Order state
	// prevents an unopened refund from proceeding here.
	if targetItem.Transferred ||
		targetItem.TokenTransferVerifiedAt != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptReturnNotUnopened
	}

	// A resolved return_unopened Inquiry is valid here only when the Order item
	// has already completed.
	//
	// This protects against a generic manual resolve operation bypassing the
	// financial return workflow.
	if inquiry.Status ==
		inquirydom.InquiryStatusResolved &&
		!targetItem.IsReturnCompleted {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptInquiryNotOpen
	}

	refund, err := uc.itemRefundService.RefundOrderItem(
		ctx,
		RefundOrderItemInput{
			InquiryID: inquiry.ID,
			OrderID:   order.ID,
			ItemIndex: itemIndex,
			CompanyID: companyID,
		},
	)
	if err != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
			Refund:  refund,
		}, err
	}

	result := ReturnReceiptResult{
		Inquiry: inquiry,
		Order:   order,
		Refund:  refund,
	}

	if err := validateReturnReceiptRefund(
		inquiry,
		order,
		itemIndex,
		refund,
	); err != nil {
		return result, err
	}

	result.FinanciallyCompleted =
		refund.IsFinanciallyCompleted()

	if !result.FinanciallyCompleted {
		// Stripe may have accepted the Refund but still report pending or
		// requires_action.
		//
		// Do not complete the Order and do not resolve the Inquiry until the
		// Refund aggregate confirms both purchaser and seller-side financial
		// completion.
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
			ErrReturnReceiptOrderCompletionMismatch
	}

	result.OrderCompleted = true

	// Inquiry resolution is always last.
	//
	// If this step fails, a retry resumes from the deterministic Refund and the
	// already-completed Order item, then retries only Inquiry resolution.
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
			ErrReturnReceiptInquiryResolutionMismatch
	}

	result.InquiryResolved = true

	// EnsureDelivery is intentionally executed even when the Order and Inquiry
	// were already completed by a previous attempt.
	//
	// This repairs the partial-failure case where the financial refund, Order
	// completion, and Inquiry resolution succeeded but delivery creation or
	// Cloud Tasks enqueue did not complete.
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

func (uc *ReturnReceiptUsecase) validateConfigured() error {
	if uc == nil ||
		uc.orderService == nil ||
		uc.inquiryRepo == nil ||
		uc.inquiryResolver == nil ||
		uc.itemRefundService == nil ||
		uc.refundCompletionNotifier == nil {
		return ErrReturnReceiptUsecaseNotConfigured
	}

	return nil
}

func validateReturnReceiptRefund(
	inquiry inquirydom.Inquiry,
	order orderdom.Order,
	itemIndex int,
	refund refunddom.Refund,
) error {
	if err := refund.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrReturnReceiptRefundMismatch,
			err,
		)
	}

	if refund.InquiryID !=
		inquiry.ID {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != itemIndex {
		return ErrReturnReceiptRefundMismatch
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) {
		return ErrReturnReceiptRefundMismatch
	}

	targetItem := order.Items[itemIndex]

	if refund.CompanyID !=
		targetItem.SellerSnapshot.CompanyID {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.AccountID !=
		targetItem.SellerSnapshot.AccountID {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.Currency !=
		refunddom.CurrencyJPY {
		return ErrReturnReceiptRefundMismatch
	}

	// Recalculate the expected purchaser refund from the authoritative Order
	// snapshot.
	//
	// Shipping and shipping consumption tax are intentionally excluded.
	expectedAmount, err :=
		orderdom.CalculateOrderItemRefundAmount(
			order,
			itemIndex,
		)
	if err != nil {
		return err
	}

	if refund.MerchandiseAmount !=
		expectedAmount.MerchandiseAmount {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.MerchandiseTaxAmount !=
		expectedAmount.MerchandiseTaxAmount {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.RefundAmount !=
		expectedAmount.RefundAmount {
		return ErrReturnReceiptRefundMismatch
	}

	return nil
}
