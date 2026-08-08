// backend/internal/application/query/console/dto/list_detail_dto.go
package dto

// ListDetailDTO is a screen DTO for listDetail page.
//
// Current requirements (frontend/console/list):
// - From List: title, description, status, assigneeId, assigneeName, audit names, timestamps
// - From ProductBlueprint: productName, productBrandName
// - From TokenBlueprint: tokenName, tokenBrandName
// - Images: imageUrls
// - Price: priceRows
type ListDetailDTO struct {
	// status
	Status string `json:"status,omitempty"`

	// listing content
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// assignee
	AssigneeID   string `json:"assigneeId,omitempty"`
	AssigneeName string `json:"assigneeName,omitempty"`

	// audit
	CreatedByName string `json:"createdByName,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`

	// updater
	UpdatedByName string `json:"updatedByName,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`

	// product
	ProductBrandName string `json:"productBrandName,omitempty"`
	ProductName      string `json:"productName,omitempty"`

	// token
	TokenBrandName string `json:"tokenBrandName,omitempty"`
	TokenName      string `json:"tokenName,omitempty"`

	// images
	ImageURLs []string `json:"imageUrls,omitempty"`

	// price
	PriceRows []ListDetailPriceRowDTO `json:"priceRows,omitempty"`
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
