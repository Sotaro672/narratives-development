// backend/internal/application/usecase/resale_trade_return_shipment_usecase.go
package usecase

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrResaleTradeReturnShipmentNotConfigured = errors.New(
		"resale trade return shipment: usecase is not configured",
	)
	ErrResaleTradeReturnShipmentInvalidBuyer = errors.New(
		"resale trade return shipment: invalid buyer",
	)
	ErrResaleTradeReturnShipmentTradeMismatch = errors.New(
		"resale trade return shipment: trade does not match order item",
	)
	ErrResaleTradeReturnShipmentOrderNotPaid = errors.New(
		"resale trade return shipment: order is not paid",
	)
	ErrResaleTradeReturnShipmentNotEligible = errors.New(
		"resale trade return shipment: trade is not eligible for return shipment",
	)
	ErrResaleTradeReturnShipmentAgreementNotReady = errors.New(
		"resale trade return shipment: return agreement is not ready for shipment",
	)
	ErrResaleTradeReturnShipmentPhysicalReturnNotRequired = errors.New(
		"resale trade return shipment: physical return is not required",
	)
	ErrResaleTradeReturnShipmentDisputed = errors.New(
		"resale trade return shipment: return is disputed",
	)
	ErrResaleTradeReturnShipmentStateConflict = errors.New(
		"resale trade return shipment: shipment state conflict",
	)
)

const resaleTradeReturnShipmentReadySystemMessageIDPrefix = "return-shipment-ready-"

// ResaleTradeReturnShipmentUsecase prepares the physical return shipment for
// an Avatar-to-Avatar Resale Trade after the buyer has explicitly accepted a
// seller proposal requiring physical return.
//
// Current responsibilities:
//   - authenticate the buyer against the persisted Trade
//   - confirm the authoritative Order item is paid, resale, dispatched and not transferred
//   - confirm the accepted ReturnAgreement proposal requires physical return
//   - create one idempotent local ReturnShipment
//   - generate development-only mock PUDO shipment data
//   - persist the mock QR payload as ready_for_dropoff
//   - create an idempotent system timeline message
//
// Important invariants:
//   - no real PUDO/Yamato API is called by this usecase
//   - the generated QR payload is development-only mock data
//   - real QR creation belongs to the PUDO/provider integration
//   - QR issuance alone never means return_shipped
//   - this usecase does not detect carrier acceptance
//   - this usecase does not detect carrier delivery
//   - this usecase does not advance ReturnAgreement to return_shipped or return_received
//   - disputed returns cannot create or prepare a new return shipment
//   - accepted proposal conditions are never changed here
//   - returnRequirement=not_required never creates ReturnShipment
//
// Current flow:
//
//	agreed
//	  -> local ReturnShipment pending
//	  -> generate development-only mock PUDO payload
//	  -> ready_for_dropoff
//
// Future production flow:
//
//	agreed
//	  -> local ReturnShipment pending
//	  -> PUDO/Yamato adapter
//	  -> provider-generated shipment ID / QR payload
//	  -> ready_for_dropoff
type ResaleTradeReturnShipmentUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	returnShipmentRepo  tradedom.ReturnShipmentRepository
	orderRepo           orderdom.Repository
	messageRepo         tradedom.MessageRepository

	now func() time.Time
}

type NewResaleTradeReturnShipmentUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	ReturnShipmentRepository  tradedom.ReturnShipmentRepository
	OrderRepository           orderdom.Repository
	MessageRepository         tradedom.MessageRepository
}

