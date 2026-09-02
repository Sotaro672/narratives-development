// backend/internal/application/query/mall/trade_query.go
package mall

import (
	"context"
	"errors"
	"sort"
	"time"

	tradedto "narratives/internal/application/query/mall/dto"
	mallshared "narratives/internal/application/query/mall/shared"
	avatardom "narratives/internal/domain/avatar"
	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeQueryNotConfigured    = errors.New("mall trade query: not configured")
	ErrTradeQueryAvatarIDEmpty    = errors.New("mall trade query: avatarId is empty")
	ErrTradeQueryUnsupportedTrade = errors.New("mall trade query: unsupported trade")
)

// TradeQuery provides the read models required by Mall ChatListPage and
// ChatDetailPage.
//
// Trade is limited to Resale transactions:
//
//	buyer Avatar <-> seller Avatar
//
// Authorization is based only on the authenticated Avatar ID supplied from
// AvatarContext. A caller that is not a participant receives ErrNotFound so the
// existence of another user's private Trade is not exposed.
//
// Cancellation and dispatch state are read from the authoritative Order item.
// Trade does not own or duplicate those states.
type TradeQuery struct {
	tradeRepo       tradedom.Repository
	messageRepo     tradedom.MessageRepository
	orderRepo       orderdom.Repository
	displayResolver mallshared.MallDisplayResolver
	avatarRepo      avatardom.Repository
}

func NewTradeQuery(
	tradeRepo tradedom.Repository,
	messageRepo tradedom.MessageRepository,
	orderRepo orderdom.Repository,
	displayResolver mallshared.MallDisplayResolver,
	avatarRepo avatardom.Repository,
) *TradeQuery {
	return &TradeQuery{
		tradeRepo:       tradeRepo,
		messageRepo:     messageRepo,
		orderRepo:       orderRepo,
		displayResolver: displayResolver,
		avatarRepo:      avatarRepo,
	}
}

