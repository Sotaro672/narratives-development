// backend/internal/application/query/mall/catalog/catalog_query_types.go
package catalogQuery

import (
	"context"

	appresolver "narratives/internal/application/resolver"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
	productBlueprintReview "narratives/internal/domain/productBlueprintReview"
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

// ProductBlueprintReview repository (read-only minimal for catalog)
// CatalogQuery では summary のみ利用するため、最小契約にする
type ProductBlueprintReviewRepository interface {
	GetProductSummary(
		ctx context.Context,
		productBlueprintID string,
		status productBlueprintReview.ReviewStatus,
	) (productBlueprintReview.ProductReviewSummary, error)
}

// ListImage repository (read-only minimal for catalog)
//
// Firebase Storage 移行後:
// - domain/listImage は削除済み
// - ListImage は domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
// - backend は GCS bucket / public URL を組み立てない
type ListImageRepository interface {
	// listId 配下の画像一覧（displayOrder を含む前提）
	FindByListID(ctx context.Context, listID string) ([]ldom.ListImage, error)
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

	// product blueprint reviews
	ProductBlueprintReviewRepo ProductBlueprintReviewRepository

	// list images (optional)
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

// optional productBlueprintReview repo
func WithProductBlueprintReviewRepo(repo ProductBlueprintReviewRepository) CatalogQueryOption {
	return func(q *CatalogQuery) {
		q.ProductBlueprintReviewRepo = repo
	}
}

// optional listImage repo
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

		ProductBlueprintReviewRepo: nil, // optional

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
