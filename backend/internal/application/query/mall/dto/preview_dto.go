// backend\internal\application\query\mall\dto\preview_dto.go
package dto

type PreviewDTO struct {
	AvatarID string `json:"avatarId"`
	ItemKey  string `json:"itemKey"`

	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`
	Qty         int    `json:"qty,omitempty"`

	// list
	Title     string `json:"title,omitempty"`
	ListImage string `json:"listImage,omitempty"`
	Price     *int   `json:"price,omitempty"`

	// ids
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// product
	ProductName        string `json:"productName,omitempty"`
	ProductBrandID     string `json:"productBrandId,omitempty"`
	ProductCompanyID   string `json:"productCompanyId,omitempty"`
	ProductBrandName   string `json:"productBrandName,omitempty"`
	ProductCompanyName string `json:"productCompanyName,omitempty"`

	// token
	TokenName   string `json:"tokenName,omitempty"`
	BrandID     string `json:"brandId,omitempty"`
	CompanyID   string `json:"companyId,omitempty"`
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`

	// model
	ModelNumber string `json:"modelNumber,omitempty"`
	Size        string `json:"size,omitempty"`
	Color       string `json:"color,omitempty"`
	RGB         *int   `json:"rgb,omitempty"`
}
