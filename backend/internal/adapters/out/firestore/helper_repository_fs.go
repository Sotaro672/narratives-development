// backend/internal/adapters/out/firestore/helper_repository_fs.go
package firestore

import (
	"reflect"
	"time"

	"cloud.google.com/go/firestore"
)

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

func normalizeOptionalString(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	normalized := *value
	return &normalized
}

func setOptionalString(m map[string]any, key string, value *string) {
	normalized := normalizeOptionalString(value)
	if normalized != nil {
		m[key] = *normalized
	}
}

func appendOptionalStringUpdate(updates []firestore.Update, path string, value *string) []firestore.Update {
	normalized := normalizeOptionalString(value)
	if normalized == nil {
		return append(updates, firestore.Update{Path: path, Value: firestore.Delete})
	}

	return append(updates, firestore.Update{Path: path, Value: *normalized})
}

func optionalStringEqual(left *string, right *string) bool {
	left = normalizeOptionalString(left)
	right = normalizeOptionalString(right)

	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func setOptionalTime(m map[string]any, key string, value *time.Time) {
	if value != nil && !value.IsZero() {
		m[key] = value.UTC()
	}
}

func optionalStringFromPatch(value *string) *string {
	return normalizeOptionalString(value)
}

func optionalTimeFromPatch(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
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
