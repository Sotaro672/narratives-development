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
	salesreceivabledom "narratives/internal/domain/salesReceivable"
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
// Order is authoritative for purchaser ownership, item identity, immutable seller
// snapshot, return-request state, unopened state and return-completion state.
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
// Reply creation and resolution must go through InquiryUsecase instead of
// mutating the repository directly so Inquiry domain lifecycle rules remain
// authoritative.
//
// EnsureReplyByMember must be idempotent for a deterministic reply ID.
type ReturnReceiptInquiryResolver interface {
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

// RefundOrderItemInput identifies the item-level financial refund belonging to
// one return Inquiry.
//
// No amount is accepted from the HTTP/frontend layer.
//
// CompanyID represents the authenticated Console scope. For a primary List item,
// ItemRefundUsecase also validates it against the seller Company. For a consumer
// resale item, CompanyID is not seller identity; seller financial state is
// resolved from Avatar/User/PayoutAccount and SalesReceivable.
type RefundOrderItemInput struct {
	InquiryID string
	OrderID   string
	ItemIndex int
	CompanyID string
}

// ReturnReceiptItemRefundService is the item-level financial service required by
// ReturnReceiptUsecase.
//
// RefundOrderItem must be idempotent. A deterministic Refund ID and deterministic
// Stripe idempotency keys prevent duplicate purchaser Refunds, seller Transfer
// Reversals and Refund documents.
type ReturnReceiptItemRefundService interface {
	RefundOrderItem(
		ctx context.Context,
		in RefundOrderItemInput,
	) (refunddom.Refund, error)
}

// ReturnReceiptRefundCompletionNotifier is the minimum notification contract
// required after one item-level return has completed financially.
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
// CompanyID and MemberID must come from authenticated Console context. They must
// not be trusted from arbitrary request-body values.
type ReceiveReturnInput struct {
	InquiryID string
	CompanyID string
	MemberID  string
}

// ReturnReceiptResult represents the complete state of one return receipt
// attempt.
//
// FinanciallyCompleted is true only after purchaser refund and the required
// seller-side financial action have completed.
//
// For List:
//   - Settlement / Stripe Transfer Reversal
//
// For Resale:
//   - SalesReceivable cancellation
//
// OrderCompleted is true only after the Order item is marked return-completed.
// InquiryResolved is true only after Inquiry becomes resolved.
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
// Execution:
//
//	Inquiry
//	-> authoritative Order/item
//	-> seller/item-type validation
//	-> item-level purchaser Refund and seller-side financial completion
//	-> Order IsReturnCompleted
//	-> refund completion reply
//	-> Inquiry resolved
//	-> refund completion notification
//
// Primary List items use Settlement and Stripe Transfer Reversal.
// Consumer resale items use SalesReceivable and never use Stripe Connect.
//
// Console company access to the Inquiry is established by the company-scoped
// Inquiry detail query before this usecase is invoked. Inside this usecase,
// CompanyID is additionally matched to SellerSnapshot.CompanyID only for List
// items because a resale seller is an Avatar/User, not a Company.
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
// The frontend sends only the Inquiry ID through the route. CompanyID and
// MemberID come from authenticated Console context.
//
// Refund amount, Order ID, item index, seller identity, Settlement ID,
// SalesReceivable ID, tax rate and monetary amounts are never accepted from the
// frontend.
func (uc *ReturnReceiptUsecase) ReceiveReturn(
	ctx context.Context,
	in ReceiveReturnInput,
) (ReturnReceiptResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ReturnReceiptResult{}, err
	}

	inquiryID := strings.TrimSpace(in.InquiryID)
	if inquiryID == "" {
		return ReturnReceiptResult{}, inquirydom.ErrInvalidID
	}

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return ReturnReceiptResult{}, ErrReturnReceiptInvalidCompanyID
	}

	memberID := strings.TrimSpace(in.MemberID)
	if memberID == "" {
		return ReturnReceiptResult{}, ErrReturnReceiptInvalidMemberID
	}

	inquiry, err := uc.inquiryRepo.GetByID(ctx, inquiryID)
	if err != nil {
		return ReturnReceiptResult{}, err
	}

	if inquiry.ID != inquiryID || inquiry.DeletedAt != nil {
		return ReturnReceiptResult{}, inquirydom.ErrNotFound
	}

	if inquiry.InquiryType != inquirydom.InquiryTypeReturnUnopened {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, ErrReturnReceiptInvalidInquiryType
	}

	if inquiry.Status == inquirydom.InquiryStatusClosed {
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

	order, err := uc.orderService.GetByID(ctx, orderID)
	if err != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
		}, err
	}

	if order.ID != orderID || order.AvatarID != inquiry.AvatarID {
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

	if err := validateReturnReceiptTargetSeller(
		companyID,
		targetItem,
	); err != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, err
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
	// A valid scan means the token transfer/opening boundary has already been
	// crossed. Even if Inquiry promotion to return_opened failed concurrently,
	// authoritative Order state prevents an unopened refund from proceeding.
	if targetItem.Transferred ||
		targetItem.TokenTransferVerifiedAt != nil {
		return ReturnReceiptResult{
			Inquiry: inquiry,
			Order:   order,
		}, ErrReturnReceiptReturnNotUnopened
	}

	// A resolved return_unopened Inquiry is valid here only when Order return
	// completion has already succeeded. This prevents generic Inquiry resolution
	// from bypassing the financial return workflow.
	if inquiry.Status == inquirydom.InquiryStatusResolved &&
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

	result.FinanciallyCompleted = refund.IsFinanciallyCompleted()
	if !result.FinanciallyCompleted {
		// Stripe may have accepted the purchaser Refund but still report pending
		// or requires_action. Do not complete Order or Inquiry until the complete
		// Refund aggregate confirms seller-side and purchaser-side completion.
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
		return result, ErrReturnReceiptOrderCompletionMismatch
	}

	result.OrderCompleted = true

	// The deterministic reply ID makes retrying this flow safe if a later step
	// fails after the refund completion reply has already been created.
	replyContent, err := buildRefundCompletionReplyContent(refund)
	if err != nil {
		return result, err
	}

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
		return result, ErrReturnReceiptInquiryResolutionMismatch
	}

	result.InquiryResolved = true

	// Always ensure notification delivery, including idempotent retries after
	// Order and Inquiry completion.
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

