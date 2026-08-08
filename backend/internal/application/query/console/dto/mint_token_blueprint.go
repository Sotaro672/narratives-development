// backend/internal/application/query/console/dto/mint_token_blueprint.go
package dto

type ListTokenBlueprintsForMintInput struct {
	BrandID string
	Page    int
	PerPage int
}

type TokenBlueprintForMintDTO struct {
	ID        string `json:"id"`
	TokenName string `json:"tokenName"`
	Symbol    string `json:"symbol"`

	BrandID string `json:"brandId,omitempty"`

	Description string `json:"description,omitempty"`
	Minted      bool   `json:"minted"`

	IconURL string `json:"iconUrl,omitempty"`
}
