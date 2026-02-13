// backend\internal\application\query\mall\catalog\catalog_query_model_mapper.go
package catalogQuery

import (
	"reflect"

	dto "narratives/internal/application/query/mall/dto"
)

func toCatalogModelVariationDTOAny(v any) (dto.CatalogModelVariationDTO, bool) {
	if v == nil {
		return dto.CatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return dto.CatalogModelVariationDTO{}, false
	}

	// pointer unwrap
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return dto.CatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return dto.CatalogModelVariationDTO{}, false
	}

	// domain (model.ModelVariation) を正とする:
	// ID, ProductBlueprintID, ModelNumber, Size, Color(Name, RGB), Measurements
	idf := rv.FieldByName("ID")
	if !idf.IsValid() || idf.Kind() != reflect.String || idf.String() == "" {
		return dto.CatalogModelVariationDTO{}, false
	}

	pbf := rv.FieldByName("ProductBlueprintID")
	if !pbf.IsValid() || pbf.Kind() != reflect.String {
		return dto.CatalogModelVariationDTO{}, false
	}

	mnf := rv.FieldByName("ModelNumber")
	if !mnf.IsValid() || mnf.Kind() != reflect.String {
		return dto.CatalogModelVariationDTO{}, false
	}

	szf := rv.FieldByName("Size")
	if !szf.IsValid() || szf.Kind() != reflect.String {
		return dto.CatalogModelVariationDTO{}, false
	}

	out := dto.CatalogModelVariationDTO{
		ID:                 idf.String(),
		ProductBlueprintID: pbf.String(),
		ModelNumber:        mnf.String(),
		Size:               szf.String(),

		ColorName: "",
		ColorRGB:  0,

		Measurements: map[string]int{},

		StockKeys: 0,
	}

	// Color (domain: Color{Name string, RGB int})
	cf := rv.FieldByName("Color")
	if cf.IsValid() {
		if cf.Kind() == reflect.Pointer {
			if !cf.IsNil() {
				cf = cf.Elem()
			}
		}
		if cf.IsValid() && cf.Kind() == reflect.Struct {
			nf := cf.FieldByName("Name")
			if nf.IsValid() && nf.Kind() == reflect.String {
				out.ColorName = nf.String()
			}
			rf := cf.FieldByName("RGB")
			if rf.IsValid() {
				out.ColorRGB = toInt(rf)
			}
		}
	}

	// Measurements (domain: map[string]int)
	mf := rv.FieldByName("Measurements")
	if mf.IsValid() {
		if mf.Kind() == reflect.Pointer {
			if !mf.IsNil() {
				mf = mf.Elem()
			}
		}
		if mf.IsValid() && mf.Kind() == reflect.Map && mf.Type().Key().Kind() == reflect.String {
			mp := make(map[string]int, mf.Len())
			iter := mf.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				if k == "" {
					continue
				}
				mp[k] = toInt(iter.Value())
			}
			out.Measurements = mp
		}
	}

	if out.Measurements == nil {
		out.Measurements = map[string]int{}
	}

	return out, true
}
