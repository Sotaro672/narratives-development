// backend/internal/application/usecase/trade_information_request_usecase.go

package usecase

import (
	"context"
	"errors"
	"time"

	tradedom "narratives/internal/domain/trade"
)

const tradeInformationRequestSystemMessageID = "information-request"

var (
	ErrTradeInformationRequestUsecaseNotConfigured = errors.New(
		"trade information request: usecase is not configured",
	)
	ErrTradeInformationRequestUnsupportedTrade = errors.New(
		"trade information request: unsupported trade",
	)
	ErrTradeInformationRequestTargetMismatch = errors.New(
		"trade information request: target avatar does not match trade",
	)
	ErrTradeInformationRequestForbidden = errors.New(
		"trade information request: access denied",
	)
	ErrTradeInformationRequestAlreadyExists = errors.New(
		"trade information request: information request already exists",
	)
)

// TradeInformationRequestUsecase coordinates an Admin information request for
// one participant in one Avatar-to-Avatar Resale Trade.
//
// One Trade may have at most one InformationRequest:
//
//	tradeInformationRequests/{tradeId}
//
// Admin chooses only TargetSide. TargetAvatarID is always resolved from the
// persisted Trade and is never accepted from the client.
//
// Creation also ensures one deterministic Trade system message:
//
//	trades/{tradeId}/messages/information-request
//
// The system message itself does not own the action counter. It is persisted as
// already read for both participants so the normal unread-message counter does
// not double-count the request. Mall keeps one action count while the
// InformationRequest remains pending for the authenticated target Avatar.
type TradeInformationRequestUsecase struct {
	tradeRepo              tradedom.Repository
	informationRequestRepo tradedom.InformationRequestRepository
	messageRepo            tradedom.MessageRepository
	now                    func() time.Time
}

type NewTradeInformationRequestUsecaseInput struct {
	TradeRepository              tradedom.Repository
	InformationRequestRepository tradedom.InformationRequestRepository
	MessageRepository            tradedom.MessageRepository
}

