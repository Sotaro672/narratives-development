// backend/internal/application/query/mall/trade_query.go
package mall

import (
	"context"
	"errors"
	"time"

	tradedto "narratives/internal/application/query/mall/dto"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeQueryNotConfigured    = errors.New("mall trade query: not configured")
	ErrTradeQueryAvatarIDEmpty    = errors.New("mall trade query: avatarId is empty")
	ErrTradeQueryUnsupportedTrade = errors.New("mall trade query: unsupported trade")
)

// TradeQuery provides the read model required by the Mall TradePage.
//
// Trade is currently limited to Resale transactions:
//
//	buyer Avatar <-> seller Avatar
//
// Authorization is based only on the authenticated Avatar ID supplied from
// AvatarContext. A caller that is not a participant receives ErrNotFound so the
// existence of another user's private Trade is not exposed.
type TradeQuery struct {
	tradeRepo   tradedom.Repository
	messageRepo tradedom.MessageRepository
}

func NewTradeQuery(
	tradeRepo tradedom.Repository,
	messageRepo tradedom.MessageRepository,
) *TradeQuery {
	return &TradeQuery{
		tradeRepo:   tradeRepo,
		messageRepo: messageRepo,
	}
}

type GetTradeByOrderItemInput struct {
	AvatarID       string
	OrderID        string
	OrderItemIndex int

	MessageLimit int

	BeforeCreatedAt *time.Time
	AfterCreatedAt  *time.Time
}

// GetByOrderItem returns the Trade and its message thread for one Resale
// OrderItem.
//
// The caller must supply AvatarID from authenticated AvatarContext rather than
// request body or query parameters.
func (q *TradeQuery) GetByOrderItem(
	ctx context.Context,
	in GetTradeByOrderItemInput,
) (tradedto.TradeDetail, error) {
	if q == nil || q.tradeRepo == nil || q.messageRepo == nil {
		return tradedto.TradeDetail{}, ErrTradeQueryNotConfigured
	}
	if in.AvatarID == "" {
		return tradedto.TradeDetail{}, ErrTradeQueryAvatarIDEmpty
	}
	if in.OrderID == "" {
		return tradedto.TradeDetail{}, tradedom.ErrInvalidOrderID
	}
	if in.OrderItemIndex < 0 {
		return tradedto.TradeDetail{}, tradedom.ErrInvalidOrderItemIndex
	}

	trade, err := q.tradeRepo.GetByOrderItem(
		ctx,
		in.OrderID,
		in.OrderItemIndex,
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}

	if trade.OrderID != in.OrderID ||
		trade.OrderItemIndex != in.OrderItemIndex {
		return tradedto.TradeDetail{}, tradedom.ErrNotFound
	}

	viewerSide, err := resolveTradeViewerSide(
		trade,
		in.AvatarID,
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}

	messages, err := q.messageRepo.ListByTradeID(
		ctx,
		trade.ID,
		tradedom.MessageListFilter{
			Limit:           in.MessageLimit,
			BeforeCreatedAt: in.BeforeCreatedAt,
			AfterCreatedAt:  in.AfterCreatedAt,
		},
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}

	return buildTradeDetailDTO(
		trade,
		viewerSide,
		messages,
	), nil
}

// GetByID returns the Trade and its message thread when the Trade ID is already
// known. This is useful after TradePage has resolved the Trade from
// orderId + orderItemIndex and for future direct Trade routes.
type GetTradeByIDInput struct {
	AvatarID string
	TradeID  string

	MessageLimit int

	BeforeCreatedAt *time.Time
	AfterCreatedAt  *time.Time
}

func (q *TradeQuery) GetByID(
	ctx context.Context,
	in GetTradeByIDInput,
) (tradedto.TradeDetail, error) {
	if q == nil || q.tradeRepo == nil || q.messageRepo == nil {
		return tradedto.TradeDetail{}, ErrTradeQueryNotConfigured
	}
	if in.AvatarID == "" {
		return tradedto.TradeDetail{}, ErrTradeQueryAvatarIDEmpty
	}
	if in.TradeID == "" {
		return tradedto.TradeDetail{}, tradedom.ErrInvalidID
	}

	trade, err := q.tradeRepo.GetByID(
		ctx,
		in.TradeID,
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}
	if trade.ID != in.TradeID {
		return tradedto.TradeDetail{}, tradedom.ErrNotFound
	}

	viewerSide, err := resolveTradeViewerSide(
		trade,
		in.AvatarID,
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}

	messages, err := q.messageRepo.ListByTradeID(
		ctx,
		trade.ID,
		tradedom.MessageListFilter{
			Limit:           in.MessageLimit,
			BeforeCreatedAt: in.BeforeCreatedAt,
			AfterCreatedAt:  in.AfterCreatedAt,
		},
	)
	if err != nil {
		return tradedto.TradeDetail{}, err
	}

	return buildTradeDetailDTO(
		trade,
		viewerSide,
		messages,
	), nil
}

func resolveTradeViewerSide(
	trade tradedom.Trade,
	avatarID string,
) (tradedom.MessageSenderSide, error) {
	if avatarID == "" {
		return "", ErrTradeQueryAvatarIDEmpty
	}

	if trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" {
		return "", ErrTradeQueryUnsupportedTrade
	}

	if avatarID == trade.BuyerAvatarID {
		return tradedom.MessageSenderSideBuyer, nil
	}

	if avatarID == trade.SellerAvatarID {
		return tradedom.MessageSenderSideSeller, nil
	}

	return "", tradedom.ErrNotFound
}

func buildTradeDetailDTO(
	trade tradedom.Trade,
	viewerSide tradedom.MessageSenderSide,
	messages []tradedom.Message,
) tradedto.TradeDetail {
	messageDTOs := make(
		[]tradedto.TradeMessage,
		0,
		len(messages),
	)

	for _, message := range messages {
		messageDTOs = append(
			messageDTOs,
			buildTradeMessageDTO(message),
		)
	}

	out := tradedto.TradeDetail{
		ID:             trade.ID,
		OrderID:        trade.OrderID,
		OrderItemIndex: trade.OrderItemIndex,
		ViewerSide:     viewerSide,
		BuyerAvatarID:  trade.BuyerAvatarID,
		SellerAvatarID: trade.SellerAvatarID,
		Status:         trade.Status,
		Messages:       messageDTOs,
	}

	if !trade.CreatedAt.IsZero() {
		out.CreatedAt = trade.CreatedAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	if !trade.UpdatedAt.IsZero() {
		out.UpdatedAt = trade.UpdatedAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	if trade.LastMessageAt != nil &&
		!trade.LastMessageAt.IsZero() {
		out.LastMessageAt = trade.LastMessageAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	return out
}

func buildTradeMessageDTO(
	message tradedom.Message,
) tradedto.TradeMessage {
	out := tradedto.TradeMessage{
		ID:         message.ID,
		TradeID:    message.TradeID,
		SenderSide: message.SenderSide,
		SenderType: message.SenderType,
		SenderID:   message.SenderID,
		Content:    message.Content,
	}

	if !message.CreatedAt.IsZero() {
		out.CreatedAt = message.CreatedAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	if message.BuyerReadAt != nil &&
		!message.BuyerReadAt.IsZero() {
		out.BuyerReadAt = message.BuyerReadAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	if message.SellerReadAt != nil &&
		!message.SellerReadAt.IsZero() {
		out.SellerReadAt = message.SellerReadAt.
			UTC().
			Format(time.RFC3339Nano)
	}

	return out
}
