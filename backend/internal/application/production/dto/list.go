// internal/application/production/dto/list.go
package dto

import productiondom "narratives/internal/domain/production"

// ProductionListItemDTO ... ProductionManagement 一覧行用
// - productiondom.Production をそのまま埋め込み
// - 一覧表示用の派生フィールドを追加
type ProductionListItemDTO struct {
	// 元エンティティ（id / productBlueprintId / brandId / assigneeId / status /
	// models / printedAt / createdAt など）
	productiondom.Production

	// 一覧専用の派生フィールド
	TotalQuantity  int    `json:"totalQuantity"`
	PrintedAtLabel string `json:"printedAtLabel"`
	CreatedAtLabel string `json:"createdAtLabel"`

	// 名前解決済みフィールド
	ProductName  string `json:"productName,omitempty"`
	BrandName    string `json:"brandName,omitempty"`
	AssigneeName string `json:"assigneeName,omitempty"`
}
