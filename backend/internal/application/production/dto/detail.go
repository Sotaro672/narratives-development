// backend/internal/application/production/dto/detail.go
package dto

import "time"

// ProductionModelRowDTO
// - ProductionQuantityCard の rows 相当
// - モデルごとの数量内訳を 1 行ずつ表現する DTO
type ProductionModelRowDTO struct {
	ModelID      string `json:"modelId"`                // model variation の ID（Firestore docId）
	ModelNumber  string `json:"modelNumber"`            // 表示用型番
	Size         string `json:"size"`                   // サイズ（例: "M", "L"）
	Color        string `json:"color"`                  // カラー名
	RGB          *int   `json:"rgb,omitempty"`          // ✅ カラーRGB（0xRRGGBB）。ModelVariation DTO と統一
	DisplayOrder int    `json:"displayOrder,omitempty"` // ✅ 採番順（PBのmodelRefs.displayOrder等）。必要な場合のみ利用
	Quantity     int    `json:"quantity"`               // 生産数量
}

// ProductionDetailDTO
// - 生産詳細ページ（ProductionDetail.tsx）向けの ViewModel
// - production 本体 + 名前解決済みの文字列 + モデル内訳をまとめて返す想定
type ProductionDetailDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`

	// Brand 関連（NameResolver で解決）
	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`

	// 担当者（assignee）関連（NameResolver で memberId → 氏名に解決）
	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	// ステータス
	Status string `json:"status"` // "manufacturing" / "printed" / "inspected" など

	// モデル別数量内訳
	Models        []ProductionModelRowDTO `json:"models"`
	TotalQuantity int                     `json:"totalQuantity"`

	// 印刷・作成・更新メタ情報（画面側で日付フォーマットして使用）
	PrintedAt *time.Time `json:"printedAt,omitempty"`

	CreatedByID   *string   `json:"createdById,omitempty"`
	CreatedByName string    `json:"createdByName"`
	CreatedAt     time.Time `json:"createdAt"`

	UpdatedByID   *string    `json:"updatedById,omitempty"`
	UpdatedByName string     `json:"updatedByName"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}