func NewTradeInformationRequestUsecase(
	in NewTradeInformationRequestUsecaseInput,
) *TradeInformationRequestUsecase {
	return &TradeInformationRequestUsecase{
		tradeRepo:              in.TradeRepository,
		informationRequestRepo: in.InformationRequestRepository,
		messageRepo:            in.MessageRepository,
		now:                    time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (u *TradeInformationRequestUsecase) SetNowFunc(now func() time.Time) {
	if u == nil || now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Admin create
// ============================================================

type CreateTradeInformationRequestInput struct {
	TradeID    string
	TargetSide tradedom.InformationRequestTargetSide
	Reason     string
}

type TradeInformationRequestResult struct {
	Trade   tradedom.Trade
	Request tradedom.InformationRequest
	Message tradedom.Message

	Created              bool
	SystemMessageEnsured bool
}

// Create creates one information request for a Trade.
//
// One Trade may have only one InformationRequest. Repeating the same request is
// treated as an idempotent retry so a previously failed system-message creation
// can be recovered safely. A different request for the same Trade is rejected.
func (u *TradeInformationRequestUsecase) Create(
	ctx context.Context,
	in CreateTradeInformationRequestInput,
) (TradeInformationRequestResult, error) {
	if err := u.validateConfigured(); err != nil {
		return TradeInformationRequestResult{}, err
	}
	if in.TradeID == "" {
		return TradeInformationRequestResult{}, tradedom.ErrInvalidID
	}
	if !tradedom.IsValidInformationRequestTargetSide(in.TargetSide) {
		return TradeInformationRequestResult{},
			tradedom.ErrInvalidInformationRequestTargetSide
	}

	trade, err := u.tradeRepo.GetByID(ctx, in.TradeID)
	if err != nil {
		return TradeInformationRequestResult{}, err
	}
	if trade.ID != in.TradeID {
		return TradeInformationRequestResult{}, tradedom.ErrNotFound
	}
	if trade.Status == tradedom.StatusClosed {
		return TradeInformationRequestResult{
			Trade: trade,
		}, tradedom.ErrTradeAlreadyClosed
	}
	if trade.Status != tradedom.StatusActive {
		return TradeInformationRequestResult{
			Trade: trade,
		}, tradedom.ErrInvalidStatus
	}
	if trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.BuyerAvatarID == "" ||
		trade.SellerAvatarID == "" {
		return TradeInformationRequestResult{
			Trade: trade,
		}, ErrTradeInformationRequestUnsupportedTrade
	}

	targetAvatarID, err := resolveTradeInformationRequestTargetAvatarID(
		trade,
		in.TargetSide,
	)
	if err != nil {
		return TradeInformationRequestResult{
			Trade: trade,
		}, err
	}

	request, err := tradedom.NewInformationRequestForCreate(
		trade.ID,
		trade.ID,
		in.TargetSide,
		targetAvatarID,
		in.Reason,
	)
	if err != nil {
		return TradeInformationRequestResult{
			Trade: trade,
		}, err
	}

	result := TradeInformationRequestResult{
		Trade: trade,
	}

	existing, err := u.informationRequestRepo.GetByTradeID(
		ctx,
		trade.ID,
	)
	switch {
	case err == nil:
		if !sameTradeInformationRequest(existing, request) {
			return result, ErrTradeInformationRequestAlreadyExists
		}

		result.Request = existing

		message, ensured, ensureErr := u.ensureSystemMessage(
			ctx,
			existing,
		)
		if ensureErr != nil {
			return result, ensureErr
		}

		result.Message = message
		result.SystemMessageEnsured = ensured
		return result, nil

	case errors.Is(err, tradedom.ErrInformationRequestNotFound):
		// Continue to create the first and only request for this Trade.

	default:
		return result, err
	}

	now := u.nowUTC()
	request.CreatedAt = now
	request.UpdatedAt = now

	created, err := u.informationRequestRepo.Create(
		ctx,
		request,
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrInformationRequestAlreadyExists,
		) {
			existing, getErr := u.informationRequestRepo.GetByTradeID(
				ctx,
				trade.ID,
			)
			if getErr != nil {
				return result, getErr
			}
			if !sameTradeInformationRequest(existing, request) {
				return result, ErrTradeInformationRequestAlreadyExists
			}

			result.Request = existing

			message, ensured, ensureErr := u.ensureSystemMessage(
				ctx,
				existing,
			)
			if ensureErr != nil {
				return result, ensureErr
			}

			result.Message = message
			result.SystemMessageEnsured = ensured
			return result, nil
		}

		return result, err
	}

	result.Request = created
	result.Created = true

	message, ensured, err := u.ensureSystemMessage(
		ctx,
		created,
	)
	if err != nil {
		// The InformationRequest is already persisted. Returning the error lets
		// the caller retry; the deterministic message ID makes that retry safe.
		return result, err
	}

	result.Message = message
	result.SystemMessageEnsured = ensured
	return result, nil
}

// ============================================================
// Mall get
// ============================================================

type GetTradeInformationRequestForAvatarInput struct {
	TradeID  string
	AvatarID string
}

// GetForAvatar returns the information request only to the Avatar it is
// addressed to.
//
// A different participant receives ErrNotFound so the request contents are not
// exposed to the non-target participant.
func (u *TradeInformationRequestUsecase) GetForAvatar(
	ctx context.Context,
	in GetTradeInformationRequestForAvatarInput,
) (tradedom.InformationRequest, error) {
	if err := u.validateConfigured(); err != nil {
		return tradedom.InformationRequest{}, err
	}
	if in.TradeID == "" {
		return tradedom.InformationRequest{}, tradedom.ErrInvalidID
	}
	if in.AvatarID == "" {
		return tradedom.InformationRequest{},
			ErrTradeInformationRequestForbidden
	}

	trade, err := u.tradeRepo.GetByID(ctx, in.TradeID)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}
	if trade.ID != in.TradeID {
		return tradedom.InformationRequest{}, tradedom.ErrNotFound
	}
	if !isTradeInformationRequestParticipant(trade, in.AvatarID) {
		return tradedom.InformationRequest{}, tradedom.ErrNotFound
	}

	request, err := u.informationRequestRepo.GetByTradeID(
		ctx,
		trade.ID,
	)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}

	if request.TradeID != trade.ID ||
		request.ID != trade.ID {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestConflict
	}
	if request.TargetAvatarID != in.AvatarID {
		return tradedom.InformationRequest{}, tradedom.ErrNotFound
	}
	if !informationRequestTargetMatchesTrade(
		request,
		trade,
	) {
		return tradedom.InformationRequest{},
			ErrTradeInformationRequestTargetMismatch
	}

	return request, nil
}

// ============================================================
// Mall submit response
// ============================================================

type SubmitTradeInformationRequestResponseInput struct {
	TradeID  string
	AvatarID string
	Response string
}

// SubmitResponse records the response from the Avatar selected by Admin.
func (u *TradeInformationRequestUsecase) SubmitResponse(
	ctx context.Context,
	in SubmitTradeInformationRequestResponseInput,
) (tradedom.InformationRequest, error) {
	if err := u.validateConfigured(); err != nil {
		return tradedom.InformationRequest{}, err
	}

	request, err := u.GetForAvatar(
		ctx,
		GetTradeInformationRequestForAvatarInput{
			TradeID:  in.TradeID,
			AvatarID: in.AvatarID,
		},
	)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}

	if err := request.Submit(
		in.Response,
		u.nowUTC(),
	); err != nil {
		return tradedom.InformationRequest{}, err
	}

	updated, err := u.informationRequestRepo.Update(
		ctx,
		request.TradeID,
		request,
	)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}

	return updated, nil
}

