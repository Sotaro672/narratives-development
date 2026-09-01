// backend/internal/application/usecase/trade_message_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeMessageUsecaseNotConfigured = errors.New("trade message usecase: not configured")
	ErrTradeMessageAvatarIDEmpty        = errors.New("trade message usecase: avatarId is empty")
	ErrTradeMessageUnsupportedTrade     = errors.New("trade message usecase: unsupported trade")
)

// TradeMessageUsecase coordinates private messages between the buyer Avatar
// and seller Avatar of a Resale Trade.
//
// Trade messages are available only for secondary-market transactions:
//
//	buyer Avatar <-> seller Avatar
//
// SenderSide and SenderType are resolved by the application layer from the
// authenticated Avatar and persisted Trade. Clients must never choose their
// own sender identity.
//
// Trade and Message persistence remain separate:
//   - TradeRepository resolves and authorizes the transaction participant.
//   - MessageRepository persists messages and read state.
type TradeMessageUsecase struct {
	tradeRepo   tradedom.Repository
	messageRepo tradedom.MessageRepository
	now         func() time.Time
}

func NewTradeMessageUsecase(
	tradeRepo tradedom.Repository,
	messageRepo tradedom.MessageRepository,
) *TradeMessageUsecase {
	return &TradeMessageUsecase{
		tradeRepo:   tradeRepo,
		messageRepo: messageRepo,
		now:         time.Now,
	}
}

// ============================================================
// Participant
// ============================================================

type TradeMessageParticipant struct {
	Trade tradedom.Trade
	Side  tradedom.MessageSenderSide
}

// ResolveParticipant resolves one authenticated Avatar as the buyer or seller
// of a Resale Trade.
//
// An Avatar that does not belong to the Trade receives tradedom.ErrNotFound so
// callers do not expose the existence of another user's private Trade.
func (u *TradeMessageUsecase) ResolveParticipant(
	ctx context.Context,
	tradeID string,
	avatarID string,
) (TradeMessageParticipant, error) {
	if u == nil || u.tradeRepo == nil || u.messageRepo == nil {
		return TradeMessageParticipant{}, ErrTradeMessageUsecaseNotConfigured
	}
	if tradeID == "" {
		return TradeMessageParticipant{}, tradedom.ErrInvalidID
	}
	if avatarID == "" {
		return TradeMessageParticipant{}, ErrTradeMessageAvatarIDEmpty
	}

	trade, err := u.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return TradeMessageParticipant{}, err
	}

	if trade.ID != tradeID {
		return TradeMessageParticipant{}, tradedom.ErrNotFound
	}
	if trade.SellerType != tradedom.SellerTypeAvatar || trade.SellerAvatarID == "" {
		return TradeMessageParticipant{}, ErrTradeMessageUnsupportedTrade
	}

	side, err := resolveTradeMessageParticipantSide(trade, avatarID)
	if err != nil {
		return TradeMessageParticipant{}, err
	}

	return TradeMessageParticipant{
		Trade: trade,
		Side:  side,
	}, nil
}

// ============================================================
// Create
// ============================================================

type CreateTradeMessageInput struct {
	TradeID  string
	AvatarID string
	Content  string
}

// CreateMessage creates one text message from the authenticated Avatar.
//
// SenderSide is derived from the persisted Trade:
//   - Trade.BuyerAvatarID == AvatarID -> buyer
//   - Trade.SellerAvatarID == AvatarID -> seller
//
// SenderType is always avatar because Trade is currently limited to Resale
// transactions between Mall Avatars.
//
// Images are intentionally not accepted here yet. Image messages should be
// introduced together with an authenticated upload/storage flow rather than
// accepting arbitrary file URLs or object paths from clients.
func (u *TradeMessageUsecase) CreateMessage(
	ctx context.Context,
	in CreateTradeMessageInput,
) (tradedom.Message, error) {
	participant, err := u.ResolveParticipant(
		ctx,
		in.TradeID,
		in.AvatarID,
	)
	if err != nil {
		return tradedom.Message{}, err
	}

	if participant.Trade.Status == tradedom.StatusClosed {
		return tradedom.Message{}, tradedom.ErrTradeAlreadyClosed
	}
	if participant.Trade.Status != tradedom.StatusActive {
		return tradedom.Message{}, tradedom.ErrInvalidStatus
	}

	message, err := tradedom.NewMessageForCreate(
		"",
		participant.Trade.ID,
		participant.Side,
		tradedom.MessageSenderTypeAvatar,
		in.AvatarID,
		in.Content,
		nil,
	)
	if err != nil {
		return tradedom.Message{}, err
	}

	created, err := u.messageRepo.Create(
		ctx,
		message,
	)
	if err != nil {
		return tradedom.Message{}, err
	}

	return created, nil
}

