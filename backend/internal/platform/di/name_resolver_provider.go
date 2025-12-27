// backend/internal/platform/di/name_resolver_provider.go
package di

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strings"

	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
)

// SNSNameResolver exposes a resolver for display names (brand/company) for SNS.
// NOTE:
// - Container に既に NameResolver フィールドがあるため、同名メソッドは衝突する。
// - ここでは SNSNameResolver() として提供する。
// - 生成した resolver は Container の NameResolver フィールドへキャッシュする。
func (c *Container) SNSNameResolver() *appresolver.NameResolver {
	if c == nil {
		return nil
	}

	// ✅ cache (Container にフィールドがある前提)
	if c.NameResolver != nil {
		return c.NameResolver
	}

	// ---- get BrandUsecase / CompanyUsecase (best-effort) ----
	var brandUC *usecase.BrandUsecase
	{
		if x, ok := any(c).(interface{ BrandUsecase() *usecase.BrandUsecase }); ok {
			brandUC = x.BrandUsecase()
		} else if x, ok := any(c).(interface{ GetBrandUsecase() *usecase.BrandUsecase }); ok {
			brandUC = x.GetBrandUsecase()
		}
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
	}

	// brand/company のどちらも無いなら作れない
	if brandUC == nil && companyUC == nil {
		log.Printf("[di] SNSNameResolver unavailable (brandUC=nil & companyUC=nil)")
		return nil
	}

	var br appresolver.BrandNameRepository
	var cr appresolver.CompanyNameRepository

	if brandUC != nil {
		br = &brandNameRepoAdapter{uc: brandUC}
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

	c.NameResolver = r // ✅ cache into existing field
	log.Printf("[di] SNSNameResolver created & cached brandRepo=%t companyRepo=%t", br != nil, cr != nil)
	return r
}

// ------------------------------------------------------------
// adapters: usecase の戻り値型揺れ（T / *T）を吸収して interface を満たす
// ------------------------------------------------------------

type brandNameRepoAdapter struct{ uc any }

func (a *brandNameRepoAdapter) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	if a == nil || a.uc == nil {
		return branddom.Brand{}, errors.New("brandNameRepoAdapter: uc is nil")
	}
	v, err := callRepo(a.uc, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return branddom.Brand{}, err
	}
	if v == nil {
		return branddom.Brand{}, errors.New("brand is nil")
	}

	switch b := v.(type) {
	case branddom.Brand:
		return b, nil
	case *branddom.Brand:
		if b == nil {
			return branddom.Brand{}, errors.New("brand is nil")
		}
		return *b, nil
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			if x, ok := rv.Interface().(*branddom.Brand); ok && x != nil {
				return *x, nil
			}
		}
		return branddom.Brand{}, errors.New("unexpected brand type")
	}
}

type companyNameRepoAdapter struct{ uc any }

func (a *companyNameRepoAdapter) GetByID(ctx context.Context, id string) (companydom.Company, error) {
	if a == nil || a.uc == nil {
		return companydom.Company{}, errors.New("companyNameRepoAdapter: uc is nil")
	}
	v, err := callRepo(a.uc, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return companydom.Company{}, err
	}
	if v == nil {
		return companydom.Company{}, errors.New("company is nil")
	}

	switch c := v.(type) {
	case companydom.Company:
		return c, nil
	case *companydom.Company:
		if c == nil {
			return companydom.Company{}, errors.New("company is nil")
		}
		return *c, nil
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			if x, ok := rv.Interface().(*companydom.Company); ok && x != nil {
				return *x, nil
			}
		}
		return companydom.Company{}, errors.New("unexpected company type")
	}
}
