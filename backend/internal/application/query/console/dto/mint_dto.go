// backend/internal/application/query/console/dto/mint_dto.go
package dto

type MintDTO struct {
	ID                 string   `json:"id"`
	InspectionID       string   `json:"inspectionId"`
	BrandID            string   `json:"brandId"`
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	TokenName          string   `json:"tokenName"`
	Products           []string `json:"products"`
	CreatedBy          string   `json:"createdBy"`
	CreatedByName      string   `json:"createdByName"`
	CreatedAt          *string  `json:"createdAt"`
	Minted             bool     `json:"minted"`
	MintedAt           *string  `json:"mintedAt"`
	ScheduledBurnDate  *string  `json:"scheduledBurnDate"`
	OnChainTxSignature string   `json:"onChainTxSignature,omitempty"`
}