func NewResaleTradeReturnShipmentUsecase(
	in NewResaleTradeReturnShipmentUsecaseInput,
) *ResaleTradeReturnShipmentUsecase {
	return &ResaleTradeReturnShipmentUsecase{
		tradeRepo:           in.TradeRepository,
		returnAgreementRepo: in.ReturnAgreementRepository,
		returnShipmentRepo:  in.ReturnShipmentRepository,
		orderRepo:           in.OrderRepository,
		messageRepo:         in.MessageRepository,
		now:                 time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (u *ResaleTradeReturnShipmentUsecase) SetNowFunc(
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

type CreateResaleTradeReturnShipmentInput struct {
	TradeID       string
	BuyerAvatarID string
}

type GetResaleTradeReturnShipmentInput struct {
	TradeID       string
	BuyerAvatarID string
}

type ResaleTradeReturnShipmentResult struct {
	Trade     tradedom.Trade
	Order     orderdom.Order
	Item      orderdom.OrderItemSnapshot
	Agreement tradedom.ReturnAgreement
	Proposal  tradedom.ReturnProposal
	Shipment  tradedom.ReturnShipment

	Changed              bool
	SystemMessageEnsured bool
}

// ============================================================
// Create / prepare
// ============================================================

// Create creates or idempotently prepares the physical return shipment.
//
// A pending ReturnShipment is persisted first.
//
// At the current development stage there is no real PUDO/Yamato integration.
// Therefore this usecase generates deterministic mock shipment data and moves
// the local ReturnShipment to ready_for_dropoff.
//
// The mock payload must never be treated as a real PUDO shipment credential.
func (u *ResaleTradeReturnShipmentUsecase) Create(
	ctx context.Context,
	in CreateResaleTradeReturnShipmentInput,
) (ResaleTradeReturnShipmentResult, error) {
	result, agreement, err := u.loadAndValidate(
		ctx,
		in.TradeID,
		in.BuyerAvatarID,
	)
	if err != nil {
		return result, err
	}

	if agreement.Status == tradedom.ReturnStatusDisputed {
		return result, ErrResaleTradeReturnShipmentDisputed
	}
	if agreement.Status != tradedom.ReturnStatusAgreed {
		return result, ErrResaleTradeReturnShipmentAgreementNotReady
	}

	shipment, err := u.returnShipmentRepo.GetByTradeID(
		ctx,
		result.Trade.ID,
	)
	if err == nil {
		if err := validateResaleTradeReturnShipmentIdentity(
			shipment,
			result.Trade,
			agreement,
			result.Proposal,
		); err != nil {
			return result, err
		}

		result.Shipment = shipment

		switch shipment.Status {
		case tradedom.ReturnShipmentStatusPending:
			return u.prepareMockShipment(
				ctx,
				result,
				agreement,
				shipment,
			)

		case tradedom.ReturnShipmentStatusReadyForDropOff:
			ensured, ensureErr := u.ensureReadySystemMessage(
				ctx,
				shipment,
				result.Proposal,
			)
			if ensureErr != nil {
				return result, ensureErr
			}

			result.SystemMessageEnsured = ensured
			return result, nil

		default:
			// This usecase currently owns only:
			//
			//	pending -> ready_for_dropoff
			//
			// Later physical shipment states are intentionally outside the
			// current implementation.
			return result, nil
		}
	}

	if !errors.Is(
		err,
		tradedom.ErrReturnShipmentNotFound,
	) {
		return result, err
	}

	shipment, err = tradedom.NewReturnShipmentForCreate(
		"",
		result.Trade.ID,
		agreement.ID,
		result.Proposal.ID,
	)
	if err != nil {
		return result, err
	}

	created, err := u.returnShipmentRepo.Create(
		ctx,
		shipment,
	)
	if err != nil {
		if !errors.Is(
			err,
			tradedom.ErrReturnShipmentAlreadyExists,
		) {
			return result, err
		}

		created, err = u.returnShipmentRepo.GetByTradeID(
			ctx,
			result.Trade.ID,
		)
		if err != nil {
			return result, err
		}
	}

	if err := validateResaleTradeReturnShipmentIdentity(
		created,
		result.Trade,
		agreement,
		result.Proposal,
	); err != nil {
		return result, err
	}

	result.Shipment = created
	result.Changed = true

	switch created.Status {
	case tradedom.ReturnShipmentStatusPending:
		return u.prepareMockShipment(
			ctx,
			result,
			agreement,
			created,
		)

	case tradedom.ReturnShipmentStatusReadyForDropOff:
		ensured, ensureErr := u.ensureReadySystemMessage(
			ctx,
			created,
			result.Proposal,
		)
		if ensureErr != nil {
			return result, ensureErr
		}

		result.SystemMessageEnsured = ensured
		return result, nil

	default:
		return result, nil
	}
}

// prepareMockShipment generates development-only mock shipment information.
//
// Real provider integration must replace this method's mock-data generation
// with a dedicated PUDO/Yamato adapter.
//
// AMOL must not implement or reproduce PUDO's production QR-generation logic.
func (u *ResaleTradeReturnShipmentUsecase) prepareMockShipment(
	ctx context.Context,
	result ResaleTradeReturnShipmentResult,
	agreement tradedom.ReturnAgreement,
	shipment tradedom.ReturnShipment,
) (ResaleTradeReturnShipmentResult, error) {
	if shipment.Status != tradedom.ReturnShipmentStatusPending {
		return result, ErrResaleTradeReturnShipmentStateConflict
	}

	latestAgreement, err :=
		u.reloadReturnAgreementForPreparation(
			ctx,
			agreement,
			result.Proposal.ID,
		)
	if err != nil {
		return result, err
	}

	agreement = latestAgreement
	result.Agreement = agreement

	if agreement.Status == tradedom.ReturnStatusDisputed {
		return result, ErrResaleTradeReturnShipmentDisputed
	}
	if agreement.Status != tradedom.ReturnStatusAgreed {
		return result, ErrResaleTradeReturnShipmentAgreementNotReady
	}

	preparedAt := normalizeResaleTradeReturnShipmentTime(
		u.nowUTC(),
		shipment.UpdatedAt,
	)

	mockProviderShipmentID :=
		buildMockPUDOReturnShipmentID(
			result.Trade.ID,
			result.Proposal.ID,
		)

	mockQRCodePayload :=
		buildMockPUDOQRCodePayload(
			mockProviderShipmentID,
		)

	if err := shipment.MarkReady(
		mockProviderShipmentID,
		mockQRCodePayload,
		nil,
		"",
		"mock_ready_for_dropoff",
		preparedAt,
	); err != nil {
		return result, err
	}

	updated, err := u.returnShipmentRepo.Update(
		ctx,
		result.Trade.ID,
		shipment,
	)
	if err != nil {
		return result, err
	}

	result.Shipment = updated
	result.Agreement = agreement
	result.Changed = true

	ensured, err := u.ensureReadySystemMessage(
		ctx,
		updated,
		result.Proposal,
	)
	if err != nil {
		return result, err
	}

	result.SystemMessageEnsured = ensured

	return result, nil
}

// ============================================================
// Get
// ============================================================

// Get returns the locally persisted ReturnShipment.
//
// It intentionally does not contact PUDO/Yamato.
//
// The current ready_for_dropoff QR payload is development-only mock data.
// Carrier shipment-state synchronization is not implemented.
func (u *ResaleTradeReturnShipmentUsecase) Get(
	ctx context.Context,
	in GetResaleTradeReturnShipmentInput,
) (ResaleTradeReturnShipmentResult, error) {
	result, agreement, err := u.loadAndValidate(
		ctx,
		in.TradeID,
		in.BuyerAvatarID,
	)
	if err != nil {
		return result, err
	}

	shipment, err := u.returnShipmentRepo.GetByTradeID(
		ctx,
		result.Trade.ID,
	)
	if err != nil {
		return result, err
	}

	if err := validateResaleTradeReturnShipmentIdentity(
		shipment,
		result.Trade,
		agreement,
		result.Proposal,
	); err != nil {
		return result, err
	}

	result.Shipment = shipment

	if shipment.Status ==
		tradedom.ReturnShipmentStatusReadyForDropOff {
		ensured, ensureErr :=
			u.ensureReadySystemMessage(
				ctx,
				shipment,
				result.Proposal,
			)
		if ensureErr != nil {
			return result, ensureErr
		}

		result.SystemMessageEnsured = ensured
	}

	return result, nil
}

// ============================================================
// ReturnAgreement reload
// ============================================================

func (u *ResaleTradeReturnShipmentUsecase) reloadReturnAgreementForPreparation(
	ctx context.Context,
	previous tradedom.ReturnAgreement,
	proposalID string,
) (tradedom.ReturnAgreement, error) {
	current, err := u.returnAgreementRepo.GetByTradeID(
		ctx,
		previous.TradeID,
	)
	if err != nil {
		return previous, err
	}

	proposalID = strings.TrimSpace(proposalID)

	if current.ID != previous.ID ||
		current.TradeID != previous.TradeID ||
		current.Proposal == nil ||
		strings.TrimSpace(current.Proposal.ID) == "" ||
		current.Proposal.ID != proposalID ||
		current.Proposal.Agreement !=
			tradedom.ReturnProposalAgreementAgree ||
		current.Proposal.RejectedAt != nil ||
		current.Proposal.ReturnRequirement !=
			tradedom.ReturnRequirementRequired ||
		current.AgreedAt == nil ||
		current.AgreedAt.IsZero() {
		return previous,
			tradedom.ErrReturnAgreementConflict
	}

	return current, nil
}

// ============================================================
// System message
// ============================================================

func (u *ResaleTradeReturnShipmentUsecase) ensureReadySystemMessage(
	ctx context.Context,
	shipment tradedom.ReturnShipment,
	proposal tradedom.ReturnProposal,
) (bool, error) {
	if shipment.Status !=
		tradedom.ReturnShipmentStatusReadyForDropOff ||
		shipment.ReadyAt == nil ||
		shipment.ReadyAt.IsZero() {
		return false, nil
	}

	proposalID := strings.TrimSpace(proposal.ID)
	if proposalID == "" {
		return false,
			tradedom.ErrInvalidReturnProposalID
	}

	message, err := tradedom.NewSystemMessageForCreate(
		resaleTradeReturnShipmentReadySystemMessageIDPrefix+
			proposalID,
		shipment.TradeID,
		"開発用のPUDO匿名返品QR（mock）を発行しました。",
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

// ============================================================
// Load / validation
// ============================================================

func (u *ResaleTradeReturnShipmentUsecase) loadAndValidate(
	ctx context.Context,
	rawTradeID string,
	rawBuyerAvatarID string,
) (
	ResaleTradeReturnShipmentResult,
	tradedom.ReturnAgreement,
	error,
) {
	if err := u.validateConfigured(); err != nil {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			err
	}

	tradeID := strings.TrimSpace(rawTradeID)
	if tradeID == "" {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidID
	}

	buyerAvatarID :=
		strings.TrimSpace(rawBuyerAvatarID)
	if buyerAvatarID == "" {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentInvalidBuyer
	}

	trade, err := u.tradeRepo.GetByID(
		ctx,
		tradeID,
	)
	if err != nil {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrNotFound
	}

	if trade.BuyerAvatarID != buyerAvatarID {
		return ResaleTradeReturnShipmentResult{},
			tradedom.ReturnAgreement{},
			tradedom.ErrNotFound
	}

	if trade.Status == tradedom.StatusClosed {
		return ResaleTradeReturnShipmentResult{
				Trade: trade,
			},
			tradedom.ReturnAgreement{},
			tradedom.ErrTradeAlreadyClosed
	}

	if trade.Status != tradedom.StatusActive {
		return ResaleTradeReturnShipmentResult{
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
		return ResaleTradeReturnShipmentResult{
				Trade: trade,
			},
			tradedom.ReturnAgreement{},
			err
	}

	item, err :=
		validateResaleTradeReturnShipmentTarget(
			order,
			trade,
			buyerAvatarID,
		)
	if err != nil {
		return ResaleTradeReturnShipmentResult{
				Trade: trade,
				Order: order,
			},
			tradedom.ReturnAgreement{},
			err
	}

	result := ResaleTradeReturnShipmentResult{
		Trade: trade,
		Order: order,
		Item:  item,
	}

	if !order.Paid {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentOrderNotPaid
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.Transferred {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentNotEligible
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

	proposal := *agreement.Proposal

	if strings.TrimSpace(proposal.ID) == "" ||
		proposal.Agreement !=
			tradedom.ReturnProposalAgreementAgree ||
		proposal.RejectedAt != nil ||
		agreement.AgreedAt == nil ||
		agreement.AgreedAt.IsZero() {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentAgreementNotReady
	}

	if proposal.ReturnRequirement ==
		tradedom.ReturnRequirementNotRequired {
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentPhysicalReturnNotRequired
	}

	if proposal.ReturnRequirement !=
		tradedom.ReturnRequirementRequired {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnRequirement
	}

	if proposal.RefundAmount <= 0 {
		return result,
			tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnRefundAmount
	}

	switch agreement.Status {
	case tradedom.ReturnStatusAgreed:
		// ReturnShipment preparation is allowed.

	case tradedom.ReturnStatusDisputed:
		result.Agreement = agreement
		result.Proposal = proposal

		return result,
			agreement,
			nil

	default:
		return result,
			tradedom.ReturnAgreement{},
			ErrResaleTradeReturnShipmentAgreementNotReady
	}

	result.Agreement = agreement
	result.Proposal = proposal

	return result, agreement, nil
}

func validateResaleTradeReturnShipmentTarget(
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
			ErrResaleTradeReturnShipmentTradeMismatch
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(order.Items) {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnShipmentTradeMismatch
	}

	item := order.Items[trade.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale ||
		strings.TrimSpace(item.ResaleID) == "" {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnShipmentTradeMismatch
	}

	if strings.TrimSpace(
		item.SellerSnapshot.AvatarID,
	) == "" ||
		item.SellerSnapshot.AvatarID !=
			trade.SellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnShipmentTradeMismatch
	}

	return item, nil
}

func validateResaleTradeReturnShipmentIdentity(
	shipment tradedom.ReturnShipment,
	trade tradedom.Trade,
	agreement tradedom.ReturnAgreement,
	proposal tradedom.ReturnProposal,
) error {
	if shipment.ID != trade.ID ||
		shipment.TradeID != trade.ID ||
		shipment.ReturnAgreementID != agreement.ID ||
		shipment.ProposalID != proposal.ID ||
		shipment.Carrier !=
			tradedom.ReturnShipmentCarrierYamato ||
		shipment.DropOffMethod !=
			tradedom.ReturnShipmentDropOffMethodPUDO {
		return tradedom.ErrReturnShipmentConflict
	}

	return nil
}

// ============================================================
// Mock PUDO data
// ============================================================

// buildMockPUDOReturnShipmentID returns a deterministic development-only
// provider shipment ID.
//
// This is not a PUDO/Yamato identifier.
func buildMockPUDOReturnShipmentID(
	tradeID string,
	proposalID string,
) string {
	source :=
		strings.TrimSpace(tradeID) +
			":" +
			strings.TrimSpace(proposalID)

	sum := sha256.Sum256([]byte(source))

	return fmt.Sprintf(
		"mock-pudo-%x",
		sum[:16],
	)
}

// buildMockPUDOQRCodePayload returns development-only data that the frontend
// may encode into a QR image.
//
// This payload is not generated by PUDO, is not accepted by PUDO, and must not
// be used as a production shipping credential.
//
// In production this value must come from the PUDO/Yamato provider adapter.
func buildMockPUDOQRCodePayload(
	mockProviderShipmentID string,
) string {
	return "amol-mock-pudo-return://" +
		strings.TrimSpace(mockProviderShipmentID)
}

// ============================================================
// Internal helpers
// ============================================================

func (u *ResaleTradeReturnShipmentUsecase) validateConfigured() error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.returnAgreementRepo == nil ||
		u.returnShipmentRepo == nil ||
		u.orderRepo == nil ||
		u.messageRepo == nil ||
		u.now == nil {
		return ErrResaleTradeReturnShipmentNotConfigured
	}

	return nil
}

func (u *ResaleTradeReturnShipmentUsecase) nowUTC() time.Time {
	return u.now().UTC()
}

func normalizeResaleTradeReturnShipmentTime(
	candidate time.Time,
	minimum time.Time,
) time.Time {
	at := candidate
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()

	if !minimum.IsZero() {
		minimum = minimum.UTC()

		if at.Before(minimum) {
			at = minimum
		}
	}

	return at
}
