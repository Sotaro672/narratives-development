// backend/internal/application/query/mall/dto/trade_dto.go
package dto

import tradedom "narratives/internal/domain/trade"

// TradeDetail is the buyer/seller private Trade view returned to Mall clients.
//
// Trade is currently limited to Resale transactions:
//
//	buyer Avatar <-> seller Avatar
//
// ViewerSide is resolved by the backend from the authenticated Avatar and must
// not be supplied by the client.
type TradeDetail struct {
	ID             string `json:"id"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	ViewerSide tradedom.MessageSenderSide `json:"viewerSide"`

	BuyerAvatarID  string `json:"buyerAvatarId"`
	SellerAvatarID string `json:"sellerAvatarId"`

	Status tradedom.Status `json:"status"`

	Messages []TradeMessage `json:"messages"`

	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
	LastMessageAt string `json:"lastMessageAt,omitempty"`
}

// TradeMessage represents one message in a Trade thread.
//
// SenderSide is sufficient for TradePage to distinguish:
//   - buyer
//   - seller
//   - system
//
// SenderType and SenderID are retained so the response also preserves the
// authoritative sender identity stored by the Trade domain.
type TradeMessage struct {
	ID      string `json:"id"`
	TradeID string `json:"tradeId"`

	SenderSide tradedom.MessageSenderSide `json:"senderSide"`
	SenderType tradedom.MessageSenderType `json:"senderType"`
	SenderID   string                     `json:"senderId"`

	Content string `json:"content,omitempty"`

	BuyerReadAt  string `json:"buyerReadAt,omitempty"`
	SellerReadAt string `json:"sellerReadAt,omitempty"`

	CreatedAt string `json:"createdAt"`
}
