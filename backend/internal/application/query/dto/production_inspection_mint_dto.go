// backend/internal/application/query/dto/production_inspection_mint_dto.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ProductionInspectionMintDTO binds productionId (=docId) to inspection/mint docs
// that share the same docId, and also exposes mint-request management fields.
//
// requestedBy = mint.CreatedBy
// createdByName = display name resolved from requestedBy (frontend currently uses this field)
type ProductionInspectionMintDTO struct {
	// For UI row key / navigation (mintRequest側では id を使うことが多いので持たせる)
	ID string `json:"id"`

	// Same as ID (explicit)
	ProductionID string `json:"productionId"`

	// ---- mintRequest list fields (frontend expects these) ----

	// トークン設計名（TokenBlueprint 名など）
	TokenName string `json:"tokenName,omitempty"`

	// プロダクト名（ProductBlueprint 名など）
	ProductName string `json:"productName,omitempty"`

	// Mint 数量（例: inspection.totalPassed）
	MintQuantity int `json:"mintQuantity"`

	// 生産量（例: production.quantity 合計）
	ProductionQuantity int `json:"productionQuantity"`

	// 検査ステータス（"inspecting" | "completed" | "notYet" など）
	InspectionStatus string `json:"inspectionStatus,omitempty"`

	// ★ requestedBy = mint.CreatedBy
	RequestedBy string `json:"requestedBy,omitempty"`

	// フロントの requesterFilter が参照している名前フィールド
	CreatedByName string `json:"createdByName,omitempty"`

	// Mint 実行日時（mint.MintedAt 等）
	MintedAt *time.Time `json:"mintedAt,omitempty"`

	// ---- raw docs (optional but useful for debugging) ----
	Inspection *inspectiondom.InspectionBatch `json:"inspection,omitempty"`
	Mint       *mintdom.Mint                  `json:"mint,omitempty"`
}
