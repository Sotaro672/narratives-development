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
	salesreceivabledom "narratives/internal/domain/salesReceivable"
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
// Order remains authoritative for purchaser ownership, item identity, immutable
// seller snapshot, payment state, return-request state, return kind,
// return-completion state and every monetary amount.
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
// completion reply and resolves the Inquiry only after financial completion and
// Order return completion.
//
// EnsureReplyByMember must be idempotent for a deterministic reply ID.
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
// RefundOpenedReturnOrderItem calculates every amount from the authoritative
// Order snapshot. The caller supplies only identity fields and the selected
// refund Policy.
//
// For a primary List item, seller-side financial state is Settlement.
// For a consumer resale item, seller-side financial state is SalesReceivable.
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
	Policy    refunddom.OpenedReturnRefundPolicy
}

// OpenedReturnReceiptResult represents one opened-return receipt attempt.
//
// FinanciallyCompleted is true only after the purchaser Stripe Refund and every
// required seller-side financial operation have completed.
//
// List:
//   - Settlement / Stripe Transfer Reversal
//
// Resale:
//   - SalesReceivable cancellation
//
// OrderCompleted is true only after Order item IsReturnCompleted becomes true.
// InquiryResolved is true only after Inquiry becomes resolved.
// AlreadyCompleted is true when the Order item had already completed before this
// invocation.
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
// Execution:
//
//	Inquiry
//	-> authoritative Order/item
//	-> seller/item-type validation
//	-> opened-return financial Refund
//	-> seller-side financial completion
//	-> Order IsReturnCompleted
//	-> refund completion reply
//	-> Inquiry resolved
//	-> refund completion notification
//
// Primary List items use Settlement and may require Stripe Transfer Reversal.
// Consumer resale items use SalesReceivable and never use Stripe Connect.
//
// Console company access to the Inquiry is established by the company-scoped
// Inquiry detail query before this usecase is invoked. Inside this usecase,
// CompanyID is additionally matched to SellerSnapshot.CompanyID only for List
// items because a resale seller is an Avatar/User, not a Company.
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
//   - Inquiry ID through the route
//   - one OpenedReturnRefundPolicy
//
// CompanyID and MemberID come from authenticated Console context.
//
// Refund amount, merchandise amount, tax amount, shipping amounts, Order item
// index, seller identity, Settlement ID and SalesReceivable ID are never accepted
// from the frontend.
func (uc *OpenedReturnReceiptUsecase) ReceiveOpenedReturn(
	ctx context.Context,
	in ReceiveOpenedReturnInput,
) (OpenedReturnReceiptResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	inquiryID := strings.TrimSpace(in.InquiryID)
	if inquiryID == "" {
		return OpenedReturnReceiptResult{}, inquirydom.ErrInvalidID
	}

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return OpenedReturnReceiptResult{}, ErrOpenedReturnReceiptInvalidCompanyID
	}

	memberID := strings.TrimSpace(in.MemberID)
	if memberID == "" {
		return OpenedReturnReceiptResult{}, ErrOpenedReturnReceiptInvalidMemberID
	}

	if err := refunddom.ValidateOpenedReturnRefundPolicy(in.Policy); err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	inquiry, err := uc.inquiryRepo.GetByID(ctx, inquiryID)
	if err != nil {
		return OpenedReturnReceiptResult{}, err
	}

	if inquiry.ID != inquiryID || inquiry.DeletedAt != nil {
		return OpenedReturnReceiptResult{}, inquirydom.ErrNotFound
	}

	if inquiry.InquiryType != inquirydom.InquiryTypeReturnOpened {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrOpenedReturnReceiptInvalidInquiryType
	}

	if inquiry.Status == inquirydom.InquiryStatusClosed {
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

	order, err := uc.orderService.GetByID(ctx, orderID)
	if err != nil {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
		}, err
	}

	if order.ID != orderID || order.AvatarID != inquiry.AvatarID {
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

	if err := validateOpenedReturnReceiptTargetSeller(
		companyID,
		targetItem,
	); err != nil {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, err
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

	if targetItem.ReturnRequestKind != orderdom.ReturnRequestKindOpened {
		return OpenedReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrOpenedReturnReceiptReturnNotOpened
	}

	// A resolved return_opened Inquiry is valid here only when the Order item has
	// already completed. This prevents generic Inquiry resolution from bypassing
	// the selected refund policy and financial workflow.
	if inquiry.Status == inquirydom.InquiryStatusResolved &&
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

	result.FinanciallyCompleted = refund.IsFinanciallyCompleted()
	if !result.FinanciallyCompleted {
		// Stripe may have accepted the purchaser Refund while still reporting
		// pending / requires_action. Do not complete Order or Inquiry until the
		// Refund aggregate confirms purchaser-side and seller-side completion.
		return result, nil
	}

	alreadyOrderCompleted := targetItem.IsReturnCompleted

	if !alreadyOrderCompleted {
		completedOrder, err := uc.orderService.CompleteReturnItem(
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
		return result, ErrOpenedReturnReceiptOrderCompletionMismatch
	}

	result.OrderCompleted = true

	// The reply contains only authoritative values persisted in Refund. Return
	// shipping amounts are seller-side burden and are not part of the original
	// purchaser Charge.
	replyContent, err := buildRefundCompletionReplyContent(refund)
	if err != nil {
		return result, err
	}

	// The deterministic reply ID makes this operation idempotent.
	_, err = uc.inquiryResolver.EnsureReplyByMember(
		ctx,
		refundCompletionReplyID(refund.ID),
		inquiry.ID,
		memberID,
		replyContent,
		nil,
	)
	if err != nil {
		return result, err
	}

	if inquiry.Status != inquirydom.InquiryStatusResolved {
		resolvedInquiry, err := uc.inquiryResolver.ResolveByMember(
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

	if inquiry.Status != inquirydom.InquiryStatusResolved ||
		inquiry.ResolvedAt == nil ||
		inquiry.ResolvedAt.IsZero() ||
		inquiry.ResolvedBy == nil ||
		*inquiry.ResolvedBy == "" {
		return result, ErrOpenedReturnReceiptInquiryResolutionMismatch
	}

	result.InquiryResolved = true

	// Always ensure notification delivery, including an idempotent retry after
	// financial, Order and Inquiry completion.
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

	result.AlreadyCompleted = alreadyOrderCompleted
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

// validateOpenedReturnReceiptTargetSeller validates the immutable seller
// snapshot according to the Order item type.
//
// List:
//   - authenticated Console Company must equal seller Company
//   - Company/Account/Stripe account identity is required
//   - resale seller fields must be empty
//
// Resale:
//   - seller is Avatar/User/PayoutAccount, not the authenticated Console Company
//   - Company access to the Inquiry is established by the company-scoped Inquiry
//     detail query before this usecase is invoked
//   - StripeAccountID must be empty
func validateOpenedReturnReceiptTargetSeller(
	companyID string,
	targetItem orderdom.OrderItemSnapshot,
) error {
	seller := targetItem.SellerSnapshot

	switch targetItem.Type {
	case orderdom.OrderItemTypeList:
		if seller.CompanyID != companyID {
			return ErrOpenedReturnReceiptCompanyMismatch
		}

		if seller.BrandID == "" ||
			seller.CompanyID == "" ||
			seller.AccountID == "" ||
			seller.StripeAccountID == "" ||
			seller.AvatarID != "" ||
			seller.UserID != "" ||
			seller.PayoutAccountID != "" {
			return ErrOpenedReturnReceiptOrderMismatch
		}

	case orderdom.OrderItemTypeResale:
		if targetItem.ResaleID == "" ||
			seller.AvatarID == "" ||
			seller.UserID == "" ||
			seller.PayoutAccountID == "" ||
			seller.PayoutAccountID != seller.UserID ||
			seller.BrandID != "" ||
			seller.CompanyID != "" ||
			seller.AccountID != "" ||
			seller.StripeAccountID != "" {
			return ErrOpenedReturnReceiptOrderMismatch
		}

	default:
		return ErrOpenedReturnReceiptOrderMismatch
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

	if itemIndex < 0 || itemIndex >= len(order.Items) {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	targetItem := order.Items[itemIndex]
	seller := targetItem.SellerSnapshot

	switch targetItem.Type {
	case orderdom.OrderItemTypeList:
		if refund.SellerType != refunddom.SellerTypeAccount ||
			refund.CompanyID != seller.CompanyID ||
			refund.AccountID != seller.AccountID ||
			refund.StripeAccountID != seller.StripeAccountID ||
			refund.AvatarID != "" ||
			refund.UserID != "" ||
			refund.PayoutAccountID != "" ||
			refund.SettlementID == "" ||
			refund.SalesReceivableID != "" {
			return ErrOpenedReturnReceiptRefundMismatch
		}

	case orderdom.OrderItemTypeResale:
		if refund.SellerType != refunddom.SellerTypeResale ||
			refund.CompanyID != "" ||
			refund.AccountID != "" ||
			refund.StripeAccountID != "" ||
			refund.AvatarID != seller.AvatarID ||
			refund.UserID != seller.UserID ||
			refund.PayoutAccountID != seller.PayoutAccountID ||
			refund.SettlementID != "" {
			return ErrOpenedReturnReceiptRefundMismatch
		}

		expectedSalesReceivableID, err := salesreceivabledom.NewID(
			order.ID,
			itemIndex,
		)
		if err != nil ||
			refund.SalesReceivableID != expectedSalesReceivableID {
			return ErrOpenedReturnReceiptRefundMismatch
		}

	default:
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.Policy != policy {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	if refund.Currency != refunddom.CurrencyJPY {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	// Recalculate every purchaser-refund and return-shipping amount from the
	// authoritative Order snapshot. Policy is the only non-Order financial input.
	expectedAmount, err := refunddom.CalculateOpenedReturnRefundAmount(
		order,
		itemIndex,
		policy,
	)
	if err != nil {
		return err
	}

	if refund.MerchandiseAmount != expectedAmount.MerchandiseAmount ||
		refund.MerchandiseTaxAmount != expectedAmount.MerchandiseTaxAmount ||
		refund.OutboundShippingAmount != expectedAmount.OutboundShippingAmount ||
		refund.OutboundShippingTaxAmount != expectedAmount.OutboundShippingTaxAmount ||
		refund.ReturnShippingAmount != expectedAmount.ReturnShippingAmount ||
		refund.ReturnShippingTaxAmount != expectedAmount.ReturnShippingTaxAmount ||
		refund.RefundAmount != expectedAmount.StripeRefundAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	totalSellerBurdenAmount, err := refund.TotalSellerBurdenAmount()
	if err != nil {
		return err
	}

	if totalSellerBurdenAmount != expectedAmount.TotalSellerBurdenAmount {
		return ErrOpenedReturnReceiptRefundMismatch
	}

	return nil
}
