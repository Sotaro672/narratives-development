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
// ============================================================

type InventoryManagementRowDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
	TokenBlueprintID   string `json:"tokenBlueprintId"` // ✅ 必須
	TokenName          string `json:"tokenName"`
	ModelNumber        string `json:"modelNumber"`
	Stock              int    `json:"stock"`
}

// ============================================================
// ✅ NEW: /inventory/ids response
// ============================================================

type InventoryIDsByProductAndTokenDTO struct {
	ProductBlueprintID string   `json:"productBlueprintId"`
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	InventoryIDs       []string `json:"inventoryIds"`
}
