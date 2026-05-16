// backend/internal/application/query/mall/dto/cart_dto.go
package dto

// CartDTO is the response shape for Mall cart list UI.
// NOTE: cart_query returns fields required for cart screen.
type CartDTO struct {
	AvatarID string                 `json:"avatarId"`
	Items    map[string]CartItemDTO `json:"items"`

	CreatedAt *string `json:"createdAt,omitempty"`
	UpdatedAt *string `json:"updatedAt,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

type CartItemDTO struct {
	// identifiers
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`

	// list fields
	Title     string `json:"title,omitempty"`
	ListImage string `json:"listImage,omitempty"` // List.ImageID or resolved image URL
	Price     *int   `json:"price,omitempty"`     // JPY

	// product fields
	ProductName string `json:"productName,omitempty"`

	// category fields
	ProductBlueprintCategoryID     string   `json:"productBlueprintCategoryId,omitempty"`
	ProductBlueprintCategoryCode   string   `json:"productBlueprintCategoryCode,omitempty"`
	ProductBlueprintCategoryKind   string   `json:"productBlueprintCategoryKind,omitempty"`
	ProductBlueprintCategoryNameEn string   `json:"productBlueprintCategoryNameEn,omitempty"`
	ProductBlueprintCategoryNameJa string   `json:"productBlueprintCategoryNameJa,omitempty"`
	ProductBlueprintCategoryPath   []string `json:"productBlueprintCategoryPath,omitempty"`

	// model common fields
	//
	// ModelKind:
	// - apparel
	// - alcohol
	// - unknown
	ModelKind   string `json:"modelKind,omitempty"`
	ModelNumber string `json:"modelNumber,omitempty"`

	// ModelLabel is a display-ready label for cart UI.
	//
	// examples:
	// - apparel: "M / Black"
	// - alcohol: "s / 720ml"
	ModelLabel string `json:"modelLabel,omitempty"`

	// apparel model fields
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`

	// alcohol model fields
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	Qty int `json:"qty"`
}
