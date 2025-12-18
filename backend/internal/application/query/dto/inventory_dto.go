// backend/internal/application/query/dto/inventory_dto.go
package dto

import (
	"time"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs for InventoryDetail page (read-only)
// ============================================================

type InventoryDetailDTO struct {
	InventoryID string `json:"inventoryId"`

	InventoryIDs          []string                            `json:"inventoryIds,omitempty"`
	TokenBlueprintID      string                              `json:"tokenBlueprintId,omitempty"`
	ProductBlueprintID    string                              `json:"productBlueprintId"`
	ModelID               string                              `json:"modelId,omitempty"`
	ProductBlueprintPatch InventoryProductBlueprintPatchDTO   `json:"productBlueprintPatch"`
	TokenBlueprint        InventoryTokenBlueprintSummaryDTO   `json:"tokenBlueprint"`
	ProductBlueprint      InventoryProductBlueprintSummaryDTO `json:"productBlueprint"`
	Rows                  []InventoryRowDTO                   `json:"rows"`
	TotalStock            int                                 `json:"totalStock"`
	UpdatedAt             time.Time                           `json:"updatedAt"`
}

type InventoryTokenBlueprintSummaryDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

type InventoryProductBlueprintSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type InventoryProductBlueprintPatchDTO struct {
	ProductName      *string             `json:"productName,omitempty"`
	BrandID          *string             `json:"brandId,omitempty"`
	ItemType         *pbdom.ItemType     `json:"itemType,omitempty"`
	Fit              *string             `json:"fit,omitempty"`
	Material         *string             `json:"material,omitempty"`
	Weight           *float64            `json:"weight,omitempty"`
	QualityAssurance *[]string           `json:"qualityAssurance,omitempty"`
	ProductIdTag     *pbdom.ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID       *string             `json:"assigneeId,omitempty"`
}

type InventoryRowDTO struct {
	TokenBlueprintID string `json:"tokenBlueprintId"`
	Token            string `json:"token"`
	ModelNumber      string `json:"modelNumber"`
	Size             string `json:"size"`
	Color            string `json:"color"`
	RGB              *int   `json:"rgb,omitempty"`
	Stock            int    `json:"stock"`
}

// ============================================================
// DTOs (Inventory Management List)
// - 列: プロダクト名 / トークン名 / 型番 / 在庫数
// ✅ 方針A: detail遷移に tokenBlueprintId が必須なので追加
// ============================================================

type InventoryManagementRowDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`

	// ✅ これが無いとフロントが detail URL を作れず tbId="-" になる
	TokenBlueprintID string `json:"tokenBlueprintId"`

	TokenName   string `json:"tokenName"`
	ModelNumber string `json:"modelNumber"`
	Stock       int    `json:"stock"`
}
