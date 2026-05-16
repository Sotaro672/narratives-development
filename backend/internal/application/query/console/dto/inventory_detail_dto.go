// backend/internal/application/query/dto/inventory_detail_dto.go
package dto

import (
	productbpdom "narratives/internal/domain/productBlueprint"
	tokenbpdom "narratives/internal/domain/tokenBlueprint"
)

// InventoryDetailRowDTO は InventoryDetail 画面向けの在庫行 DTO。
// GET /inventory/{inventoryId} の rows として返す。
// frontend 側で /models/by-blueprint/{productBlueprintId}/variations を追加取得しなくてよいように、
// productBlueprintCategory.Kind に応じた model 情報をここへ含める。
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

type InventoryDetailDTO struct {
	InventoryID string `json:"inventoryId"`

	TokenBlueprintID   string `json:"tokenBlueprintId"`
	ProductBlueprintID string `json:"productBlueprintId"`

	// ProductBlueprintCard へ渡すための Patch
	// nil の場合は返さない
	ProductBlueprintPatch *productbpdom.Patch `json:"productBlueprintPatch,omitempty"`

	// TokenBlueprintCard へ渡すための Patch
	// nil の場合は返さない
	TokenBlueprintPatch *tokenbpdom.Patch `json:"tokenBlueprintPatch,omitempty"`

	Rows       []InventoryDetailRowDTO `json:"rows"`
	TotalStock int                     `json:"totalStock"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}
