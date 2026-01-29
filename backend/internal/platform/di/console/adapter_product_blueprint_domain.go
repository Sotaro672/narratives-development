// backend/internal/platform/di/console/adapter_product_blueprint_domain.go
package console

import (
	"context"
	"strings"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ========================================
// productBlueprint ドメインサービス用アダプタ
// ========================================

type productBlueprintDomainRepoAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *productBlueprintDomainRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInternal
	}
	return a.repo.GetByID(ctx, strings.TrimSpace(id))
}

// companyId → productBlueprintIds を repo に委譲
func (a *productBlueprintDomainRepoAdapter) ListIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if a == nil || a.repo == nil {
		return nil, productbpdom.ErrInternal
	}
	return a.repo.ListIDsByCompany(ctx, strings.TrimSpace(companyID))
}