// ============================================================
// List
// ============================================================

type ListTradeMessagesInput struct {
	TradeID  string
	AvatarID string

	Limit int

	BeforeCreatedAt *time.Time
	AfterCreatedAt  *time.Time
}

// ListMessages returns messages only after confirming that the authenticated
// Avatar is a participant of the Trade.
func (u *TradeMessageUsecase) ListMessages(
	ctx context.Context,
	in ListTradeMessagesInput,
) ([]tradedom.Message, error) {
	participant, err := u.ResolveParticipant(
		ctx,
		in.TradeID,
		in.AvatarID,
	)
	if err != nil {
		return nil, err
	}

	return u.messageRepo.ListByTradeID(
		ctx,
		participant.Trade.ID,
		tradedom.MessageListFilter{
			Limit:           in.Limit,
			BeforeCreatedAt: in.BeforeCreatedAt,
			AfterCreatedAt:  in.AfterCreatedAt,
		},
	)
}

// ============================================================
// Read state
// ============================================================

type MarkTradeMessagesReadInput struct {
	TradeID  string
	AvatarID string
}

// MarkRead marks messages from the opposite side as read by the authenticated
// Avatar.
//
// readAt is generated on the server and is never accepted from the client.
func (u *TradeMessageUsecase) MarkRead(
	ctx context.Context,
	in MarkTradeMessagesReadInput,
) error {
	participant, err := u.ResolveParticipant(
		ctx,
		in.TradeID,
		in.AvatarID,
	)
	if err != nil {
		return err
	}

	if u.now == nil {
		return ErrTradeMessageUsecaseNotConfigured
	}

	readAt := u.now().UTC()

	switch participant.Side {
	case tradedom.MessageSenderSideBuyer:
		return u.messageRepo.MarkReadByBuyer(
			ctx,
			participant.Trade.ID,
			readAt,
		)

	case tradedom.MessageSenderSideSeller:
		return u.messageRepo.MarkReadBySeller(
			ctx,
			participant.Trade.ID,
			readAt,
		)

	default:
		return tradedom.ErrInvalidMessageSenderSide
	}
}

// ============================================================
// Unread count
// ============================================================

type CountUnreadTradeMessagesInput struct {
	TradeID  string
	AvatarID string
}

// CountUnread returns the unread message count for the authenticated Avatar.
func (u *TradeMessageUsecase) CountUnread(
	ctx context.Context,
	in CountUnreadTradeMessagesInput,
) (int, error) {
	participant, err := u.ResolveParticipant(
		ctx,
		in.TradeID,
		in.AvatarID,
	)
	if err != nil {
		return 0, err
	}

	switch participant.Side {
	case tradedom.MessageSenderSideBuyer:
		return u.messageRepo.CountUnreadForBuyer(
			ctx,
			participant.Trade.ID,
		)

	case tradedom.MessageSenderSideSeller:
		return u.messageRepo.CountUnreadForSeller(
			ctx,
			participant.Trade.ID,
		)

	default:
		return 0, tradedom.ErrInvalidMessageSenderSide
	}
}

// ============================================================
// Internal
// ============================================================

func resolveTradeMessageParticipantSide(
	trade tradedom.Trade,
	avatarID string,
) (tradedom.MessageSenderSide, error) {
	if avatarID == "" {
		return "", ErrTradeMessageAvatarIDEmpty
	}

	if avatarID == trade.BuyerAvatarID {
		return tradedom.MessageSenderSideBuyer, nil
	}

	if avatarID == trade.SellerAvatarID {
		return tradedom.MessageSenderSideSeller, nil
	}

	return "", tradedom.ErrNotFound
}
