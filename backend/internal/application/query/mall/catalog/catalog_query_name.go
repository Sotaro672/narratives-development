// backend/internal/application/query/mall/catalog/catalog_query_name.go

package catalogQuery

import (
	"context"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

func fillProductBlueprintNames(ctx context.Context, r *appresolver.NameResolver, dtoPB *dto.CatalogProductBlueprintDTO) {
	if r == nil || dtoPB == nil {
		return
	}

	if dtoPB.BrandID != "" {
		if bn := r.ResolveBrandName(ctx, dtoPB.BrandID); bn != "" {
			dtoPB.BrandName = bn
		}
	}

	if dtoPB.CompanyID != "" {
		if cn := r.ResolveCompanyName(ctx, dtoPB.CompanyID); cn != "" {
			dtoPB.CompanyName = cn
		}
	}
}

// tbdom.Patch は value 型（string/bool）前提。CompanyName は存在しない。
func fillTokenBlueprintPatchNames(ctx context.Context, r *appresolver.NameResolver, p *tbdom.Patch) {
	if r == nil || p == nil {
		return
	}

	if p.BrandID != "" && p.BrandName == "" {
		if bn := r.ResolveBrandName(ctx, p.BrandID); bn != "" {
			p.BrandName = bn
		}
	}
}
