// backend/internal/application/query/mall/dto/history_dto.go
package dto

import (
	"narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

type HistoryOrderPage = common.PageResult[HistoryOrder]

type EnrichHistoryOrderPageInput = HistoryOrderPage

type HistoryOrder struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot      orderdom.ShippingSnapshot      `json:"shippingSnapshot"`
	ShippingQuoteSnapshot orderdom.ShippingQuoteSnapshot `json:"shippingQuoteSnapshot"`
	PaymentMethodSnapshot orderdom.PaymentMethodSnapshot `json:"paymentMethodSnapshot"`

	Paid  bool               `json:"paid"`
	Items []HistoryOrderItem `json:"items"`

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type HistoryOrderItem struct {
	Type orderdom.OrderItemType `json:"type"`

	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	ResaleID string `json:"resaleId,omitempty"`

	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`

	BrandID string `json:"brandId,omitempty"`

	TokenName string `json:"tokenName,omitempty"`
	TokenIcon string `json:"tokenIcon,omitempty"`

	BrandName string `json:"brandName,omitempty"`
	BrandIcon string `json:"brandIcon,omitempty"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	IsReturnRequested bool   `json:"isReturnRequested"`
	ReturnRequestedAt string `json:"returnRequestedAt,omitempty"`

	IsReturnCompleted bool   `json:"isReturnCompleted"`
	ReturnCompletedAt string `json:"returnCompletedAt,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
}
