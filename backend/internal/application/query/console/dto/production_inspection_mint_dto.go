// backend/internal/application/query/dto/production_inspection_mint_dto.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

type ProductionInspectionMintDTO struct {
	ID                 string                         `json:"id"`
	ProductionID       string                         `json:"productionId"`
	TokenBlueprintID   string                         `json:"tokenBlueprintId,omitempty"`
	TokenName          string                         `json:"tokenName,omitempty"`
	ProductName        string                         `json:"productName,omitempty"`
	MintQuantity       int                            `json:"mintQuantity"`
	ProductionQuantity int                            `json:"productionQuantity"`
	InspectionStatus   string                         `json:"inspectionStatus,omitempty"`
	RequestedBy        string                         `json:"requestedBy,omitempty"`
	CreatedByName      string                         `json:"createdByName,omitempty"`
	MintedAt           *time.Time                     `json:"mintedAt,omitempty"`
	Inspection         *inspectiondom.InspectionBatch `json:"inspection,omitempty"`
	Mint               *mintdom.Mint                  `json:"mint,omitempty"`
}
