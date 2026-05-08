// backend/internal/application/mint/dto/detail.go
package dto

import inspectiondom "narratives/internal/domain/inspection"

// MintDetailDTO は MintRequestDetail（詳細）向け DTO です。
// - scheduledBurnDate / brandId / brandName / tokenName を画面へ返す
// - 日付はフロントで扱いやすいように string で返す（nil 可）
type MintDetailDTO struct {
	BrandID           string  `json:"brandId"`
	BrandName         string  `json:"brandName"`
	TokenName         string  `json:"tokenName"`
	ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"` // "YYYY-MM-DD" を想定
}

// ============================================================
// 画面向け DTO（inspections 一覧 / 詳細で共用）
// ============================================================

// MintModelMeta は、モデル情報をフロントに渡すためのメタ情報
type MintModelMeta struct {
	Size      string `json:"size"`
	ColorName string `json:"colorName"`
	RGB       int    `json:"rgb"`
}

// MintInspectionView は Mint 管理画面向けの Inspection 表現。
// 元の InspectionBatch に加えて、productBlueprintId / productName、
// そして modelId → size/color/rgb のマップを付与して返す。
type MintInspectionView struct {
	inspectiondom.InspectionBatch

	// Production → ProductBlueprint の join 結果
	ProductBlueprintID string `json:"productBlueprintId"`

	// ★ 名前解決は presenter が担当（usecase では埋めない）
	ProductName string `json:"productName"`

	// モデル情報テーブル: modelId → { size, colorName, rgb }
	ModelMeta map[string]MintModelMeta `json:"modelMeta"`
}
