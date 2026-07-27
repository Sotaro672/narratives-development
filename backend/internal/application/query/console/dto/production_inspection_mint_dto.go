// backend/internal/application/query/console/dto/production_inspection_mint_dto.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

type ProductionInspectionMintDTO struct {
	ID           string `json:"id"`
	ProductionID string `json:"productionId"`

	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`
	TokenName        string `json:"tokenName,omitempty"`
	ProductName      string `json:"productName,omitempty"`

	MintQuantity       int `json:"mintQuantity"`
	ProductionQuantity int `json:"productionQuantity"`

	InspectionStatus string `json:"inspectionStatus,omitempty"`

	// mintsドキュメントを作成したmemberId。
	CreatedBy string `json:"createdBy,omitempty"`

	// createdByに対応する表示名。
	CreatedByName string `json:"createdByName,omitempty"`

	// Mint申請ボタンを押したmemberId。
	RequestedBy string `json:"requestedBy,omitempty"`

	// requestedByに対応する表示名。
	RequestedByName string `json:"requestedByName,omitempty"`

	MintedAt *time.Time `json:"mintedAt,omitempty"`

	Inspection *inspectiondom.InspectionBatch `json:"inspection,omitempty"`
	Mint       *mintdom.Mint                  `json:"mint,omitempty"`
}
