// backend/internal/application/query/mall/dto/trade_dto.go
package dto

import (
	resaledom "narratives/internal/domain/resale"
	tradedom "narratives/internal/domain/trade"
)

// TradeResaleImage represents one resale condition image displayed in Trade detail.
type TradeResaleImage struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	DisplayOrder int    `json:"displayOrder"`
}

// TradeResaleDetail represents the resale listing information required by
// TradeDetailPage.
//
// This read model intentionally contains only display information required by
// the Trade UI. The authoritative resale aggregate remains owned by the resale
// domain.
type TradeResaleDetail struct {
	ID          string                    `json:"id"`
	Condition   resaledom.ResaleCondition `json:"condition"`
	Description string                    `json:"description,omitempty"`

	ModelID      string                  `json:"modelId,omitempty"`
	Kind         string                  `json:"kind,omitempty"`
	ModelNumber  string                  `json:"modelNumber,omitempty"`
	Size         string                  `json:"size,omitempty"`
	Color        *resaledom.ResaleColor  `json:"color,omitempty"`
	Measurements map[string]int          `json:"measurements,omitempty"`
	Volume       *resaledom.ResaleVolume `json:"volume,omitempty"`

	Images []TradeResaleImage `json:"images"`
}

// TradeReturnConsultation represents the buyer's return consultation displayed
// in Trade detail.
//
// The authoritative consultation is owned by trade.ReturnAgreement.
type TradeReturnConsultation struct {
	ID     string                            `json:"id"`
	Reason tradedom.ReturnConsultationReason `json:"reason"`
	Detail string                            `json:"detail"`

	CreatedAt string `json:"createdAt"`
}

// TradeReturnProposal represents the seller's latest answer to a return
// consultation.
//
// When Agreement is "agree", ReturnRequirement and RefundAmount contain the
// conditions presented to the buyer.
//
// When Agreement is "disagree", those condition fields are omitted.
type TradeReturnProposal struct {
	ID string `json:"id"`

	Agreement         tradedom.ReturnProposalAgreement `json:"agreement"`
	ReturnRequirement tradedom.ReturnRequirement       `json:"returnRequirement,omitempty"`
	RefundAmount      int                              `json:"refundAmount,omitempty"`

	CreatedAt  string `json:"createdAt"`
	RejectedAt string `json:"rejectedAt,omitempty"`
}

// TradeDetail is the buyer/seller private Trade view returned to Mall clients.
//
// Trade is currently limited to Resale transactions:
//
//	buyer Avatar <-> seller Avatar
//
// ViewerSide is resolved by the backend from the authenticated Avatar and must
// not be supplied by the client.
//
// Cancellation, dispatch, transfer and refund-limit state are read from the
// authoritative Order item associated with this Trade.
//
// Return negotiation state is read from the authoritative ReturnAgreement.
// When no ReturnAgreement exists, ReturnStatus is returned as "none".
//
// ReturnShipmentStatus is the locally persisted reverse-logistics state for
// this Trade. It is omitted when no ReturnShipment exists. This value represents
// only AMOL's persisted shipment state and does not imply that carrier status
// has been queried or detected.
type TradeDetail struct {
	ID             string `json:"id"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	ViewerSide tradedom.MessageSenderSide `json:"viewerSide"`

	ProductName string            `json:"productName,omitempty"`
	Resale      TradeResaleDetail `json:"resale"`

	BuyerAvatarID   string `json:"buyerAvatarId"`
	BuyerAvatarName string `json:"buyerAvatarName,omitempty"`
	BuyerAvatarIcon string `json:"buyerAvatarIcon,omitempty"`

	SellerAvatarID   string `json:"sellerAvatarId"`
	SellerAvatarName string `json:"sellerAvatarName,omitempty"`
	SellerAvatarIcon string `json:"sellerAvatarIcon,omitempty"`

	Status tradedom.Status `json:"status"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	ReturnStatus         tradedom.ReturnStatus          `json:"returnStatus"`
	ReturnConsultation   *TradeReturnConsultation       `json:"returnConsultation,omitempty"`
	ReturnProposal       *TradeReturnProposal           `json:"returnProposal,omitempty"`
	ReturnShipmentStatus *tradedom.ReturnShipmentStatus `json:"returnShipmentStatus,omitempty"`

	MerchandiseRefundMaxAmount int `json:"merchandiseRefundMaxAmount"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`

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
