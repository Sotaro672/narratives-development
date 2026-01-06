// backend/internal/application/query/dto/inventory_detail_dto.go
package dto

import (
	productbpdom "narratives/internal/domain/productBlueprint"
	tokenbpdom "narratives/internal/domain/tokenBlueprint"
)

// 画面（detail）向けの最小 DTO（まずは count を通す）
type InventoryDetailRowDTO struct {
	ModelID     string `json:"modelId"`
	ModelNumber string `json:"modelNumber"`

	// まずは count を stock に入れる
	Stock int `json:"stock"`

	// あとで拡張（取れない間は "-" / "" / null でOK）
	Size  string `json:"size"`
	Color string `json:"color"`
	RGB   *int   `json:"rgb,omitempty"`
}

type InventoryDetailDTO struct {
	InventoryID string `json:"inventoryId"`

	TokenBlueprintID   string `json:"tokenBlueprintId"`
	ProductBlueprintID string `json:"productBlueprintId"`

	// ✅ ProductBlueprintCard へ渡すための Patch
	// nil の場合は返さない（omitempty）
	ProductBlueprintPatch *productbpdom.Patch `json:"productBlueprintPatch,omitempty"`

	// ✅ NEW: TokenBlueprintCard へ渡すための Patch
	// nil の場合は返さない（omitempty）
	TokenBlueprintPatch *tokenbpdom.Patch `json:"tokenBlueprintPatch,omitempty"`

	Rows       []InventoryDetailRowDTO `json:"rows"`
	TotalStock int                     `json:"totalStock"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}
