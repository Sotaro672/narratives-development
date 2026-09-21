// backend/internal/application/usecase/resale_trade_return_consultation_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrResaleTradeReturnConsultationNotConfigured = errors.New(
		"resale trade return consultation: usecase is not configured",
	)
	ErrResaleTradeReturnConsultationInvalidBuyer = errors.New(
		"resale trade return consultation: invalid buyer",
	)
	ErrResaleTradeReturnConsultationTradeMismatch = errors.New(
		"resale trade return consultation: trade does not match order item",
	)
	ErrResaleTradeReturnConsultationOrderNotPaid = errors.New(
		"resale trade return consultation: order is not paid",
	)
	ErrResaleTradeReturnConsultationNotEligible = errors.New(
		"resale trade return consultation: trade is not eligible for return consultation",
	)
	ErrResaleTradeReturnConsultationAlreadyExists = errors.New(
		"resale trade return consultation: another consultation already exists",
	)
)

const (
	resaleTradeReturnConsultationSystemMessageID = "return-consultation"
)

// ResaleTradeReturnConsultationUsecase starts the return-negotiation flow for
// one Avatar-to-Avatar Resale Trade.
//
// This usecase does not mutate the legacy Order return-request fields.
//
// Responsibilities:
//   - authenticate the buyer against the persisted Trade
//   - confirm that the Trade represents an Avatar-to-Avatar Resale transaction
//   - confirm the authoritative Order item is paid, dispatched and not transferred
//   - create exactly one ReturnAgreement for the Trade
//   - create an idempotent system timeline message
//
// Return negotiation state is owned by ReturnAgreement:
//
//	none -> discussing
//
// Order remains authoritative for purchase, cancellation, dispatch and token
// transfer state.
type ResaleTradeReturnConsultationUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	orderRepo           orderdom.Repository
	messageRepo         tradedom.MessageRepository

	now func() time.Time
}

type NewResaleTradeReturnConsultationUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	OrderRepository           orderdom.Repository
	MessageRepository         tradedom.MessageRepository
}

