// backend\internal\application\production\query.go
package production

import (
	"context"
	"errors"
	"strings"

	dto "narratives/internal/application/production/dto"
	productiondom "narratives/internal/domain/production"
)

// ============================
// List delegation port
// ============================

// ProductionListQuery is the minimal read/query port that ProductionUsecase delegates list operations to.
// Implemented by query/console.CompanyProductionQueryService.
type ProductionListQuery interface {
	ListProductionsByCurrentCompany(ctx context.Context) ([]productiondom.Production, error)
	ListProductionsWithAssigneeName(ctx context.Context) ([]dto.ProductionListItemDTO, error)
}

// ============================
// Queries (CRUD / read-by-id)
// ============================

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
	p, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return productiondom.Production{}, err
	}
	if p == nil {
		// RepositoryPort 実装側が nil を返した場合も NotFound 相当として扱う
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *p, nil
}

// RepositoryPort に Exists は無いので、GetByID ベースで存在確認する
func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
	_, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, productiondom.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ============================
// List APIs (delegated to query service)
// ============================

// List returns productions by enforcing the only allowed list route at the application boundary:
// companyId -> productBlueprintIds -> productions.
// This usecase delegates to the query service; legacy implementation is removed.
func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	if u.listQuery == nil {
		return nil, errors.New("internal: ProductionUsecase.listQuery is not configured")
	}
	return u.listQuery.ListProductionsByCurrentCompany(ctx)
}

// ListWithAssigneeName returns list page DTOs for /productions.
// DTO enrichment belongs to read/query side; this usecase delegates to the query service.
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]dto.ProductionListItemDTO, error) {
	if u.listQuery == nil {
		return nil, errors.New("internal: ProductionUsecase.listQuery is not configured")
	}
	return u.listQuery.ListProductionsWithAssigneeName(ctx)
}
