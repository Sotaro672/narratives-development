// backend\internal\application\query\mall\catalog\catalog_query_name.go
package catalogQuery

import (
	"context"
	"reflect"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

func fillProductBlueprintNames(ctx context.Context, r *appresolver.NameResolver, dtoPB *dto.CatalogProductBlueprintDTO) {
	if r == nil || dtoPB == nil {
		return
	}

	brandID := strings.TrimSpace(dtoPB.BrandID)
	companyID := strings.TrimSpace(dtoPB.CompanyID)

	if brandID != "" {
		bn := strings.TrimSpace(r.ResolveBrandName(ctx, brandID))
		if bn != "" {
			setStringFieldBestEffort(dtoPB, "BrandName", bn)
		}
	}

	if companyID != "" {
		cn := strings.TrimSpace(r.ResolveCompanyName(ctx, companyID))
		if cn != "" {
			setStringFieldBestEffort(dtoPB, "CompanyName", cn)
		}
	}
}

// tbdom.Patch は value 型（string/bool）前提。CompanyName は存在しない。
func fillTokenBlueprintPatchNames(ctx context.Context, r *appresolver.NameResolver, p *tbdom.Patch) {
	if r == nil || p == nil {
		return
	}

	brandID := strings.TrimSpace(p.BrandID)
	if brandID != "" && strings.TrimSpace(p.BrandName) == "" {
		if bn := strings.TrimSpace(r.ResolveBrandName(ctx, brandID)); bn != "" {
			p.BrandName = bn
		}
	}
}

func setStringFieldBestEffort(target any, fieldName string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(value)
	case reflect.Pointer:
		if f.Type().Elem().Kind() == reflect.String {
			s := value
			f.Set(reflect.ValueOf(&s))
		}
	}
}

func getStringFieldBestEffort(target any, fieldName string) string {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		return ""
	}

	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return ""
		}
		f = f.Elem()
	}
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	return ""
}
