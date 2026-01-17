// backend/internal/application/query/mall/dto/catalog_dto.go
package dto

import (
	ldom "narratives/internal/domain/list"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// DTOs (for catalog.dart)
// ============================================================

type CatalogDTO struct {
	List CatalogListDTO `json:"list"`

	Inventory      *CatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string               `json:"inventoryError,omitempty"`

	ProductBlueprint      *CatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                      `json:"productBlueprintError,omitempty"`

	// tokenBlueprint patch
	TokenBlueprint      *tbdom.Patch `json:"tokenBlueprint,omitempty"`
	TokenBlueprintError string       `json:"tokenBlueprintError,omitempty"`

	ModelVariations      []CatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                     `json:"modelVariationsError,omitempty"`
}

type CatalogListDTO struct {
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

// ============================================================
// Inventory DTOs (handler-aligned; NO legacy compatibility)
// - products is removed
// - availableStock is computed on frontend as (accumulation - reservedCount)
// ============================================================

type CatalogInventoryModelStockDTO struct {
	Accumulation  int `json:"accumulation"`
	ReservedCount int `json:"reservedCount"`
}

type CatalogInventoryDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	// keep: inventory handler also returns modelIds
	ModelIDs []string `json:"modelIds,omitempty"`

	// stock (key=modelId)
	Stock map[string]CatalogInventoryModelStockDTO `json:"stock,omitempty"`
}

// ============================================================
// ProductBlueprint DTO
// ============================================================

type CatalogProductBlueprintDTO struct {
	ID               string   `json:"id"`
	ProductName      string   `json:"productName"`
	BrandID          string   `json:"brandId"`
	CompanyID        string   `json:"companyId"`
	BrandName        string   `json:"brandName,omitempty"`
	CompanyName      string   `json:"companyName,omitempty"`
	ItemType         string   `json:"itemType"`
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight,omitempty"`
	Printed          bool     `json:"printed"`
	QualityAssurance []string `json:"qualityAssurance"`
	ProductIDTagType string   `json:"productIdTagType"`
}

// ============================================================
// ModelVariation DTO
// - products is removed (NO legacy compatibility)
// ============================================================

type CatalogModelVariationDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`
	ModelNumber        string `json:"modelNumber"`
	Size               string `json:"size"`

	ColorName string `json:"colorName"`
	ColorRGB  int    `json:"colorRGB"`

	// emptyでも {} を返す（catalog_query 側で非nil化する）
	Measurements map[string]int `json:"measurements"`

	// 在庫の「model種類数」（必要なら残す）
	StockKeys int `json:"stockKeys,omitempty"`
}
