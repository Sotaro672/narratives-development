// backend/internal/application/query/mall/dto/history_dto.go
package dto

import (
	"narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

type HistoryResolveModelInput struct {
	ItemType string

	ModelID     string
	InventoryID string
	ListID      string

	ResaleID string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

type HistoryResolvedModel struct {
	ItemType string `json:"itemType,omitempty"`

	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	ResaleID string `json:"resaleId,omitempty"`

	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`

	BrandID string `json:"brandId,omitempty"`

	// model variation 共通
	Kind        string `json:"kind,omitempty"`
	ModelNumber string `json:"modelNumber,omitempty"`

	// apparel
	Size         string         `json:"size,omitempty"`
	Color        *HistoryColor  `json:"color,omitempty"`
	Measurements map[string]int `json:"measurements,omitempty"`

	// alcohol
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	TokenName string `json:"tokenName,omitempty"`
	TokenIcon string `json:"tokenIcon,omitempty"`

	BrandName string `json:"brandName,omitempty"`
	BrandIcon string `json:"brandIcon,omitempty"`
}

type HistoryColor struct {
	Name string `json:"name,omitempty"`
	Hex  string `json:"hex,omitempty"`

	// 必要なら frontend 側で rgb 表示にも使えるように保持
	RGB *int `json:"rgb,omitempty"`
}

type HistoryOrderPage = common.PageResult[HistoryOrder]

type EnrichHistoryOrderPageInput = HistoryOrderPage

type HistoryOrder struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot      orderdom.ShippingSnapshot      `json:"shippingSnapshot"`
	PaymentMethodSnapshot orderdom.PaymentMethodSnapshot `json:"paymentMethodSnapshot"`

	Paid  bool               `json:"paid"`
	Items []HistoryOrderItem `json:"items"`

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type HistoryOrderItem struct {
	ItemType string `json:"itemType,omitempty"`

	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	ResaleID string `json:"resaleId,omitempty"`

	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`

	BrandID string `json:"brandId,omitempty"`

	// model variation 共通
	Kind        string `json:"kind,omitempty"`
	ModelNumber string `json:"modelNumber,omitempty"`

	// apparel
	Size         string         `json:"size,omitempty"`
	Color        *HistoryColor  `json:"color,omitempty"`
	Measurements map[string]int `json:"measurements,omitempty"`

	// alcohol
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	TokenName string `json:"tokenName,omitempty"`
	TokenIcon string `json:"tokenIcon,omitempty"`

	BrandName string `json:"brandName,omitempty"`
	BrandIcon string `json:"brandIcon,omitempty"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCanceled   bool `json:"isCanceled"`
	IsDispatched bool `json:"isDispatched"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
}
