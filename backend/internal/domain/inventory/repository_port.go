package inventory

import "context"

// RepositoryPort is output port for inventories persistence.
type RepositoryPort interface {
	Create(ctx context.Context, m Mint) (Mint, error)
	GetByID(ctx context.Context, id string) (Mint, error)
	Update(ctx context.Context, m Mint) (Mint, error)
	Delete(ctx context.Context, id string) error

	// Queries
	ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Mint, error)
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]Mint, error)

	// stock の modelIds 補助フィールドで検索する想定
	ListByModelID(ctx context.Context, modelID string) ([]Mint, error)
	ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]Mint, error)

	// atomic upsert
	// - docId = productBlueprintId__tokenBlueprintId
	// - stock.<modelId> = { products: map[productId]bool, accumulation: len(products) } を置換
	UpsertByProductBlueprintAndToken(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (Mint, error)
}
