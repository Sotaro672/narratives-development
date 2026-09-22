// backend/internal/application/usecase/resale_trade_return_proposal_response_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrResaleTradeReturnProposalResponseNotConfigured = errors.New(
		"resale trade return proposal response: usecase is not configured",
	)
	ErrResaleTradeReturnProposalResponseInvalidBuyer = errors.New(
		"resale trade return proposal response: invalid buyer",
	)
	ErrResaleTradeReturnProposalResponseTradeMismatch = errors.New(
		"resale trade return proposal response: trade does not match order item",
	)
	ErrResaleTradeReturnProposalResponseOrderNotPaid = errors.New(
		"resale trade return proposal response: order is not paid",
	)
	ErrResaleTradeReturnProposalResponseNotEligible = errors.New(
		"resale trade return proposal response: trade is not eligible for return proposal response",
	)
	ErrResaleTradeReturnProposalResponseProposalMismatch = errors.New(
		"resale trade return proposal response: proposal does not match current proposal",
	)
	ErrResaleTradeReturnProposalResponseRefundAmountInvalid = errors.New(
		"resale trade return proposal response: proposal refund amount is invalid",
	)
)

const (
	resaleTradeReturnProposalAcceptedSystemMessageIDPrefix = "return-proposal-accepted-"
	resaleTradeReturnProposalRejectedSystemMessageIDPrefix = "return-proposal-rejected-"
)

// ResaleTradeReturnProposalRefundService executes or resumes the agreed refund
// when the accepted proposal does not require physical return.
type ResaleTradeReturnProposalRefundService interface {
	Refund(
		ctx context.Context,
		in RefundResaleTradeReturnInput,
	) (ResaleTradeReturnRefundResult, error)
}

// ResaleTradeReturnProposalResponseUsecase records the buyer's explicit response
// to the seller's latest agreed return proposal.
//
// Responsibilities:
//   - authenticate the buyer against the persisted Trade
//   - confirm the Trade represents an Avatar-to-Avatar Resale transaction
//   - confirm the authoritative Order item is paid, dispatched and not transferred
//   - confirm proposalId identifies the current seller proposal
//   - revalidate the proposed refund amount against the authoritative Order
//   - accept or reject the current ReturnAgreement proposal
//   - persist the ReturnAgreement transition
//   - create an idempotent system timeline message
//   - start or resume refund processing immediately when physical return is not required
//
// Accept:
//
//	proposed -> agreed
//
// For returnRequirement=not_required:
//
//	proposed
//	  -> agreed
//	  -> refund_processing
//	  -> completed
//
// For returnRequirement=required, the accepted proposal remains agreed until the
// physical return flow proceeds.
//
// Reject:
//
//	proposed -> discussing
//
// An accepted proposal becomes the immutable return/refund condition used by
// later return-shipment and refund processing.
//
// This usecase does not mutate legacy Order return-request fields.
type ResaleTradeReturnProposalResponseUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	orderRepo           orderdom.Repository
	messageRepo         tradedom.MessageRepository
	returnRefundService ResaleTradeReturnProposalRefundService

	now func() time.Time
}

type NewResaleTradeReturnProposalResponseUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	OrderRepository           orderdom.Repository
	MessageRepository         tradedom.MessageRepository
	ReturnRefundService       ResaleTradeReturnProposalRefundService
}

