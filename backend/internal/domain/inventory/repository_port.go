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

	// ★ NEW
	ListByModelID(ctx context.Context, modelID string) ([]Mint, error)
	ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]Mint, error)

	// atomic
	IncrementAccumulation(ctx context.Context, id string, delta int) (Mint, error)

	// ★ NEW (atomic upsert)
	UpsertByModelAndToken(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (Mint, error)
}
