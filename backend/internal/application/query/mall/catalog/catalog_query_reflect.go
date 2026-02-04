// backend\internal\application\query\mall\catalog\catalog_query_reflect.go
package catalogQuery

import (
	"reflect"
	"strings"

	pbdom "narratives/internal/domain/productBlueprint"
)

func pickStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
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

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
	}
	return ""
}

func pickIntField(v any, fieldNames ...string) int {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return 0
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		return toInt(f)
	}
	return 0
}

func pickProductIDTagType(pb *pbdom.ProductBlueprint) string {
	if pb == nil {
		return ""
	}

	if s := pickStringField(*pb, "ProductIDTagType", "ProductIdTagType", "productIdTagType"); s != "" {
		return s
	}

	rv := reflect.ValueOf(pb)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, parent := range []string{"ProductIDTag", "ProductIdTag", "productIdTag"} {
		f := rv.FieldByName(parent)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		if f.Kind() != reflect.Struct {
			continue
		}

		tf := f.FieldByName("Type")
		if tf.IsValid() && tf.Kind() == reflect.String {
			return strings.TrimSpace(tf.String())
		}
	}

	return ""
}

func extractID(v any) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
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

	for _, name := range []string{"ID", "Id", "ModelID", "ModelId"} {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
	}

	return ""
}

func toInt(v reflect.Value) int {
	if !v.IsValid() {
		return 0
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	default:
		return 0
	}
}
