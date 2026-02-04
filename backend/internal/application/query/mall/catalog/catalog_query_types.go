// backend/internal/application/query/mall/catalog/catalog_query_types.go
package catalogQuery

import (
	"context"

	appresolver "narratives/internal/application/resolver"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Ports (minimal contracts for this query)
// ============================================================

type InventoryRepository interface {
	GetByID(ctx context.Context, id string) (invdom.Mint, error)
}

type ProductBlueprintRepository interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type TokenBlueprintPatchRepository interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

// ✅ ListImage repository (read-only minimal for catalog)
// catalog は listId から「全画像（displayOrder含む）」を返したい
type ListImageRepository interface {
	// listId 配下の画像一覧（displayOrder を含む前提）
	FindByListID(ctx context.Context, listID string) ([]listimgdom.ListImage, error)
}

// ============================================================
// Query
// ============================================================

type CatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository
	TokenRepo     TokenBlueprintPatchRepository

	ModelRepo modeldom.RepositoryPort

	// ✅ list images (optional)
	ListImageRepo ListImageRepository

	NameResolver *appresolver.NameResolver
}

// ============================================================
// Constructor Options (single entrypoint)
// ============================================================

type CatalogQueryOption func(*CatalogQuery)

func WithTokenBlueprintPatchRepo(tokenRepo TokenBlueprintPatchRepository) CatalogQueryOption {
	return func(q *CatalogQuery) {
		q.TokenRepo = tokenRepo
	}
}

func WithNameResolver(nameResolver *appresolver.NameResolver) CatalogQueryOption {
	return func(q *CatalogQuery) {
		q.NameResolver = nameResolver
	}
}

// ✅ optional listImage repo
func WithListImageRepo(repo ListImageRepository) CatalogQueryOption {
	return func(q *CatalogQuery) {
		q.ListImageRepo = repo
	}
}

// NewCatalogQuery is the ONLY wiring entrypoint.
// All dependencies must be routed through this constructor.
func NewCatalogQuery(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
	opts ...CatalogQueryOption,
) *CatalogQuery {
	q := &CatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     nil, // optional
		ModelRepo:     modelRepo,

		ListImageRepo: nil, // optional
		NameResolver:  nil, // optional
	}

	for _, opt := range opts {
		if opt != nil {
			opt(q)
		}
	}
	return q
}
