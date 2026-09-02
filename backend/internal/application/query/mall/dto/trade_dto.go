// backend/internal/application/query/mall/dto/trade_dto.go
package dto

import (
	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

// TradeDetail is the buyer/seller private Trade view returned to Mall clients.
//
// Trade is currently limited to Resale transactions:
//
//	buyer Avatar <-> seller Avatar
//
// ViewerSide is resolved by the backend from the authenticated Avatar and must
// not be supplied by the client.
//
// Cancellation, dispatch, return, and transfer state are read from the
// authoritative Order item associated with this Trade and are not owned by
// the Trade aggregate.
type TradeDetail struct {
	ID             string `json:"id"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	ViewerSide tradedom.MessageSenderSide `json:"viewerSide"`

	ProductName string `json:"productName,omitempty"`

	BuyerAvatarID   string `json:"buyerAvatarId"`
	BuyerAvatarName string `json:"buyerAvatarName,omitempty"`
	BuyerAvatarIcon string `json:"buyerAvatarIcon,omitempty"`

	SellerAvatarID   string `json:"sellerAvatarId"`
	SellerAvatarName string `json:"sellerAvatarName,omitempty"`
	SellerAvatarIcon string `json:"sellerAvatarIcon,omitempty"`

	Status tradedom.Status `json:"status"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	IsReturnRequested bool                       `json:"isReturnRequested"`
	ReturnRequestKind orderdom.ReturnRequestKind `json:"returnRequestKind,omitempty"`
	IsReturnCompleted bool                       `json:"isReturnCompleted"`
	Transferred       bool                       `json:"transferred"`

	ReturnRequestedAt string `json:"returnRequestedAt,omitempty"`
	ReturnCompletedAt string `json:"returnCompletedAt,omitempty"`
	TransferredAt     string `json:"transferredAt,omitempty"`

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