// TradeListItem is the ChatListPage read model for one Trade.
//
// ViewerSide and CounterpartAvatarID are derived from the authenticated Avatar.
// LatestMessage and UnreadMessageCount are aggregated from Trade messages.
type TradeListItem struct {
	ID             string `json:"id"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	ViewerSide tradedom.MessageSenderSide `json:"viewerSide"`

	ProductName string `json:"productName,omitempty"`

	CounterpartAvatarID   string `json:"counterpartAvatarId"`
	CounterpartAvatarName string `json:"counterpartAvatarName,omitempty"`
	CounterpartAvatarIcon string `json:"counterpartAvatarIcon,omitempty"`

	Status tradedom.Status `json:"status"`

	LatestMessage      *tradedto.TradeMessage `json:"latestMessage,omitempty"`
	UnreadMessageCount int                    `json:"unreadMessageCount"`
	LatestActivityAt   string                 `json:"latestActivityAt"`

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// TradeListResult is the Trade portion of the Mall chat list response.
type TradeListResult struct {
	Items []TradeListItem `json:"items"`
}

// ListForAvatar returns active Resale Trades in which the authenticated Avatar
// participates as buyer or seller.
//
// Cancelled Trade Order items are excluded from ChatListPage. Cancellation is
// read from the authoritative Order item rather than duplicated on Trade.
//
// The repository resolves both participant roles. This query enriches each
// Trade with viewer-side information, latest message, unread count and latest
// activity time, then orders the result newest-first for ChatListPage.
func (q *TradeQuery) ListForAvatar(
	ctx context.Context,
	avatarID string,
) (TradeListResult, error) {
	if q == nil ||
		q.tradeRepo == nil ||
		q.messageRepo == nil ||
		q.orderRepo == nil ||
		q.displayResolver == nil ||
		q.avatarRepo == nil {
		return TradeListResult{}, ErrTradeQueryNotConfigured
	}
	if avatarID == "" {
		return TradeListResult{}, ErrTradeQueryAvatarIDEmpty
	}

	trades, err := q.tradeRepo.ListByAvatarID(ctx, avatarID)
	if err != nil {
		return TradeListResult{}, err
	}

	items := make([]TradeListItem, 0, len(trades))
	for _, trade := range trades {
		viewerSide, err := resolveTradeViewerSide(trade, avatarID)
		if err != nil {
			return TradeListResult{}, err
		}

		orderItemState, err := q.getTradeOrderItemState(
			ctx,
			trade,
		)
		if err != nil {
			return TradeListResult{}, err
		}
		if orderItemState.IsCancelled {
			continue
		}

		latestMessage, err := q.getLatestMessage(ctx, trade.ID)
		if err != nil {
			return TradeListResult{}, err
		}

		unreadMessageCount, err := q.countUnreadMessages(ctx, trade.ID, viewerSide)
		if err != nil {
			return TradeListResult{}, err
		}

		display := q.resolveTradeDisplay(
			ctx,
			trade,
			orderItemState.ProductBlueprintID,
		)

		items = append(items, buildTradeListItem(
			trade,
			viewerSide,
			display,
			latestMessage,
			unreadMessageCount,
		))
	}

	sort.SliceStable(items, func(i, j int) bool {
		return tradeListItemActivityTime(items[i]).
			After(tradeListItemActivityTime(items[j]))
	})

	return TradeListResult{
		Items: items,
	}, nil
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
// This is primarily used when an OrderDetail item has orderId + itemIndex and
// needs to resolve the corresponding Trade before opening ChatDetailPage.
//
// The caller must supply AvatarID from authenticated AvatarContext rather than
// request body or query parameters.
func (q *TradeQuery) GetByOrderItem(
	ctx context.Context,
	in GetTradeByOrderItemInput,
) (tradedto.TradeDetail, error) {
	if q == nil ||
		q.tradeRepo == nil ||
		q.messageRepo == nil ||
		q.orderRepo == nil ||
		q.displayResolver == nil ||
		q.avatarRepo == nil {
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

	orderItemState, err := q.getTradeOrderItemState(
		ctx,
		trade,
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

	display := q.resolveTradeDisplay(
		ctx,
		trade,
		orderItemState.ProductBlueprintID,
	)

	return buildTradeDetailDTO(
		trade,
		viewerSide,
		orderItemState,
		display,
		messages,
	), nil
}

type GetTradeByIDInput struct {
	AvatarID string
	TradeID  string

	MessageLimit int

	BeforeCreatedAt *time.Time
	AfterCreatedAt  *time.Time
}

// GetByID returns the Trade and its message thread for ChatDetailPage.
func (q *TradeQuery) GetByID(
	ctx context.Context,
	in GetTradeByIDInput,
) (tradedto.TradeDetail, error) {
	if q == nil ||
		q.tradeRepo == nil ||
		q.messageRepo == nil ||
		q.orderRepo == nil ||
		q.displayResolver == nil ||
		q.avatarRepo == nil {
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

	orderItemState, err := q.getTradeOrderItemState(
		ctx,
		trade,
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

	display := q.resolveTradeDisplay(
		ctx,
		trade,
		orderItemState.ProductBlueprintID,
	)

	return buildTradeDetailDTO(
		trade,
		viewerSide,
		orderItemState,
		display,
		messages,
	), nil
}

type tradeOrderItemState struct {
	IsCancelled        bool
	IsDispatched       bool
	ProductBlueprintID string
}

func (q *TradeQuery) getTradeOrderItemState(
	ctx context.Context,
	trade tradedom.Trade,
) (tradeOrderItemState, error) {
	if q == nil || q.orderRepo == nil {
		return tradeOrderItemState{}, ErrTradeQueryNotConfigured
	}
	if trade.OrderID == "" {
		return tradeOrderItemState{}, tradedom.ErrInvalidOrderID
	}
	if trade.OrderItemIndex < 0 {
		return tradeOrderItemState{}, tradedom.ErrInvalidOrderItemIndex
	}

	order, err := q.orderRepo.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return tradeOrderItemState{}, err
	}

	if order.ID != trade.OrderID ||
		order.AvatarID != trade.BuyerAvatarID ||
		trade.OrderItemIndex >= len(order.Items) {
		return tradeOrderItemState{}, ErrTradeQueryUnsupportedTrade
	}

	item := order.Items[trade.OrderItemIndex]
	if item.Type != orderdom.OrderItemTypeResale ||
		item.SellerSnapshot.AvatarID == "" ||
		item.SellerSnapshot.AvatarID != trade.SellerAvatarID {
		return tradeOrderItemState{}, ErrTradeQueryUnsupportedTrade
	}

	return tradeOrderItemState{
		IsCancelled:        item.IsCancelled,
		IsDispatched:       item.IsDispatched,
		ProductBlueprintID: item.ProductBlueprintID,
	}, nil
}

func (q *TradeQuery) getLatestMessage(
	ctx context.Context,
	tradeID string,
) (*tradedto.TradeMessage, error) {
	messages, err := q.messageRepo.ListByTradeID(
		ctx,
		tradeID,
		tradedom.MessageListFilter{
			Limit: 1,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}

	latest := buildTradeMessageDTO(
		messages[len(messages)-1],
	)

	return &latest, nil
}

func (q *TradeQuery) countUnreadMessages(
	ctx context.Context,
	tradeID string,
	viewerSide tradedom.MessageSenderSide,
) (int, error) {
	switch viewerSide {
	case tradedom.MessageSenderSideBuyer:
		return q.messageRepo.CountUnreadForBuyer(
			ctx,
			tradeID,
		)

	case tradedom.MessageSenderSideSeller:
		return q.messageRepo.CountUnreadForSeller(
			ctx,
			tradeID,
		)

	default:
		return 0, ErrTradeQueryUnsupportedTrade
	}
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

type tradeDisplay struct {
	ProductName string

	BuyerAvatarName string
	BuyerAvatarIcon string

	SellerAvatarName string
	SellerAvatarIcon string
}

func (q *TradeQuery) resolveTradeDisplay(
	ctx context.Context,
	trade tradedom.Trade,
	productBlueprintID string,
) tradeDisplay {
	out := tradeDisplay{}

	if q != nil && q.displayResolver != nil && productBlueprintID != "" {
		productBlueprint, err := q.displayResolver.ResolveProductBlueprintInfo(
			ctx,
			productBlueprintID,
		)
		if err == nil {
			out.ProductName = productBlueprint.ProductName
		}
	}

	if q == nil || q.avatarRepo == nil {
		return out
	}

	if trade.BuyerAvatarID != "" {
		buyerAvatar, err := q.avatarRepo.GetByID(ctx, trade.BuyerAvatarID)
		if err == nil {
			out.BuyerAvatarName = buyerAvatar.AvatarName
			if buyerAvatar.AvatarIcon != nil {
				out.BuyerAvatarIcon = *buyerAvatar.AvatarIcon
			}
		}
	}

	if trade.SellerAvatarID != "" {
		sellerAvatar, err := q.avatarRepo.GetByID(ctx, trade.SellerAvatarID)
		if err == nil {
			out.SellerAvatarName = sellerAvatar.AvatarName
			if sellerAvatar.AvatarIcon != nil {
				out.SellerAvatarIcon = *sellerAvatar.AvatarIcon
			}
		}
	}

	return out
}

func buildTradeListItem(
	trade tradedom.Trade,
	viewerSide tradedom.MessageSenderSide,
	display tradeDisplay,
	latestMessage *tradedto.TradeMessage,
	unreadMessageCount int,
) TradeListItem {
	counterpartAvatarID := trade.SellerAvatarID
	counterpartAvatarName := display.SellerAvatarName
	counterpartAvatarIcon := display.SellerAvatarIcon

	if viewerSide == tradedom.MessageSenderSideSeller {
		counterpartAvatarID = trade.BuyerAvatarID
		counterpartAvatarName = display.BuyerAvatarName
		counterpartAvatarIcon = display.BuyerAvatarIcon
	}

	latestActivityAt := tradeLatestActivityAt(trade)

	out := TradeListItem{
		ID:                    trade.ID,
		OrderID:               trade.OrderID,
		OrderItemIndex:        trade.OrderItemIndex,
		ViewerSide:            viewerSide,
		ProductName:           display.ProductName,
		CounterpartAvatarID:   counterpartAvatarID,
		CounterpartAvatarName: counterpartAvatarName,
		CounterpartAvatarIcon: counterpartAvatarIcon,
		Status:                trade.Status,
		LatestMessage:         latestMessage,
		UnreadMessageCount:    unreadMessageCount,
	}

	if !latestActivityAt.IsZero() {
		out.LatestActivityAt = latestActivityAt.
			UTC().
			Format(time.RFC3339Nano)
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

	return out
}

func tradeLatestActivityAt(
	trade tradedom.Trade,
) time.Time {
	latest := trade.CreatedAt

	if trade.UpdatedAt.After(latest) {
		latest = trade.UpdatedAt
	}
	if trade.LastMessageAt != nil &&
		trade.LastMessageAt.After(latest) {
		latest = *trade.LastMessageAt
	}

	return latest.UTC()
}

func tradeListItemActivityTime(
	item TradeListItem,
) time.Time {
	if item.LatestActivityAt == "" {
		return time.Time{}
	}

	value, err := time.Parse(
		time.RFC3339Nano,
		item.LatestActivityAt,
	)
	if err != nil {
		return time.Time{}
	}

	return value
}

func buildTradeDetailDTO(
	trade tradedom.Trade,
	viewerSide tradedom.MessageSenderSide,
	orderItemState tradeOrderItemState,
	display tradeDisplay,
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
		ID:               trade.ID,
		OrderID:          trade.OrderID,
		OrderItemIndex:   trade.OrderItemIndex,
		ViewerSide:       viewerSide,
		ProductName:      display.ProductName,
		BuyerAvatarID:    trade.BuyerAvatarID,
		BuyerAvatarName:  display.BuyerAvatarName,
		BuyerAvatarIcon:  display.BuyerAvatarIcon,
		SellerAvatarID:   trade.SellerAvatarID,
		SellerAvatarName: display.SellerAvatarName,
		SellerAvatarIcon: display.SellerAvatarIcon,
		Status:           trade.Status,
		IsCancelled:      orderItemState.IsCancelled,
		IsDispatched:     orderItemState.IsDispatched,
		Messages:         messageDTOs,
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
