// backend/internal/platform/di/console/adapter_query_list.go
package console

import (
	"context"
	"errors"

	productbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

//
// ========================================
// Query / List 用アダプタ
// ========================================
//

// pbQueryRepoAdapter adapts ProductBlueprintRepositoryFS to query.ProductBlueprintQueryRepo.
type pbQueryRepoAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) // value 戻り
	}
}

func (a *pbQueryRepoAdapter) ListIDsByCompany(ctx context.Context, companyID string) ([]string, error) {
	return a.repo.ListIDsByCompany(ctx, companyID)
}

func (a *pbQueryRepoAdapter) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return a.repo.GetByID(ctx, id)
}

// pbIDsByCompanyAdapter adapts ProductBlueprintRepositoryFS to query.productBlueprintIDsByCompanyReader
type pbIDsByCompanyAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *pbIDsByCompanyAdapter) ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error) {
	return a.repo.ListIDsByCompany(ctx, companyID)
}

// pbPatchByIDAdapter adapts ProductBlueprintRepositoryFS to query.productBlueprintPatchReader
type pbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func (a *pbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	return a.repo.GetPatchByID(ctx, id)
}

// tbGetterAdapter adapts TokenBlueprintRepositoryFS (pointer return) to query.TokenBlueprintGetter (value return).
type tbGetterAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
	}
}

func (a *tbGetterAdapter) GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error) {
	if a == nil || a.repo == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprint getter adapter is nil")
	}
	tb, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	if tb == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprint not found")
	}
	return *tb, nil
}
