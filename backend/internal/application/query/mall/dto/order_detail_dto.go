// backend\internal\application\query\mall\dto\order_detail_dto.go
package dto

import orderdom "narratives/internal/domain/order"

type OrderDetail struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingQuoteSnapshot orderdom.ShippingQuoteSnapshot `json:"shippingQuoteSnapshot"`

	SubtotalAmount int `json:"subtotalAmount"`
	ShippingAmount int `json:"shippingAmount"`
	ConsumptionTax int `json:"consumptionTax"`
	TotalAmount    int `json:"totalAmount"`

	Paid bool `json:"paid"`

	RefundStatus   string `json:"refundStatus"`
	RefundedAmount int    `json:"refundedAmount"`
	RefundedAt     string `json:"refundedAt,omitempty"`

	Items []OrderDetailItem `json:"items"`

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type OrderDetailColor struct {
	Name string `json:"name,omitempty"`
	RGB  int    `json:"rgb,omitempty"`
}

type OrderDetailItem struct {
	ItemType string `json:"itemType,omitempty"`

	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ResaleID    string `json:"resaleId,omitempty"`

	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`

	BrandID   string `json:"brandId,omitempty"`
	BrandName string `json:"brandName,omitempty"`
	BrandIcon string `json:"brandIcon,omitempty"`

	TokenName string `json:"tokenName,omitempty"`
	TokenIcon string `json:"tokenIcon,omitempty"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath,omitempty"`
	ConsumptionTaxRate           int      `json:"consumptionTaxRate"`

	Kind         string            `json:"kind,omitempty"`
	ModelNumber  string            `json:"modelNumber,omitempty"`
	Size         string            `json:"size,omitempty"`
	Color        *OrderDetailColor `json:"color,omitempty"`
	Measurements map[string]int    `json:"measurements,omitempty"`

	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	IsReturnRequested bool                       `json:"isReturnRequested"`
	ReturnRequestKind orderdom.ReturnRequestKind `json:"returnRequestKind,omitempty"`
	ReturnRequestedAt string                     `json:"returnRequestedAt,omitempty"`

	IsReturnCompleted bool   `json:"isReturnCompleted"`
	ReturnCompletedAt string `json:"returnCompletedAt,omitempty"`

	TokenTransferVerifiedAt string `json:"tokenTransferVerifiedAt,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
}
