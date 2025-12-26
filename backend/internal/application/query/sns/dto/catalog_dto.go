// backend/internal/application/query/sns/dto/catalog_dto.go
package dto

import (
	ldom "narratives/internal/domain/list"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// DTOs (for catalog.dart)
// ============================================================

type SNSCatalogDTO struct {
	List SNSCatalogListDTO `json:"list"`

	Inventory      *SNSCatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string                  `json:"inventoryError,omitempty"`

	ProductBlueprint      *SNSCatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                         `json:"productBlueprintError,omitempty"`

	// ✅ NEW: tokenBlueprint patch
	TokenBlueprint      *tbdom.Patch `json:"tokenBlueprint,omitempty"`
	TokenBlueprintError string       `json:"tokenBlueprintError,omitempty"`

	ModelVariations      []SNSCatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                        `json:"modelVariationsError,omitempty"`
}

type SNSCatalogListDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	// linkage (catalog.dart uses these)
	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

// ✅ inventory stock model value (same shape as SNS inventory response)
type SNSCatalogInventoryModelStockDTO struct {
	Products map[string]bool `json:"products,omitempty"`
}

type SNSCatalogInventoryDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	// (optional) inventory handler has this; keep it compatible
	ModelIDs []string `json:"modelIds,omitempty"`

	// ✅ NEW: stock を扱えるようにする（products を含める）
	// key=modelId
	Stock map[string]SNSCatalogInventoryModelStockDTO `json:"stock,omitempty"`
}

type SNSCatalogProductBlueprintDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`

	// ✅ NEW: 表示用（catalog_query.go 側で name_resolver などから埋める）
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`

	ItemType string `json:"itemType"`
	Fit      string `json:"fit"`
	Material string `json:"material"`

	// domain 側が float64 のため、0 の場合は omitempty で落ちる
	Weight float64 `json:"weight,omitempty"`

	Printed bool `json:"printed"`

	QualityAssurance []string `json:"qualityAssurance"`
	ProductIDTagType string   `json:"productIdTagType"`
}

type SNSCatalogModelVariationDTO struct {
	ID                 string             `json:"id"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	ModelNumber        string             `json:"modelNumber"`
	Size               string             `json:"size"`
	Color              SNSCatalogColorDTO `json:"color"`
	Measurements       map[string]int     `json:"measurements,omitempty"`

	// ✅ NEW: 在庫の「件数（key数）」だけ画面へ渡す
	StockKeys int `json:"stockKeys,omitempty"`
}

type SNSCatalogColorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"`
}
