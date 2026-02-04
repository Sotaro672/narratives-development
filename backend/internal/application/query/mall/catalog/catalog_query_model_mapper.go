// backend\internal\application\query\mall\catalog\catalog_query_model_mapper.go
package catalogQuery

import (
	"reflect"
	"strings"

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
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return dto.CatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return dto.CatalogModelVariationDTO{}, false
	}

	id := pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	if strings.TrimSpace(id) == "" {
		return dto.CatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	out := dto.CatalogModelVariationDTO{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(pbID),
		ModelNumber:        strings.TrimSpace(modelNumber),
		Size:               strings.TrimSpace(size),

		ColorName: "",
		ColorRGB:  0,

		Measurements: map[string]int{},

		StockKeys: 0,
	}

	if s := pickStringField(rv.Interface(), "ColorName", "colorName"); s != "" {
		out.ColorName = strings.TrimSpace(s)
	}

	if f := rv.FieldByName("ColorRGB"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("ColorRgb"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("RGB"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("Rgb"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else {
		if c := rv.FieldByName("Color"); c.IsValid() {
			if c.Kind() == reflect.Pointer {
				if !c.IsNil() {
					c = c.Elem()
				}
			}
			if c.IsValid() && c.Kind() == reflect.Struct {
				nf := c.FieldByName("Name")
				if nf.IsValid() && nf.Kind() == reflect.String {
					out.ColorName = strings.TrimSpace(nf.String())
				}
				rf := c.FieldByName("RGB")
				if rf.IsValid() {
					out.ColorRGB = toInt(rf)
				}
			}
		}
	}

	if m := rv.FieldByName("Measurements"); m.IsValid() {
		if m.Kind() == reflect.Pointer {
			if !m.IsNil() {
				m = m.Elem()
			}
		}
		if m.IsValid() && m.Kind() == reflect.Map && m.Type().Key().Kind() == reflect.String {
			mp := make(map[string]int)
			iter := m.MapRange()
			for iter.Next() {
				k := strings.TrimSpace(iter.Key().String())
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
