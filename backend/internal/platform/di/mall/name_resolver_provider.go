// backend/internal/platform/di/mall/name_resolver_provider.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

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
//   - NameResolver は（存在すれば）Container の NameResolver フィールドへ best-effort でキャッシュします。
func MallNameResolver(c *consoleDI.Container) *appresolver.NameResolver {
	if c == nil {
		return nil
	}

	// ✅ cache read (best-effort)
	if cached := getNameResolverFieldBestEffort(c); cached != nil {
		return cached
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

	if brandUC == nil && companyUC == nil {
		log.Printf("[di.mall] MallNameResolver unavailable (brandUC=nil & companyUC=nil)")
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

	// ✅ cache write (best-effort)
	setNameResolverFieldBestEffort(c, r)

	log.Printf("[di.mall] MallNameResolver created brandRepo=%t companyRepo=%t cached=%t", br != nil, cr != nil, getNameResolverFieldBestEffort(c) != nil)
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

// ------------------------------------------------------------
// reflection helpers (shared inside package mall)
// ------------------------------------------------------------

// callRepo: target の候補メソッドを順に探して呼び出す（T or (T,error) を許容）
func callRepo(target any, methodNames []string, args ...any) (any, error) {
	if target == nil {
		return nil, errors.New("callRepo: target is nil")
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return nil, errors.New("callRepo: target is invalid")
	}

	for _, name := range methodNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		in := make([]reflect.Value, 0, len(args))
		for _, a := range args {
			in = append(in, reflect.ValueOf(a))
		}
		if m.Type().NumIn() != len(in) {
			continue
		}

		out := m.Call(in)

		switch len(out) {
		case 1:
			return out[0].Interface(), nil
		case 2:
			if !out[1].IsValid() || out[1].IsNil() {
				return out[0].Interface(), nil
			}
			if e, ok := out[1].Interface().(error); ok && e != nil {
				return nil, e
			}
			return nil, fmt.Errorf("callRepo: %s second return is not error", name)
		default:
			return nil, fmt.Errorf("callRepo: %s has unsupported return count=%d", name, len(out))
		}
	}

	return nil, fmt.Errorf("callRepo: no callable method found: %v", methodNames)
}

func getNameResolverFieldBestEffort(c any) *appresolver.NameResolver {
	if c == nil {
		return nil
	}
	rv := reflect.ValueOf(c)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	f := rv.FieldByName("NameResolver")
	if !f.IsValid() || !f.CanInterface() {
		return nil
	}
	if f.Kind() == reflect.Pointer && f.IsNil() {
		return nil
	}
	if r, ok := f.Interface().(*appresolver.NameResolver); ok {
		return r
	}
	return nil
}

func setNameResolverFieldBestEffort(c any, r *appresolver.NameResolver) {
	if c == nil || r == nil {
		return
	}
	rv := reflect.ValueOf(c)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName("NameResolver")
	if !f.IsValid() || !f.CanSet() {
		return
	}
	if f.Type() == reflect.TypeOf((*appresolver.NameResolver)(nil)) {
		f.Set(reflect.ValueOf(r))
	}
}
