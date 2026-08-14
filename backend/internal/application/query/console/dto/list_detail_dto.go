// backend/internal/application/query/console/dto/list_detail_dto.go
package dto

// ListDetailDTO is the BFF response DTO for the Console list detail page.
// Frontend must treat this response as the source of truth and must not restore missing values with aliases or normalizers.
type ListDetailDTO struct {
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`

	Status      string `json:"status"`
	Title       string `json:"title"`
	Description string `json:"description"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`

	UpdatedBy     string `json:"updatedBy,omitempty"`
	UpdatedByName string `json:"updatedByName,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`

	ProductBlueprintID string `json:"productBlueprintId"`
	ProductBrandID     string `json:"productBrandId"`
	ProductBrandName   string `json:"productBrandName"`
	ProductName        string `json:"productName"`

	TokenBlueprintID string `json:"tokenBlueprintId"`
	TokenBrandID     string `json:"tokenBrandId"`
	TokenBrandName   string `json:"tokenBrandName"`
	TokenName        string `json:"tokenName"`

	PrimaryImageID string               `json:"primaryImageId,omitempty"`
	Images         []ListDetailImageDTO `json:"images"`

	PriceRows []ListDetailPriceRowDTO `json:"priceRows"`
}

// ListDetailImageDTO is the image information required by the list detail page.
// id is the list image document ID and primaryImageId points to one of these IDs.
type ListDetailImageDTO struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	DisplayOrder int    `json:"displayOrder"`
}

// ListDetailPriceRowDTO is a PriceCard row.
// Model information is resolved by the backend BFF.
// apparel: modelNumber / size / color / rgb
// alcohol: modelNumber / volumeValue / volumeUnit
type ListDetailPriceRowDTO struct {
	ModelID      string `json:"modelId"`
	Kind         string `json:"kind"`
	ModelNumber  string `json:"modelNumber"`
	DisplayOrder *int   `json:"displayOrder,omitempty"`
	Stock        int    `json:"stock"`

	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`

	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	Price *int `json:"price,omitempty"`
}
