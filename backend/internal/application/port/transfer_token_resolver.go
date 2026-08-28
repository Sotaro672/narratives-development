// backend/internal/application/port/transfer_token_resolver.go
package port

import "context"

// TokenResolver resolves the token required for a product transfer.
type TokenResolver interface {
	ResolveTokenByProductID(
		ctx context.Context,
		productID string,
	) (TokenForTransfer, error)
}

// TokenForTransfer is the minimal token state required by the transfer flow.
type TokenForTransfer struct {
	ProductID        string
	BrandID          string
	AssetID          string
	TokenBlueprintID string
}
