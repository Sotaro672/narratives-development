// backend/internal/platform/di/mall/name_resolver_provider.go
package mall

import (
	"context"
	"errors"
	"log"

	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"

	consoleDI "narratives/internal/platform/di/console"
)

// MallNameResolver exposes a resolver for display names (brand/company) for Mall.
//
// NOTE:
//   - Goでは「別パッケージの型（= non-local type）」に対してメソッドを追加できません。
//     そのため (c *consoleDI.Container) のレシーバメソッドではなく、関数として提供します。
//     必要な場合は呼び出し元の DI container 側で明示的に保持してください。
func MallNameResolver(c *consoleDI.Container) *appresolver.NameResolver {
	if c == nil {
		return nil
	}

	var brandRepo branddom.Repository
	if c.BrandRepo != nil {
		brandRepo = c.BrandRepo
	}

	var companyUC *usecase.CompanyUsecase
	{
		if x, ok := any(c).(interface {
			CompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.CompanyUsecase()
		} else if x, ok := any(c).(interface {
			GetCompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.GetCompanyUsecase()
		}

		if companyUC == nil {
			companyUC = c.CompanyUC
		}
	}

	if brandRepo == nil && companyUC == nil {
		log.Printf("[di.mall] MallNameResolver unavailable (brandRepo=nil & companyUC=nil)")
		return nil
	}

	var br appresolver.BrandNameRepository
	var cr appresolver.CompanyNameRepository

	if brandRepo != nil {
		br = &brandNameRepoAdapter{repo: brandRepo}
	}
	if companyUC != nil {
		cr = &companyNameRepoAdapter{uc: companyUC}
	}

	r := appresolver.NewNameResolver(
		br,
		cr,
		nil, // productBlueprintRepo
		nil, // memberRepo
		nil, // modelNumberRepo
		nil, // tokenBlueprintRepo
	)

	log.Printf("[di.mall] MallNameResolver created brandRepo=%t companyRepo=%t", br != nil, cr != nil)

	return r
}

// ------------------------------------------------------------
// adapters
// ------------------------------------------------------------

type brandNameRepoAdapter struct {
	repo branddom.Repository
}

func (a *brandNameRepoAdapter) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	if a == nil || a.repo == nil {
		return branddom.Brand{}, errors.New("brandNameRepoAdapter: repo is nil")
	}
	if id == "" {
		return branddom.Brand{}, errors.New("brandNameRepoAdapter: id is empty")
	}

	return a.repo.GetByID(ctx, id)
}

type companyNameRepoAdapter struct {
	uc *usecase.CompanyUsecase
}

func (a *companyNameRepoAdapter) GetByID(ctx context.Context, id string) (companydom.Company, error) {
	if a == nil || a.uc == nil {
		return companydom.Company{}, errors.New("companyNameRepoAdapter: uc is nil")
	}
	if id == "" {
		return companydom.Company{}, errors.New("companyNameRepoAdapter: id is empty")
	}

	return a.uc.GetByID(ctx, id)
}
