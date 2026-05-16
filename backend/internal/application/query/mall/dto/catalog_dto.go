// backend/internal/application/query/mall/dto/catalog_dto.go
package dto

import (
	ldom "narratives/internal/domain/list"
)

// ============================================================
// DTOs for catalog response
// ============================================================

type CatalogDTO struct {
	List CatalogListDTO `json:"list"`

	// listId 配下の画像一覧（displayOrder 付き）
	// 取得失敗しても画面は壊さない想定なので Error は best-effort
	ListImages      []CatalogListImageDTO `json:"listImages,omitempty"`
	ListImagesError string                `json:"listImagesError,omitempty"`

	Inventory      *CatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string               `json:"inventoryError,omitempty"`

	ProductBlueprint      *CatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                      `json:"productBlueprintError,omitempty"`

	TokenBlueprint      *CatalogTokenBlueprintDTO `json:"tokenBlueprint,omitempty"`
	TokenBlueprintError string                    `json:"tokenBlueprintError,omitempty"`

	ModelVariations      []CatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                     `json:"modelVariationsError,omitempty"`

	ProductReviewSummary      *CatalogProductReviewSummaryDTO `json:"productReviewSummary,omitempty"`
	ProductReviewSummaryError string                          `json:"productReviewSummaryError,omitempty"`
}

type CatalogListDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"`
	Prices      []ldom.ListPriceRow `json:"prices"`

	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

// ============================================================
// TokenBlueprint DTO
// ============================================================

type CatalogTokenBlueprintDTO struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	Description string `json:"description"`
	TokenIcon   string `json:"tokenIcon"`
}

// ============================================================
// ListImage DTOs
// ============================================================

type CatalogListImageDTO struct {
	ID         string `json:"id"`
	ListID     string `json:"listId"`
	URL        string `json:"url"`
	ObjectPath string `json:"objectPath"`
	FileName   string `json:"fileName,omitempty"`

	DisplayOrder int   `json:"displayOrder"`
	Size         int64 `json:"size,omitempty"`
}

// ============================================================
// Inventory DTOs
// ============================================================

type CatalogInventoryModelStockDTO struct {
	Accumulation  int `json:"accumulation"`
	ReservedCount int `json:"reservedCount"`
}

type CatalogInventoryDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	ModelIDs []string                                 `json:"modelIds,omitempty"`
	Stock    map[string]CatalogInventoryModelStockDTO `json:"stock,omitempty"`
}

// ============================================================
// ProductBlueprint DTO
// ============================================================

type CatalogProductBlueprintModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

type CatalogProductBlueprintDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`

	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`

	Printed          bool   `json:"printed"`
	ProductIDTagType string `json:"productIdTagType"`

	ProductBlueprintCategoryID     string   `json:"productBlueprintCategoryId,omitempty"`
	ProductBlueprintCategoryCode   string   `json:"productBlueprintCategoryCode,omitempty"`
	ProductBlueprintCategoryKind   string   `json:"productBlueprintCategoryKind,omitempty"`
	ProductBlueprintCategoryNameEn string   `json:"productBlueprintCategoryNameEn,omitempty"`
	ProductBlueprintCategoryNameJa string   `json:"productBlueprintCategoryNameJa,omitempty"`
	ProductBlueprintCategoryPath   []string `json:"productBlueprintCategoryPath,omitempty"`

	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	ModelRefs []CatalogProductBlueprintModelRefDTO `json:"modelRefs,omitempty"`
}

// ============================================================
// ModelVariation DTO
// ============================================================

type CatalogModelVariationDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`

	// model kind
	// - apparel
	// - alcohol
	Kind string `json:"kind,omitempty"`

	ModelNumber string `json:"modelNumber"`

	// apparel
	Size string `json:"size,omitempty"`

	ColorName string `json:"colorName,omitempty"`
	ColorRGB  int    `json:"colorRGB,omitempty"`

	Measurements map[string]int `json:"measurements,omitempty"`

	// alcohol
	VolumeValue *float64 `json:"volumeValue,omitempty"`
	VolumeUnit  string   `json:"volumeUnit,omitempty"`

	StockKeys int `json:"stockKeys,omitempty"`
}

// ============================================================
// ProductBlueprintReview Summary DTO
// ============================================================

type CatalogProductReviewSummaryDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	Status             string `json:"status"`

	TotalCount    int     `json:"totalCount"`
	AverageRating float64 `json:"averageRating"`

	Rating5Count int `json:"rating5Count"`
	Rating4Count int `json:"rating4Count"`
	Rating3Count int `json:"rating3Count"`
	Rating2Count int `json:"rating2Count"`
	Rating1Count int `json:"rating1Count"`
}
