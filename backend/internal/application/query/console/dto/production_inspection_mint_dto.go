// backend/internal/application/query/console/dto/production_inspection_mint_dto.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

type ProductionInspectionMintDTO struct {
	ID           string `json:"id,omitempty"`
	ProductionID string `json:"productionId"`

	// ManagementPage / DetailPage 用: Firestore mints と同じく JSON では minted:boolean として返す
	Minted bool `json:"minted"`

	// Detail 側で patch を取得するための ID
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	MintQuantity       int `json:"mintQuantity"`
	ProductionQuantity int `json:"productionQuantity"`

	RequestedByName string `json:"requestedByName,omitempty"`

	MintedAt  *time.Time `json:"mintedAt,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`

	InspectionStatus string `json:"inspectionStatus,omitempty"`

	// 互換用。ManagementPage には返さない。
	Inspection *inspectiondom.InspectionBatch `json:"-"`
	Mint       *mintdom.Mint                  `json:"-"`
}
