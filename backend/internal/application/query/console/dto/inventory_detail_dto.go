// backend/internal/application/query/console/dto/inventory_detail_dto.go
package dto

// InventoryProductBlueprintCategoryDTO は
// Inventory Detail 画面向けの
// ProductBlueprintCategory snapshot。
//
// HTTP response では lowerCamelCase を正とする。
type InventoryProductBlueprintCategoryDTO struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

// InventoryProductIDTagDTO は
// Inventory Detail 画面向けの ProductIDTag。
type InventoryProductIDTagDTO struct {
	Type string `json:"type"`
}

// InventoryProductBlueprintModelRefDTO は
// Inventory Detail 画面向けの model reference。
type InventoryProductBlueprintModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

// InventoryProductBlueprintPatchDTO は
// Inventory Detail 画面向けの
// ProductBlueprint read model。
//
// domain の Patch を HTTP response へ直接公開せず、
// この DTO の lowerCamelCase JSON contract を正とする。
type InventoryProductBlueprintPatchDTO struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`
	CompanyID string `json:"companyId"`

	ProductBlueprintCategory InventoryProductBlueprintCategoryDTO `json:"productBlueprintCategory"`

	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	ProductIDTag InventoryProductIDTagDTO `json:"productIdTag"`

	AssigneeID string `json:"assigneeId"`

	ModelRefs []InventoryProductBlueprintModelRefDTO `json:"modelRefs"`
}

// InventoryTokenBlueprintPatchDTO は
// Inventory Detail 画面向けの
// TokenBlueprint read model。
//
// TokenBlueprintCard で必要な表示情報を
// Backend 側で完成させて返す。
type InventoryTokenBlueprintPatchDTO struct {
	ID        string `json:"id"`
	TokenName string `json:"tokenName"`
	Symbol    string `json:"symbol"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`
	CompanyID string `json:"companyId"`

	Description string `json:"description"`

	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`

	IconURL string `json:"iconUrl,omitempty"`
}

// InventoryShippingAddressDTO は
// Inventory Detail 画面向けの
// 在庫保管場所 read model。
//
// shippingAddress domain entity を
// frontend へ直接公開せず、
// Inventory Detail 画面で必要な住所情報だけを返す。
//
// country は JP 固定のため、
// Inventory Detail 画面では返さない。
type InventoryShippingAddressDTO struct {
	ID string `json:"id"`

	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
}

// InventoryDetailRowDTO は
// Inventory Detail 画面向けの在庫行 DTO。
//
// GET /inventory/{inventoryId} の rows として返す。
//
// frontend 側では
// /models/by-blueprint/{productBlueprintId}/variations
// を追加取得せず、この rows を正とする。
type InventoryDetailRowDTO struct {
	ModelID     string `json:"modelId"`
	Kind        string `json:"kind,omitempty"`
	ModelNumber string `json:"modelNumber"`

	Stock int `json:"stock"`

	// apparel 系 model 用
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`

	// alcohol 系 model 用
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`
}

// InventoryDetailDTO は
// GET /inventory/{inventoryId} の
// Inventory Detail 画面専用 BFF response。
//
// frontend 側で別 API の response を merge せず、
// この DTO を唯一の正とする。
type InventoryDetailDTO struct {
	InventoryID string `json:"inventoryId"`

	TokenBlueprintID   string `json:"tokenBlueprintId"`
	ProductBlueprintID string `json:"productBlueprintId"`

	ProductBlueprintPatch *InventoryProductBlueprintPatchDTO `json:"productBlueprintPatch,omitempty"`
	TokenBlueprintPatch   *InventoryTokenBlueprintPatchDTO   `json:"tokenBlueprintPatch,omitempty"`

	ShippingAddressID string `json:"shippingAddressId,omitempty"`

	ShippingAddress *InventoryShippingAddressDTO `json:"shippingAddress,omitempty"`

	ShippingAddressOptions []InventoryShippingAddressDTO `json:"shippingAddressOptions"`

	Rows       []InventoryDetailRowDTO `json:"rows"`
	TotalStock int                     `json:"totalStock"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}
