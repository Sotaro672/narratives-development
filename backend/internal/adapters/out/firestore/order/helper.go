// backend\internal\adapters\out\firestore\helper_repository_fs.go
package firestore

import (
	"reflect"
	"strings"
)

// asTime returns (time, ok)

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
func getFilterString(v any, field string) (string, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return "", false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		// try lowerCamel (e.g., userId / avatarId)
		f = rv.FieldByName(lowerFirst(field))
		if !f.IsValid() {
			return "", false
		}
	}
	// string
	if f.Kind() == reflect.String {
		return f.String(), true
	}
	// *string
	if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return "", true
		}
		return f.Elem().String(), true
	}
	return "", false
}
