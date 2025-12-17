// backend\internal\application\query\dto\inventory_dto.go
package dto

import (
	"time"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs for InventoryDetail page (read-only)
// ============================================================

type InventoryDetailDTO struct {
	// 互換用: 単一取得では inventoryId、pbId 指定の詳細では pbId が入る想定
	InventoryID string `json:"inventoryId"`

	// ✅ NEW: 詳細で対象になる inventoryId 群（pbId + tokenBlueprintId で絞り込んだ結果）
	// 互換のため optional にする
	InventoryIDs []string `json:"inventoryIds,omitempty"`

	// tokenBlueprintId を指定している場合は必ず入る（pbId のみの場合は空のことがある）
	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`

	// pbId query の場合に必ず入る
	ProductBlueprintID string `json:"productBlueprintId"`

	// 単一 inventoryId 取得の場合に入る（pbId query の場合は空のことがある）
	ModelID string `json:"modelId,omitempty"`

	ProductBlueprintPatch InventoryProductBlueprintPatchDTO   `json:"productBlueprintPatch"`
	TokenBlueprint        InventoryTokenBlueprintSummaryDTO   `json:"tokenBlueprint"`
	ProductBlueprint      InventoryProductBlueprintSummaryDTO `json:"productBlueprint"`

	Rows       []InventoryRowDTO `json:"rows"`
	TotalStock int               `json:"totalStock"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// ★ dto パッケージ内で TokenBlueprintSummaryDTO が既に存在するため、Inventory 専用名にする
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

// InventoryCard 用（フロント側の命名に合わせる）
//   - token 列を左に追加
//   - modelCode -> modelNumber
//   - colorName -> color
//   - colorCode -> rgb（数値 or 変換可能な文字列を想定）
//     ※ JSON では数値(int)で返す（rgbIntToHex が使える）
type InventoryRowDTO struct {
	// ✅ 追加: 集計キーに必要（pbId + tokenBlueprintId で 1行にまとめるため）
	TokenBlueprintID string `json:"tokenBlueprintId"`

	Token       string `json:"token"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	RGB         *int   `json:"rgb,omitempty"`
	Stock       int    `json:"stock"`
}

// ============================================================
// DTOs (Inventory Management List)
// - 列: プロダクト名 / トークン名 / 型番 / 在庫数
// ============================================================

type InventoryManagementRowDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
	TokenName          string `json:"tokenName"`
	ModelNumber        string `json:"modelNumber"`
	Stock              int    `json:"stock"`
}
