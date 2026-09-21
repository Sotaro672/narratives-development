// backend/internal/application/usecase/resale_trade_return_proposal_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrResaleTradeReturnProposalNotConfigured = errors.New(
		"resale trade return proposal: usecase is not configured",
	)
	ErrResaleTradeReturnProposalInvalidSeller = errors.New(
		"resale trade return proposal: invalid seller",
	)
	ErrResaleTradeReturnProposalTradeMismatch = errors.New(
		"resale trade return proposal: trade does not match order item",
	)
	ErrResaleTradeReturnProposalOrderNotPaid = errors.New(
		"resale trade return proposal: order is not paid",
	)
	ErrResaleTradeReturnProposalNotEligible = errors.New(
		"resale trade return proposal: trade is not eligible for return proposal",
	)
	ErrResaleTradeReturnProposalRefundAmountExceedsMaximum = errors.New(
		"resale trade return proposal: refund amount exceeds maximum",
	)
)

const resaleTradeReturnProposalSystemMessageIDPrefix = "return-proposal-"

// ResaleTradeReturnProposalUsecase records the seller's response to a buyer's
// return consultation for one Avatar-to-Avatar Resale Trade.
//
// Responsibilities:
//   - authenticate the seller against the persisted Trade
//   - confirm the authoritative Order item is paid, dispatched and not transferred
//   - load the Trade ReturnAgreement
//   - validate the seller's proposed refund against the authoritative Order
//   - persist ReturnAgreement.Propose()
//   - create an idempotent system timeline message
//
// Seller agreement:
//
//	discussing -> proposed
//
// Seller disagreement:
//
//	discussing -> discussing
//
// A proposal with agreement=agree must later be explicitly accepted by the
// buyer before any return shipment or refund processing may begin.
//
// This usecase does not mutate legacy Order return-request fields.
type ResaleTradeReturnProposalUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	orderRepo           orderdom.Repository
	messageRepo         tradedom.MessageRepository

	now func() time.Time
}

type NewResaleTradeReturnProposalUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	OrderRepository           orderdom.Repository
	MessageRepository         tradedom.MessageRepository
}