func NewResaleTradeReturnConsultationUsecase(
	in NewResaleTradeReturnConsultationUsecaseInput,
) *ResaleTradeReturnConsultationUsecase {
	return &ResaleTradeReturnConsultationUsecase{
		tradeRepo:           in.TradeRepository,
		returnAgreementRepo: in.ReturnAgreementRepository,
		orderRepo:           in.OrderRepository,
		messageRepo:         in.MessageRepository,
		now:                 time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (u *ResaleTradeReturnConsultationUsecase) SetNowFunc(
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

type CreateResaleTradeReturnConsultationInput struct {
	TradeID       string
	BuyerAvatarID string
	Reason        tradedom.ReturnConsultationReason
	Detail        string
}

type ResaleTradeReturnConsultationResult struct {
	Trade     tradedom.Trade
	Order     orderdom.Order
	Item      orderdom.OrderItemSnapshot
	Agreement tradedom.ReturnAgreement

	Created              bool
	SystemMessageEnsured bool
}

// ============================================================
// Create consultation
// ============================================================

// Create starts or idempotently confirms a buyer return consultation.
//
// Repeating the exact same request is safe:
//
//	first request
//	  -> ReturnAgreement created
//	  -> system message ensured
//
//	retry
//	  -> existing ReturnAgreement returned
//	  -> same system message ensured
//
// A request with different consultation contents for a Trade that already has
// a ReturnAgreement is rejected as a conflict.
func (u *ResaleTradeReturnConsultationUsecase) Create(
	ctx context.Context,
	in CreateResaleTradeReturnConsultationInput,
) (ResaleTradeReturnConsultationResult, error) {
	if err := u.validateConfigured(); err != nil {
		return ResaleTradeReturnConsultationResult{}, err
	}

	tradeID := strings.TrimSpace(in.TradeID)
	if tradeID == "" {
		return ResaleTradeReturnConsultationResult{},
			tradedom.ErrInvalidID
	}

	buyerAvatarID := strings.TrimSpace(in.BuyerAvatarID)
	if buyerAvatarID == "" {
		return ResaleTradeReturnConsultationResult{},
			ErrResaleTradeReturnConsultationInvalidBuyer
	}

	detail := strings.TrimSpace(in.Detail)

	if !tradedom.IsValidReturnConsultationReason(in.Reason) {
		return ResaleTradeReturnConsultationResult{},
			tradedom.ErrInvalidReturnConsultationReason
	}

	trade, err := u.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return ResaleTradeReturnConsultationResult{}, err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" {
		return ResaleTradeReturnConsultationResult{},
			tradedom.ErrNotFound
	}

	if trade.BuyerAvatarID != buyerAvatarID {
		return ResaleTradeReturnConsultationResult{},
			tradedom.ErrNotFound
	}

	if trade.Status == tradedom.StatusClosed {
		return ResaleTradeReturnConsultationResult{
			Trade: trade,
		}, tradedom.ErrTradeAlreadyClosed
	}
	if trade.Status != tradedom.StatusActive {
		return ResaleTradeReturnConsultationResult{
			Trade: trade,
		}, tradedom.ErrInvalidStatus
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return ResaleTradeReturnConsultationResult{
			Trade: trade,
		}, err
	}

	item, err := validateResaleTradeReturnConsultationTarget(
		order,
		trade,
		buyerAvatarID,
	)
	if err != nil {
		return ResaleTradeReturnConsultationResult{
			Trade: trade,
			Order: order,
		}, err
	}

	result := ResaleTradeReturnConsultationResult{
		Trade: trade,
		Order: order,
		Item:  item,
	}

	if !order.Paid {
		return result,
			ErrResaleTradeReturnConsultationOrderNotPaid
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.Transferred {
		return result,
			ErrResaleTradeReturnConsultationNotEligible
	}

	// First check for an existing aggregate so retries of the same request are
	// idempotent without attempting another Firestore create.
	existing, err := u.returnAgreementRepo.GetByTradeID(
		ctx,
		tradeID,
	)
	switch {
	case err == nil:
		if !sameResaleTradeReturnConsultation(
			existing,
			in.Reason,
			detail,
		) {
			return result,
				ErrResaleTradeReturnConsultationAlreadyExists
		}

		result.Agreement = existing

		ensured, err := u.ensureSystemMessage(
			ctx,
			tradeID,
			in.Reason,
		)
		if err != nil {
			return result, err
		}

		result.SystemMessageEnsured = ensured
		return result, nil

	case errors.Is(
		err,
		tradedom.ErrReturnAgreementNotFound,
	):
		// Continue to create.

	default:
		return result, err
	}

	now := u.nowUTC()

	agreement, err :=
		tradedom.NewReturnAgreementForCreate(
			tradeID,
			tradeID,
			"",
			in.Reason,
			detail,
		)
	if err != nil {
		return result, err
	}

	agreement.CreatedAt = now
	agreement.UpdatedAt = now
	agreement.Consultation.CreatedAt = now

	created, err := u.returnAgreementRepo.Create(
		ctx,
		agreement,
	)
	if err != nil {
		// Another request may have created the same consultation after our
		// initial GetByTradeID. Resolve that race as an idempotent retry when
		// the persisted consultation is identical.
		if errors.Is(
			err,
			tradedom.ErrReturnAgreementAlreadyExists,
		) {
			existing, getErr :=
				u.returnAgreementRepo.GetByTradeID(
					ctx,
					tradeID,
				)
			if getErr != nil {
				return result, getErr
			}

			if !sameResaleTradeReturnConsultation(
				existing,
				in.Reason,
				detail,
			) {
				return result,
					ErrResaleTradeReturnConsultationAlreadyExists
			}

			result.Agreement = existing

			ensured, ensureErr :=
				u.ensureSystemMessage(
					ctx,
					tradeID,
					in.Reason,
				)
			if ensureErr != nil {
				return result, ensureErr
			}

			result.SystemMessageEnsured = ensured
			return result, nil
		}

		return result, err
	}

	result.Agreement = created
	result.Created = true

	ensured, err := u.ensureSystemMessage(
		ctx,
		tradeID,
		in.Reason,
	)
	if err != nil {
		// ReturnAgreement has already been persisted at this point.
		// Returning the error allows the caller to retry. The deterministic
		// system-message ID makes that retry safe.
		return result, err
	}

	result.SystemMessageEnsured = ensured
	return result, nil
}

// ============================================================
// System message
// ============================================================

func (u *ResaleTradeReturnConsultationUsecase) ensureSystemMessage(
	ctx context.Context,
	tradeID string,
	reason tradedom.ReturnConsultationReason,
) (bool, error) {
	message, err := tradedom.NewSystemMessageForCreate(
		resaleTradeReturnConsultationSystemMessageID,
		tradeID,
		buildResaleTradeReturnConsultationSystemMessage(
			reason,
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

func buildResaleTradeReturnConsultationSystemMessage(
	reason tradedom.ReturnConsultationReason,
) string {
	return "購入者が返品についての相談を開始しました。返品理由: " +
		resaleTradeReturnConsultationReasonLabel(reason)
}

func resaleTradeReturnConsultationReasonLabel(
	reason tradedom.ReturnConsultationReason,
) string {
	switch reason {
	case tradedom.ReturnConsultationReasonNotAsDescribed:
		return "商品説明と状態が異なる"

	case tradedom.ReturnConsultationReasonDamaged:
		return "商品が破損している"

	case tradedom.ReturnConsultationReasonWrongItem:
		return "異なる商品が届いた"

	case tradedom.ReturnConsultationReasonOther:
		return "その他"

	default:
		return "その他"
	}
}

// ============================================================
// Validation
// ============================================================

func validateResaleTradeReturnConsultationTarget(
	order orderdom.Order,
	trade tradedom.Trade,
	buyerAvatarID string,
) (orderdom.OrderItemSnapshot, error) {
	if order.ID == "" ||
		order.ID != trade.OrderID ||
		order.AvatarID != trade.BuyerAvatarID ||
		order.AvatarID != buyerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnConsultationTradeMismatch
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(order.Items) {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnConsultationTradeMismatch
	}

	item := order.Items[trade.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale ||
		item.ResaleID == "" {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnConsultationTradeMismatch
	}

	if item.SellerSnapshot.AvatarID == "" ||
		item.SellerSnapshot.AvatarID !=
			trade.SellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			ErrResaleTradeReturnConsultationTradeMismatch
	}

	return item, nil
}

func sameResaleTradeReturnConsultation(
	agreement tradedom.ReturnAgreement,
	reason tradedom.ReturnConsultationReason,
	detail string,
) bool {
	return agreement.TradeID != "" &&
		agreement.Consultation.Reason == reason &&
		strings.TrimSpace(
			agreement.Consultation.Detail,
		) == strings.TrimSpace(detail)
}

// ============================================================
// Internal
// ============================================================

func (u *ResaleTradeReturnConsultationUsecase) validateConfigured() error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.returnAgreementRepo == nil ||
		u.orderRepo == nil ||
		u.messageRepo == nil ||
		u.now == nil {
		return ErrResaleTradeReturnConsultationNotConfigured
	}

	return nil
}

func (u *ResaleTradeReturnConsultationUsecase) nowUTC() time.Time {
	return u.now().UTC()
}
