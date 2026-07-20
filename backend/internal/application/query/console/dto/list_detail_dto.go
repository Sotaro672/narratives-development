// backend/internal/application/query/console/dto/list_detail_dto.go
package dto

// ListDetailDTO is a screen DTO for listDetail page.
//
// Current requirements (frontend/console/list):
// - From List: id, inventoryId, title, description, status, assigneeId, createdAt, imageUrls, priceRows
// - From ProductBlueprint: productName, productBrandName
// - From TokenBlueprint: tokenName, tokenBrandName
//
// Note:
// - brandId fields are optional but useful for future expansion/debugging.
type ListDetailDTO struct {
	// List identity
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`
	// status
	Status string `json:"status,omitempty"`
	// listing content
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// assignee
	AssigneeID   string `json:"assigneeId,omitempty"`
	AssigneeName string `json:"assigneeName,omitempty"`
	// audit
	CreatedBy     string `json:"createdBy,omitempty"`
	CreatedByName string `json:"createdByName,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	// updater
	UpdatedBy     string `json:"updatedBy,omitempty"`
	UpdatedByName string `json:"updatedByName,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
	// primary image
	ImageID string `json:"imageId,omitempty"`
	// primary list image url (representative image)
	// - ImageID に対応する URL を返す（なければ空文字）
	ListImageURL string `json:"listImageUrl,omitempty"`
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
	ImageURLs []string       `json:"imageUrls,omitempty"`
	Images    []ListImageDTO `json:"images,omitempty"`
	// price (PriceCard)
	PriceRows   []ListDetailPriceRowDTO `json:"priceRows,omitempty"`
	TotalStock  int                     `json:"totalStock,omitempty"`
	PriceNote   string                  `json:"priceNote,omitempty"`
	CurrencyJPY bool                    `json:"currencyJpy,omitempty"`
}

// ListImageDTO is a list image record DTO for listDetail page.
//
// Firebase Storage policy:
// - ID/ImageID is the image record document ID.
// - URL is Firebase Storage downloadURL.
// - DisplayOrder is the display order in the list image record.
type ListImageDTO struct {
	ID           string `json:"id"`
	ImageID      string `json:"imageId"`
	URL          string `json:"url"`
	DisplayOrder int    `json:"displayOrder"`
}

// ListDetailPriceRowDTO is a row DTO for PriceCard in listDetail.
// productBlueprintCategory / model kind に応じた model 情報を含める。
// - apparel: modelNumber / size / color / rgb
// - alcohol: modelNumber / volumeValue / volumeUnit
type ListDetailPriceRowDTO struct {
	ModelID string `json:"modelId"`
	// model kind
	// - apparel
	// - alcohol
	Kind string `json:"kind,omitempty"`
	// 型番
	ModelNumber string `json:"modelNumber,omitempty"`
	// displayOrder from productBlueprintPatch.ModelRefs
	// - 0 は未設定として nil を許容
	DisplayOrder *int `json:"displayOrder,omitempty"`
	// In list detail, stock is still shown
	Stock int `json:"stock"`
	// apparel 系表示用
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`
	// alcohol 系表示用
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`
	// Price nullable
	Price *int `json:"price,omitempty"`
}
