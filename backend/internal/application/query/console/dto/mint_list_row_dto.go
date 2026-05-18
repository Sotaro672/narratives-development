// backend/internal/application/query/console/dto/mint_list_row_dto.go
package dto

type MintListRowDTO struct {
	InspectionID   string  `json:"inspectionId"`
	MintID         string  `json:"mintId"`
	TokenBlueprint string  `json:"tokenBlueprintId"`
	TokenName      string  `json:"tokenName"`
	CreatedByName  string  `json:"createdByName"`
	MintedAt       *string `json:"mintedAt"`
}
