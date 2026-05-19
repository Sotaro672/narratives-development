// backend/internal/application/query/console/dto/mint_token_blueprint.go
package dto

type ListTokenBlueprintsForMintInput struct {
	BrandID string
	Page    int
	PerPage int
}

type TokenBlueprintForMintDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}