func NewResaleTradeReturnProposalUsecase(
	in NewResaleTradeReturnProposalUsecaseInput,
) *ResaleTradeReturnProposalUsecase {
	return &ResaleTradeReturnProposalUsecase{
		tradeRepo:           in.TradeRepository,
		returnAgreementRepo: in.ReturnAgreementRepository,
		orderRepo:           in.OrderRepository,
		messageRepo:         in.MessageRepository,
		now:                 time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (u *ResaleTradeReturnProposalUsecase) SetNowFunc(
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

type CreateResaleTradeReturnProposalInput struct {
	TradeID        string
	SellerAvatarID string

	Agreement         tradedom.ReturnProposalAgreement
	ReturnRequirement tradedom.ReturnRequirement
	RefundAmount      int
}

type ResaleTradeReturnProposalResult struct {
	Trade     tradedom.Trade
	Order     orderdom.Order
	Item      orderdom.OrderItemSnapshot
	Agreement tradedom.ReturnAgreement
	Proposal  tradedom.ReturnProposal

	MerchandiseRefundMaxAmount int

	Changed              bool
	SystemMessageEnsured bool
}

// ============================================================
// Create proposal
// ============================================================

// Create records or idempotently confirms the seller's latest return proposal.
//
// agreement=agree:
//
//   - ReturnRequirement must be required or not_required.
//   - RefundAmount must be greater than 0.
//   - RefundAmount must not exceed the authoritative merchandise refund maximum.
//   - ReturnAgreement moves from discussing to proposed.
//
// agreement=disagree:
//
//   - ReturnRequirement must be empty.
//   - RefundAmount must be 0.
//   - ReturnAgreement remains discussing.
//
// Repeating the currently persisted proposal with the same contents is
// idempotent. This is important when ReturnAgreement was persisted but system
// message creation failed and the HTTP request is retried.
//
// A previously buyer-rejected proposal is not considered an idempotent match;
// the seller may submit the same conditions again as a new proposal.
func (u *ResaleTradeReturnProposalUsecase) Create(
	ctx context.Context,
	in CreateResaleTradeReturnProposalInput,
) (ResaleTradeReturnProposalResult, error) {
	if err := u.validateConfigured(); err != nil {
		return ResaleTradeReturnProposalResult{}, err
	}

	tradeID := strings.TrimSpace(in.TradeID)
	if tradeID == "" {
		return ResaleTradeReturnProposalResult{},
			tradedom.ErrInvalidID
	}

	sellerAvatarID := strings.TrimSpace(in.SellerAvatarID)
	if sellerAvatarID == "" {
		return ResaleTradeReturnProposalResult{},
			ErrResaleTradeReturnProposalInvalidSeller
	}

	if !tradedom.IsValidReturnProposalAgreement(in.Agreement) {
		return ResaleTradeReturnProposalResult{},
			tradedom.ErrInvalidReturnProposalAgreement
	}

	switch in.Agreement {
	case tradedom.ReturnProposalAgreementAgree:
		if !tradedom.IsValidReturnRequirement(
			in.ReturnRequirement,
		) {
			return ResaleTradeReturnProposalResult{},
				tradedom.ErrInvalidReturnRequirement
		}
		if in.RefundAmount <= 0 {
			return ResaleTradeReturnProposalResult{},
				tradedom.ErrInvalidReturnRefundAmount
		}

	case tradedom.ReturnProposalAgreementDisagree:
		if in.ReturnRequirement != "" {
			return ResaleTradeReturnProposalResult{},
				tradedom.ErrInvalidReturnRequirement
		}
		if in.RefundAmount != 0 {
			return ResaleTradeReturnProposalResult{},
				tradedom.ErrInvalidReturnRefundAmount
		}
	}

	trade, err := u.tradeRepo.GetByID(
		ctx,
		tradeID,
	)
	if err != nil {
		return ResaleTradeReturnProposalResult{}, err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" {
		return ResaleTradeReturnProposalResult{},
			tradedom.ErrNotFound
	}

	if trade.SellerAvatarID != sellerAvatarID {
		return ResaleTradeReturnProposalResult{},
			tradedom.ErrNotFound
	}

	if trade.Status == tradedom.StatusClosed {
		return ResaleTradeReturnProposalResult{
			Trade: trade,
		}, tradedom.ErrTradeAlreadyClosed
	}
	if trade.Status != tradedom.StatusActive {
		return ResaleTradeReturnProposalResult{
			Trade: trade,
		}, tradedom.ErrInvalidStatus
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return ResaleTradeReturnProposalResult{
			Trade: trade,
		}, err
	}

	item, err := validateResaleTradeReturnProposalTarget(
		order,
		trade,
		sellerAvatarID,
	)
	if err != nil {
		return ResaleTradeReturnProposalResult{
			Trade: trade,
			Order: order,
		}, err
	}

	result := ResaleTradeReturnProposalResult{
		Trade: trade,
		Order: order,
		Item:  item,
	}

	if !order.Paid {
		return result,
			ErrResaleTradeReturnProposalOrderNotPaid
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.Transferred {
		return result,
			ErrResaleTradeReturnProposalNotEligible
	}

	refundAmountSummary, err :=
		refunddom.CalculateOrderItemRefundAmount(
			order,
			trade.OrderItemIndex,
		)
	if err != nil {
		return result, err
	}

	result.MerchandiseRefundMaxAmount =
		refundAmountSummary.RefundAmount

	if in.Agreement ==
		tradedom.ReturnProposalAgreementAgree &&
		in.RefundAmount >
			result.MerchandiseRefundMaxAmount {
		return result,
			ErrResaleTradeReturnProposalRefundAmountExceedsMaximum
	}

	agreement, err :=
		u.returnAgreementRepo.GetByTradeID(
			ctx,
			tradeID,
		)
	if err != nil {
		return result, err
	}

	if agreement.ID != tradeID ||
		agreement.TradeID != tradeID {
		return result,
			tradedom.ErrReturnAgreementConflict
	}

	// If the same active proposal already exists, treat the request as an
	// idempotent retry. This also repairs a previous request that persisted the
	// aggregate but failed before creating its system message.
	if sameResaleTradeReturnProposal(
		agreement,
		in.Agreement,
		in.ReturnRequirement,
		in.RefundAmount,
	) {
		result.Agreement = agreement
		result.Proposal = *agreement.Proposal

		ensured, ensureErr :=
			u.ensureSystemMessage(
				ctx,
				tradeID,
				*agreement.Proposal,
			)
		if ensureErr != nil {
			return result, ensureErr
		}

		result.SystemMessageEnsured = ensured
		return result, nil
	}

	if agreement.Status !=
		tradedom.ReturnStatusDiscussing {
		return result,
			tradedom.ErrReturnProposalNotAllowed
	}

	now := u.nowUTC()

	if err := agreement.Propose(
		"",
		in.Agreement,
		in.ReturnRequirement,
		in.RefundAmount,
		now,
	); err != nil {
		return result, err
	}

	updated, err := u.returnAgreementRepo.Update(
		ctx,
		tradeID,
		agreement,
	)
	if err != nil {
		return result, err
	}

	if updated.Proposal == nil ||
		strings.TrimSpace(updated.Proposal.ID) == "" {
		return result,
			tradedom.ErrReturnAgreementConflict
	}

	result.Agreement = updated
	result.Proposal = *updated.Proposal
	result.Changed = true

	ensured, err := u.ensureSystemMessage(
		ctx,
		tradeID,
		*updated.Proposal,
	)
	if err != nil {
		// ReturnAgreement has already been persisted. Returning the error allows
		// the caller to retry the same request. sameResaleTradeReturnProposal
		// then detects the persisted proposal and only retries message creation.
		return result, err
	}

	result.SystemMessageEnsured = ensured
	return result, nil
}

// ============================================================
// System message
// ============================================================

func (u *ResaleTradeReturnProposalUsecase) ensureSystemMessage(
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
		resaleTradeReturnProposalSystemMessageIDPrefix+
			proposalID,
		tradeID,
		buildResaleTradeReturnProposalSystemMessage(
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

func buildResaleTradeReturnProposalSystemMessage(
	proposal tradedom.ReturnProposal,
) string {
	switch proposal.Agreement {
	case tradedom.ReturnProposalAgreementAgree:
		return fmt.Sprintf(
			"出品者が返品条件を提示しました。返金額: %d円、商品返送: %s",
			proposal.RefundAmount,
			resaleTradeReturnRequirementLabel(
				proposal.ReturnRequirement,
			),
		)

	case tradedom.ReturnProposalAgreementDisagree:
		return "出品者が返品に合意しませんでした。メッセージで相談を継続してください。"

	default:
		return "出品者が返品相談に回答しました。"
	}
}

func resaleTradeReturnRequirementLabel(
	requirement tradedom.ReturnRequirement,
) string {
	switch requirement {
	case tradedom.ReturnRequirementRequired:
		return "必要"

	case tradedom.ReturnRequirementNotRequired:
		return "不要"

	default:
		return "-"
	}
}

// ============================================================
// Validation
// ============================================================

func validateResaleTradeReturnProposalTarget(
	order orderdom.Order,
	trade tradedom.Trade,
	sellerAvatarID string,
) (orderdom.OrderItemSnapshot, error) {
	if order.ID == "" ||
		order.ID != trade.OrderID ||
		order.AvatarID == "" ||
		order.AvatarID != trade.BuyerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalTradeMismatch
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(order.Items) {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalTradeMismatch
	}

	item := order.Items[trade.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale ||
		item.ResaleID == "" {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalTradeMismatch
	}

	if item.SellerSnapshot.AvatarID == "" ||
		item.SellerSnapshot.AvatarID !=
			trade.SellerAvatarID ||
		item.SellerSnapshot.AvatarID !=
			sellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnProposalTradeMismatch
	}

	return item, nil
}

func sameResaleTradeReturnProposal(
	agreement tradedom.ReturnAgreement,
	proposalAgreement tradedom.ReturnProposalAgreement,
	returnRequirement tradedom.ReturnRequirement,
	refundAmount int,
) bool {
	if agreement.Proposal == nil ||
		agreement.Proposal.ID == "" ||
		agreement.Proposal.RejectedAt != nil {
		return false
	}

	proposal := agreement.Proposal

	if proposal.Agreement != proposalAgreement ||
		proposal.ReturnRequirement != returnRequirement ||
		proposal.RefundAmount != refundAmount {
		return false
	}

	switch proposalAgreement {
	case tradedom.ReturnProposalAgreementAgree:
		return agreement.Status ==
			tradedom.ReturnStatusProposed

	case tradedom.ReturnProposalAgreementDisagree:
		return agreement.Status ==
			tradedom.ReturnStatusDiscussing

	default:
		return false
	}
}

// ============================================================
// Internal
// ============================================================

func (u *ResaleTradeReturnProposalUsecase) validateConfigured() error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.returnAgreementRepo == nil ||
		u.orderRepo == nil ||
		u.messageRepo == nil ||
		u.now == nil {
		return ErrResaleTradeReturnProposalNotConfigured
	}

	return nil
}

func (u *ResaleTradeReturnProposalUsecase) nowUTC() time.Time {
	return u.now().UTC()
}