// ============================================================
// Pending action
// ============================================================

// HasPendingActionForAvatar determines whether one Trade contributes one
// information-request action to the authenticated Avatar's message counter.
//
// NotFound means the Trade has never received an InformationRequest and is not
// an error for this read path.
func (u *TradeInformationRequestUsecase) HasPendingActionForAvatar(
	ctx context.Context,
	tradeID string,
	avatarID string,
) (bool, error) {
	if err := u.validateConfigured(); err != nil {
		return false, err
	}
	if tradeID == "" {
		return false, tradedom.ErrInvalidID
	}
	if avatarID == "" {
		return false, ErrTradeInformationRequestForbidden
	}

	request, err := u.informationRequestRepo.GetByTradeID(
		ctx,
		tradeID,
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrInformationRequestNotFound,
		) {
			return false, nil
		}

		return false, err
	}

	if request.TradeID != tradeID ||
		request.ID != tradeID {
		return false, tradedom.ErrInformationRequestConflict
	}

	return request.Status ==
		tradedom.InformationRequestStatusPending &&
		request.TargetAvatarID == avatarID, nil
}

// ============================================================
// System message
// ============================================================

func (u *TradeInformationRequestUsecase) ensureSystemMessage(
	ctx context.Context,
	request tradedom.InformationRequest,
) (tradedom.Message, bool, error) {
	message, err := tradedom.NewSystemMessageForCreate(
		tradeInformationRequestSystemMessageID,
		request.TradeID,
		"運営から追加情報の入力が求められています。",
	)
	if err != nil {
		return tradedom.Message{}, false, err
	}

	now := u.nowUTC()
	message.CreatedAt = now

	// The information request has its own pending-action state. Marking this
	// system message as read for both sides prevents normal unread-message
	// counting from adding another count on top of that action.
	message.BuyerReadAt = tradeInformationRequestTimePtr(now)
	message.SellerReadAt = tradeInformationRequestTimePtr(now)

	created, err := u.messageRepo.Create(
		ctx,
		message,
	)
	if err != nil {
		if !errors.Is(
			err,
			tradedom.ErrMessageAlreadyExists,
		) {
			return tradedom.Message{}, false, err
		}

		existing, getErr := u.messageRepo.GetByID(
			ctx,
			request.TradeID,
			tradeInformationRequestSystemMessageID,
		)
		if getErr != nil {
			return tradedom.Message{}, false, getErr
		}

		return existing, true, nil
	}

	return created, true, nil
}

// ============================================================
// Validation / helpers
// ============================================================

func resolveTradeInformationRequestTargetAvatarID(
	trade tradedom.Trade,
	targetSide tradedom.InformationRequestTargetSide,
) (string, error) {
	switch targetSide {
	case tradedom.InformationRequestTargetBuyer:
		if trade.BuyerAvatarID == "" {
			return "", ErrTradeInformationRequestTargetMismatch
		}

		return trade.BuyerAvatarID, nil

	case tradedom.InformationRequestTargetSeller:
		if trade.SellerType != tradedom.SellerTypeAvatar ||
			trade.SellerAvatarID == "" {
			return "", ErrTradeInformationRequestTargetMismatch
		}

		return trade.SellerAvatarID, nil

	default:
		return "", tradedom.ErrInvalidInformationRequestTargetSide
	}
}

func informationRequestTargetMatchesTrade(
	request tradedom.InformationRequest,
	trade tradedom.Trade,
) bool {
	switch request.TargetSide {
	case tradedom.InformationRequestTargetBuyer:
		return request.TargetAvatarID == trade.BuyerAvatarID

	case tradedom.InformationRequestTargetSeller:
		return trade.SellerType == tradedom.SellerTypeAvatar &&
			request.TargetAvatarID == trade.SellerAvatarID

	default:
		return false
	}
}

func isTradeInformationRequestParticipant(
	trade tradedom.Trade,
	avatarID string,
) bool {
	return avatarID == trade.BuyerAvatarID ||
		avatarID == trade.SellerAvatarID
}

func sameTradeInformationRequest(
	current tradedom.InformationRequest,
	expected tradedom.InformationRequest,
) bool {
	return current.ID == expected.ID &&
		current.TradeID == expected.TradeID &&
		current.TargetSide == expected.TargetSide &&
		current.TargetAvatarID == expected.TargetAvatarID &&
		current.Reason == expected.Reason
}

func (u *TradeInformationRequestUsecase) validateConfigured() error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.informationRequestRepo == nil ||
		u.messageRepo == nil ||
		u.now == nil {
		return ErrTradeInformationRequestUsecaseNotConfigured
	}

	return nil
}

func (u *TradeInformationRequestUsecase) nowUTC() time.Time {
	return u.now().UTC()
}

func tradeInformationRequestTimePtr(
	value time.Time,
) *time.Time {
	normalized := value.UTC()
	return &normalized
}