// validateReturnReceiptTargetSeller validates the immutable seller snapshot
// according to the Order item type.
//
// List:
//   - Console Company must equal the seller Company.
//   - Company/Account/Stripe account identity is required.
//   - resale seller fields must be empty.
//
// Resale:
//   - seller is Avatar/User/PayoutAccount, not the authenticated Console Company.
//   - Company access to the Inquiry is established by the company-scoped Inquiry
//     detail query before this usecase is invoked.
//   - StripeAccountID must be empty.
func validateReturnReceiptTargetSeller(
	companyID string,
	targetItem orderdom.OrderItemSnapshot,
) error {
	seller := targetItem.SellerSnapshot

	switch targetItem.Type {
	case orderdom.OrderItemTypeList:
		if seller.CompanyID != companyID {
			return ErrReturnReceiptCompanyMismatch
		}

		if seller.BrandID == "" ||
			seller.CompanyID == "" ||
			seller.AccountID == "" ||
			seller.StripeAccountID == "" ||
			seller.AvatarID != "" ||
			seller.UserID != "" ||
			seller.PayoutAccountID != "" {
			return ErrReturnReceiptOrderMismatch
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
			return ErrReturnReceiptOrderMismatch
		}

	default:
		return ErrReturnReceiptOrderMismatch
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

	if refund.InquiryID != inquiry.ID {
		return ErrReturnReceiptRefundMismatch
	}

	if refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != itemIndex {
		return ErrReturnReceiptRefundMismatch
	}

	if itemIndex < 0 || itemIndex >= len(order.Items) {
		return ErrReturnReceiptRefundMismatch
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
			return ErrReturnReceiptRefundMismatch
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
			return ErrReturnReceiptRefundMismatch
		}

		expectedSalesReceivableID, err := salesreceivabledom.NewID(
			order.ID,
			itemIndex,
		)
		if err != nil ||
			refund.SalesReceivableID != expectedSalesReceivableID {
			return ErrReturnReceiptRefundMismatch
		}

	default:
		return ErrReturnReceiptRefundMismatch
	}

	if refund.Currency != refunddom.CurrencyJPY {
		return ErrReturnReceiptRefundMismatch
	}

	// Unopened return refunds merchandise and merchandise consumption tax only.
	// Shipping and shipping consumption tax are intentionally excluded.
	expectedAmount, err := refunddom.CalculateOrderItemRefundAmount(
		order,
		itemIndex,
	)
	if err != nil {
		return err
	}

	if refund.MerchandiseAmount != expectedAmount.MerchandiseAmount ||
		refund.MerchandiseTaxAmount != expectedAmount.MerchandiseTaxAmount ||
		refund.RefundAmount != expectedAmount.RefundAmount {
		return ErrReturnReceiptRefundMismatch
	}

	return nil
}
