// backend/internal/application/query/mall/dto/catalog_dto.go
package dto

import (
	ldom "narratives/internal/domain/list"
)

// ============================================================
// DTOs (for catalog.dart)
// ============================================================

type CatalogDTO struct {
	List CatalogListDTO `json:"list"`

	// listId 配下の画像一覧（displayOrder 付き）
	// - 空配列は返さない（omitempty）
	// - 取得失敗しても画面は壊さない想定なので Error は best-effort
	ListImages      []CatalogListImageDTO `json:"listImages,omitempty"`
	ListImagesError string                `json:"listImagesError,omitempty"`

	Inventory      *CatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string               `json:"inventoryError,omitempty"`

	ProductBlueprint      *CatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                      `json:"productBlueprintError,omitempty"`

	// catalog 専用の tokenBlueprint DTO（Patch を直接返さない）
	TokenBlueprint      *CatalogTokenBlueprintDTO `json:"tokenBlueprint,omitempty"`
	TokenBlueprintError string                    `json:"tokenBlueprintError,omitempty"`

	ModelVariations      []CatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                     `json:"modelVariationsError,omitempty"`

	// productBlueprintReview（商品単位の口コミ集計）
	ProductReviewSummary      *CatalogProductReviewSummaryDTO `json:"productReviewSummary,omitempty"`
	ProductReviewSummaryError string                          `json:"productReviewSummaryError,omitempty"`
}

type CatalogListDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

// ============================================================
// TokenBlueprint DTO (catalog response)
// 要件：minted/brandId/companyId/metadataUri は不要
// 追加：companyName/tokenIcon/description
// - description/tokenIcon は空でもキーを返すため omitempty を付けない
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
// ListImage DTOs (absolute schema)
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

	ItemType string  `json:"itemType"`
	Fit      string  `json:"fit"`
	Material string  `json:"material"`
	Weight   float64 `json:"weight,omitempty"`
	Printed  bool    `json:"printed"`

	QualityAssurance []string `json:"qualityAssurance"`
	ProductIDTagType string   `json:"productIdTagType"`

	ModelRefs []CatalogProductBlueprintModelRefDTO `json:"modelRefs,omitempty"`
}

// ============================================================
// ModelVariation DTO
// - apparel / alcohol の両方に対応する
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
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	StockKeys int `json:"stockKeys,omitempty"`
}

// ============================================================
// ProductBlueprintReview Summary DTO
// - 平均評価
// - 件数
// - 星別分布（5..1）
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
