// backend/internal/application/query/dto/list_detail_dto.go
package dto

// ListDetailDTO is a screen DTO for listDetail page.
//
// Current requirements (frontend/console/list):
// - From List: id, inventoryId, title, description, status/decision, assigneeId, createdAt, imageUrls, priceRows
// - From ProductBlueprint: productName, productBrandName
// - From TokenBlueprint: tokenName, tokenBrandName
//
// Note:
// - brandId fields are optional but useful for future expansion/debugging.
type ListDetailDTO struct {
	// List identity
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`

	// status/decision
	// - frontend normalizes decision from decision/status
	Status   string `json:"status,omitempty"`
	Decision string `json:"decision,omitempty"`

	// listing content
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// assignee
	AssigneeID   string `json:"assigneeId,omitempty"`
	AssigneeName string `json:"assigneeName,omitempty"`

	// audit
	CreatedBy     string `json:"createdBy,omitempty"`
	CreatedByName string `json:"createdByName,omitempty"` // ✅ createdBy の表示名
	CreatedAt     string `json:"createdAt,omitempty"`

	// ✅ NEW: 更新者（UID）と表示名
	UpdatedBy     string `json:"updatedBy,omitempty"`
	UpdatedByName string `json:"updatedByName,omitempty"`

	UpdatedAt string `json:"updatedAt,omitempty"`

	// primary image
	ImageID string `json:"imageId,omitempty"`

	// ids derived from inventoryId
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// product
	ProductBrandID   string `json:"productBrandId,omitempty"`
	ProductBrandName string `json:"productBrandName,omitempty"`
	ProductName      string `json:"productName,omitempty"`

	// token
	TokenBrandID   string `json:"tokenBrandId,omitempty"`
	TokenBrandName string `json:"tokenBrandName,omitempty"`
	TokenName      string `json:"tokenName,omitempty"`

	// images
	ImageURLs []string `json:"imageUrls,omitempty"`

	// price (PriceCard)
	PriceRows   []ListDetailPriceRowDTO `json:"priceRows,omitempty"`
	TotalStock  int                     `json:"totalStock,omitempty"`
	PriceNote   string                  `json:"priceNote,omitempty"`
	CurrencyJPY bool                    `json:"currencyJpy,omitempty"`
}

// ListDetailPriceRowDTO is a row DTO for PriceCard in listDetail.
type ListDetailPriceRowDTO struct {
	ModelID string `json:"modelId"`

	// In list detail, stock is still shown
	Stock int `json:"stock"`

	// Display
	Size  string `json:"size"`
	Color string `json:"color"`
	RGB   *int   `json:"rgb,omitempty"`

	// Price (nullable)
	Price *int `json:"price,omitempty"`
}