func NewResaleTradeReturnProposalResponseUsecase(
	in NewResaleTradeReturnProposalResponseUsecaseInput,
) *ResaleTradeReturnProposalResponseUsecase {
	return &ResaleTradeReturnProposalResponseUsecase{
		tradeRepo:           in.TradeRepository,
		returnAgreementRepo: in.ReturnAgreementRepository,
		orderRepo:           in.OrderRepository,
		messageRepo:         in.MessageRepository,
		returnRefundService: in.ReturnRefundService,
		now:                 time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (u *ResaleTradeReturnProposalResponseUsecase) SetNowFunc(
	now func() time.Time,
) {
	if u == nil || now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Input / Result
// ============================================================

type RespondResaleTradeReturnProposalInput struct {
	TradeID       string
	ProposalID    string
	BuyerAvatarID string
}

type ResaleTradeReturnProposalResponseResult struct {
	Trade     tradedom.Trade
	Order     orderdom.Order
	Item      orderdom.OrderItemSnapshot
	Agreement tradedom.ReturnAgreement
	Proposal  tradedom.ReturnProposal

	RefundResult *ResaleTradeReturnRefundResult

	Changed              bool
	SystemMessageEnsured bool
}

// ============================================================
// Accept
// ============================================================

// Accept records explicit buyer acceptance of the seller's current proposal.
//
// Repeating acceptance for the same proposal is idempotent when that proposal
// has already been accepted. This allows recovery when ReturnAgreement,
// system-message creation, or no-physical-return refund processing succeeded
// only partially.
//
// For returnRequirement=not_required, acceptance immediately starts or resumes
// ResaleTradeReturnRefundUsecase. No ReturnShipment is created or consulted.
func (u *ResaleTradeReturnProposalResponseUsecase) Accept(
	ctx context.Context,
	in RespondResaleTradeReturnProposalInput,
) (ResaleTradeReturnProposalResponseResult, error) {
	result, agreement, err := u.loadAndValidate(
		ctx,
		in,
	)
	if err != nil {
		return result, err
	}

	// The same proposal may already have been accepted by a previous request
	// whose HTTP response, system-message creation, or refund processing failed.
	//
	// AgreedAt is authoritative evidence that this proposal passed through
	// AcceptProposal. Later states such as return_shipped, refund_processing or
	// completed are therefore also valid idempotent retries.
	if isAcceptedResaleTradeReturnProposal(
		agreement,
		in.ProposalID,
	) {
		result.Agreement = agreement
		result.Proposal = *agreement.Proposal

		return u.finalizeAcceptedProposal(
			ctx,
			in,
			result,
		)
	}

	if agreement.Status != tradedom.ReturnStatusProposed {
		return result,
			tradedom.ErrReturnProposalCannotBeAccepted
	}

	if err := agreement.AcceptProposal(
		u.nowUTC(),
	); err != nil {
		return result, err
	}

	updated, err := u.returnAgreementRepo.Update(
		ctx,
		result.Trade.ID,
		agreement,
	)
	if err != nil {
		return result, err
	}

	if !isAcceptedResaleTradeReturnProposal(
		updated,
		in.ProposalID,
	) {
		return result,
			tradedom.ErrReturnAgreementConflict
	}

	result.Agreement = updated
	result.Proposal = *updated.Proposal
	result.Changed = true

	return u.finalizeAcceptedProposal(
		ctx,
		in,
		result,
	)
}

// finalizeAcceptedProposal performs the retry-safe side effects that follow
// buyer acceptance.
//
// Order:
//
//  1. ensure deterministic accepted system message
//  2. for not_required only, execute/resume financial refund
//
// If the system message fails, the refund is not started yet. A retry of Accept
// enters the accepted-proposal branch and resumes from the same point.
//
// If the refund partially succeeds and then fails, a retry reaches the same
// deterministic ItemRefund flow through ResaleTradeReturnRefundUsecase.
func (u *ResaleTradeReturnProposalResponseUsecase) finalizeAcceptedProposal(
	ctx context.Context,
	in RespondResaleTradeReturnProposalInput,
	result ResaleTradeReturnProposalResponseResult,
) (ResaleTradeReturnProposalResponseResult, error) {
	ensured, err := u.ensureAcceptedSystemMessage(
		ctx,
		result.Trade.ID,
		result.Proposal,
	)
	if err != nil {
		return result, err
	}

	result.SystemMessageEnsured = ensured

	if result.Proposal.ReturnRequirement !=
		tradedom.ReturnRequirementNotRequired {
		return result, nil
	}

	// A dispute stops automatic financial progression. The accepted proposal
	// remains historically accepted, but dispute resolution owns further action.
	if result.Agreement.Status ==
		tradedom.ReturnStatusDisputed {
		return result, nil
	}

	switch result.Agreement.Status {
	case tradedom.ReturnStatusAgreed,
		tradedom.ReturnStatusRefundProcessing,
		tradedom.ReturnStatusCompleted:

	default:
		return result, nil
	}

	if u.returnRefundService == nil {
		return result,
			ErrResaleTradeReturnProposalResponseNotConfigured
	}

	refundResult, err :=
		u.returnRefundService.Refund(
			ctx,
			RefundResaleTradeReturnInput{
				TradeID:       result.Trade.ID,
				BuyerAvatarID: in.BuyerAvatarID,
			},
		)

	result.RefundResult = &refundResult

	// Refund may have persisted ReturnAgreement progress before a later
	// financial or notification operation returned an error. Preserve the
	// newest known state in the response so callers do not receive stale
	// "agreed" state after the aggregate has advanced.
	if refundResult.Agreement.ID != "" {
		result.Agreement =
			refundResult.Agreement
	}
	if refundResult.Proposal.ID != "" {
		result.Proposal =
			refundResult.Proposal
	}

	if err != nil {
		return result, err
	}

	return result, nil
}

// ============================================================
// Reject
// ============================================================

// Reject records buyer rejection of the seller's current proposal.
//
// Repeating rejection for the same proposal is idempotent while that rejected
// proposal remains the latest persisted proposal.
//
// Once the seller creates another proposal, the previous proposalId no longer
// matches and cannot be used to reject the new proposal.
func (u *ResaleTradeReturnProposalResponseUsecase) Reject(
	ctx context.Context,
	in RespondResaleTradeReturnProposalInput,
) (ResaleTradeReturnProposalResponseResult, error) {
	result, agreement, err := u.loadAndValidate(
		ctx,
		in,
	)
	if err != nil {
		return result, err
	}

	if isRejectedResaleTradeReturnProposal(
		agreement,
		in.ProposalID,
	) {
		result.Agreement = agreement
		result.Proposal = *agreement.Proposal

		ensured, ensureErr :=
			u.ensureRejectedSystemMessage(
				ctx,
				result.Trade.ID,
				*agreement.Proposal,
			)
		if ensureErr != nil {
			return result, ensureErr
		}

		result.SystemMessageEnsured = ensured
		return result, nil
	}

	if agreement.Status != tradedom.ReturnStatusProposed {
		return result,
			tradedom.ErrReturnProposalCannotBeRejected
	}

	now := u.nowUTC()

	if err := agreement.RejectProposal(now); err != nil {
		return result, err
	}

	updated, err := u.returnAgreementRepo.Update(
		ctx,
		result.Trade.ID,
		agreement,
	)
	if err != nil {
		return result, err
	}

	if !isRejectedResaleTradeReturnProposal(
		updated,
		in.ProposalID,
	) {
		return result,
			tradedom.ErrReturnAgreementConflict
	}

	result.Agreement = updated
	result.Proposal = *updated.Proposal
	result.Changed = true

	ensured, err := u.ensureRejectedSystemMessage(
		ctx,
		result.Trade.ID,
		*updated.Proposal,
	)
	if err != nil {
		// ReturnAgreement has already been persisted. A retry of Reject with
		// the same proposalId retries only deterministic message creation.
		return result, err
	}

	result.SystemMessageEnsured = ensured
	return result, nil
}

// ============================================================
// Load / validation
// ============================================================

func (u *ResaleTradeReturnProposalResponseUsecase) loadAndValidate(
	ctx context.Context,
	in RespondResaleTradeReturnProposalInput,
) (
	ResaleTradeReturnProposalResponseResult,
	tradedom.ReturnAgreement,
	error,
) {
	if err := u.validateConfigured(); err != nil {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			err
	}

	tradeID := strings.TrimSpace(in.TradeID)
	if tradeID == "" {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidID
	}

	proposalID := strings.TrimSpace(in.ProposalID)
	if proposalID == "" {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnProposalID
	}

	buyerAvatarID := strings.TrimSpace(in.BuyerAvatarID)
	if buyerAvatarID == "" {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseInvalidBuyer
	}

	trade, err := u.tradeRepo.GetByID(
		ctx,
		tradeID,
	)
	if err != nil {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrNotFound
	}

	if trade.BuyerAvatarID != buyerAvatarID {
		return ResaleTradeReturnProposalResponseResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrNotFound
	}

	if trade.Status == tradedom.StatusClosed {
		return ResaleTradeReturnProposalResponseResult{
				Trade: trade,
			},
			tradedom.ReturnAgreement{},
			tradedom.ErrTradeAlreadyClosed
	}
	if trade.Status != tradedom.StatusActive {
		return ResaleTradeReturnProposalResponseResult{
				Trade: trade,
			},
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidStatus
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return ResaleTradeReturnProposalResponseResult{
				Trade: trade,
			},
			tradedom.ReturnAgreement{},
			err
	}

	item, err :=
		validateResaleTradeReturnProposalResponseTarget(
			order,
			trade,
			buyerAvatarID,
		)
	if err != nil {
		return ResaleTradeReturnProposalResponseResult{
				Trade: trade,
				Order: order,
			},
			tradedom.ReturnAgreement{},
			err
	}

	result := ResaleTradeReturnProposalResponseResult{
		Trade: trade,
		Order: order,
		Item:  item,
	}

	if !order.Paid {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseOrderNotPaid
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.Transferred {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseNotEligible
	}

	agreement, err :=
		u.returnAgreementRepo.GetByTradeID(
			ctx,
			tradeID,
		)
	if err != nil {
		return result,
			tradedom.ReturnAgreement{},
			err
	}

	if agreement.ID != tradeID ||
		agreement.TradeID != tradeID {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementConflict
	}

	if agreement.Proposal == nil {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrReturnProposalNotFound
	}

	if strings.TrimSpace(
		agreement.Proposal.ID,
	) != proposalID {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseProposalMismatch
	}

	if agreement.Proposal.Agreement !=
		tradedom.ReturnProposalAgreementAgree {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseProposalMismatch
	}

	if !tradedom.IsValidReturnRequirement(
		agreement.Proposal.ReturnRequirement,
	) {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnRequirement
	}

	if agreement.Proposal.RefundAmount <= 0 {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnRefundAmount
	}

	refundAmountSummary, err :=
		refunddom.CalculateOrderItemRefundAmount(
			order,
			trade.OrderItemIndex,
		)
	if err != nil {
		return result,
			tradedom.ReturnAgreement{},
			err
	}

	if agreement.Proposal.RefundAmount >
		refundAmountSummary.RefundAmount {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnProposalResponseRefundAmountInvalid
	}

	result.Agreement = agreement
	result.Proposal = *agreement.Proposal

	return result, agreement, nil
}

func validateResaleTradeReturnProposalResponseTarget(
	order orderdom.Order,
	trade tradedom.Trade,
	buyerAvatarID string,
) (orderdom.OrderItemSnapshot, error) {
	if order.ID == "" ||
		order.ID != trade.OrderID ||
		order.AvatarID == "" ||
		order.AvatarID != trade.BuyerAvatarID ||
		order.AvatarID != buyerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalResponseTradeMismatch
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(order.Items) {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalResponseTradeMismatch
	}

	item := order.Items[trade.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale ||
		item.ResaleID == "" {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalResponseTradeMismatch
	}

	if item.SellerSnapshot.AvatarID == "" ||
		item.SellerSnapshot.AvatarID !=
			trade.SellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalResponseTradeMismatch
	}

	return item, nil
}

// ============================================================
// Idempotency
// ============================================================

func isAcceptedResaleTradeReturnProposal(
	agreement tradedom.ReturnAgreement,
	proposalID string,
) bool {
	proposalID = strings.TrimSpace(proposalID)

	if proposalID == "" ||
		agreement.Proposal == nil ||
		agreement.Proposal.ID != proposalID ||
		agreement.Proposal.Agreement !=
			tradedom.ReturnProposalAgreementAgree ||
		agreement.Proposal.RejectedAt != nil ||
		agreement.AgreedAt == nil ||
		agreement.AgreedAt.IsZero() {
		return false
	}

	switch agreement.Status {
	case tradedom.ReturnStatusAgreed,
		tradedom.ReturnStatusReturnShipped,
		tradedom.ReturnStatusReturnReceived,
		tradedom.ReturnStatusRefundProcessing,
		tradedom.ReturnStatusCompleted:
		return true

	case tradedom.ReturnStatusDisputed:
		// A dispute may be opened after buyer acceptance. AgreedAt proves that
		// this proposal was accepted before entering the disputed state.
		return true

	default:
		return false
	}
}

func isRejectedResaleTradeReturnProposal(
	agreement tradedom.ReturnAgreement,
	proposalID string,
) bool {
	proposalID = strings.TrimSpace(proposalID)

	if proposalID == "" ||
		agreement.Proposal == nil ||
		agreement.Proposal.ID != proposalID ||
		agreement.Proposal.Agreement !=
			tradedom.ReturnProposalAgreementAgree ||
		agreement.Proposal.RejectedAt == nil ||
		agreement.Proposal.RejectedAt.IsZero() ||
		agreement.AgreedAt != nil {
		return false
	}

	switch agreement.Status {
	case tradedom.ReturnStatusDiscussing,
		tradedom.ReturnStatusDisputed:
		return true

	default:
		return false
	}
}

// ============================================================
// System messages
// ============================================================

func (u *ResaleTradeReturnProposalResponseUsecase) ensureAcceptedSystemMessage(
	ctx context.Context,
	tradeID string,
	proposal tradedom.ReturnProposal,
) (bool, error) {
	proposalID := strings.TrimSpace(proposal.ID)
	if proposalID == "" {
		return false,
			tradedom.ErrInvalidReturnProposalID
	}

	message, err := tradedom.NewSystemMessageForCreate(
		resaleTradeReturnProposalAcceptedSystemMessageIDPrefix+
			proposalID,
		tradeID,
		buildResaleTradeReturnProposalAcceptedSystemMessage(
			proposal,
		),
	)
	if err != nil {
		return false, err
	}

	message.CreatedAt = u.nowUTC()

	_, err = u.messageRepo.Create(
		ctx,
		message,
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrMessageAlreadyExists,
		) {
			return true, nil
		}

		return false, err
	}

	return true, nil
}

func (u *ResaleTradeReturnProposalResponseUsecase) ensureRejectedSystemMessage(
	ctx context.Context,
	tradeID string,
	proposal tradedom.ReturnProposal,
) (bool, error) {
	proposalID := strings.TrimSpace(proposal.ID)
	if proposalID == "" {
		return false,
			tradedom.ErrInvalidReturnProposalID
	}

	message, err := tradedom.NewSystemMessageForCreate(
		resaleTradeReturnProposalRejectedSystemMessageIDPrefix+
			proposalID,
		tradeID,
		"購入者が提示された返品条件に同意しませんでした。メッセージで相談を継続してください。",
	)
	if err != nil {
		return false, err
	}

	message.CreatedAt = u.nowUTC()

	_, err = u.messageRepo.Create(
		ctx,
		message,
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrMessageAlreadyExists,
		) {
			return true, nil
		}

		return false, err
	}

	return true, nil
}

func buildResaleTradeReturnProposalAcceptedSystemMessage(
	proposal tradedom.ReturnProposal,
) string {
	switch proposal.ReturnRequirement {
	case tradedom.ReturnRequirementRequired:
		return "購入者が提示された返品条件に同意しました。PUDO匿名返品の手続きへ進みます。"

	case tradedom.ReturnRequirementNotRequired:
		return "購入者が提示された返品条件に同意しました。商品返送なしで返金手続きへ進みます。"

	default:
		return "購入者が提示された返品条件に同意しました。"
	}
}

// ============================================================
// Internal
// ============================================================

func (u *ResaleTradeReturnProposalResponseUsecase) validateConfigured() error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.returnAgreementRepo == nil ||
		u.orderRepo == nil ||
		u.messageRepo == nil ||
		u.now == nil {
		return ErrResaleTradeReturnProposalResponseNotConfigured
	}

	return nil
}

func (u *ResaleTradeReturnProposalResponseUsecase) nowUTC() time.Time {
	return u.now().UTC()
}
