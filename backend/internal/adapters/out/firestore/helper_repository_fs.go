// backend/internal/adapters/out/firestore/helper_repository_fs.go
package firestore

import (
	"reflect"
	"time"
)

func asString(v any) string {
	if v == nil {
		return ""
	}

	value, ok := v.(string)
	if !ok {
		return ""
	}

	return value
}

func asInt(v any) int {
	if v == nil {
		return 0
	}

	value, ok := v.(int64)
	if !ok {
		return 0
	}

	return int(value)
}

func asBool(v any) bool {
	if v == nil {
		return false
	}

	value, ok := v.(bool)
	if !ok {
		return false
	}

	return value
}

func containsString(xs []string, v string) bool {
	if v == "" || len(xs) == 0 {
		return false
	}

	for _, x := range xs {
		if x == v {
			return true
		}
	}

	return false
}

func getStringField(obj any, field string) string {
	rv := reflect.ValueOf(obj)

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return ""
	}

	f := rv.FieldByName(field)
	if !f.IsValid() {
		return ""
	}

	if f.Kind() == reflect.String {
		return f.String()
	}

	return ""
}

func setOptionalString(m map[string]any, key string, value *string) {
	if value != nil && *value != "" {
		m[key] = *value
	}
}

func setOptionalTime(m map[string]any, key string, value *time.Time) {
	if value != nil && !value.IsZero() {
		m[key] = value.UTC()
	}
}

func optionalStringFromPatch(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	v := *value
	return &v
}

func optionalTimeFromPatch(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
}

func ptrStringFromMap(m map[string]any, key string) *string {
	s := asString(m[key])
	if s == "" {
		return nil
	}

	return &s
}

func ptrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

func anyImageMatches[T any](items []T, fn func(T) bool) bool {
	for _, item := range items {
		if fn(item) {
			return true
		}
	}

	return false
}
