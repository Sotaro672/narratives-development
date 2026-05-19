// backend/internal/application/query/console/dto/mint_token_blueprint.go
package dto

type ListTokenBlueprintsForMintInput struct {
	BrandID string
	Page    int
	PerPage int
}

type TokenBlueprintPatchDTO struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description"`
	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`
	IconURL     string `json:"iconUrl,omitempty"`
}
