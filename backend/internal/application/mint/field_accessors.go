// backend/internal/application/mint/field_accessors.go
package mint

import (
	"reflect"
	"strings"
)

// setIfExistsString sets a string field on a struct (or pointer-to-struct) if it exists and is settable.
func setIfExistsString(target any, fieldName string, value string) {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}

	// string
	if f.Kind() == reflect.String {
		f.SetString(strings.TrimSpace(value))
		return
	}

	// *string
	if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String {
		v := strings.TrimSpace(value)
		if v == "" {
			// 空文字の場合は何もしない（nil に落とす等のポリシーはここでは持たない）
			return
		}
		if f.IsNil() {
			f.Set(reflect.New(f.Type().Elem()))
		}
		f.Elem().SetString(v)
		return
	}
}

// getIfExistsString reads a string field from a struct (or pointer-to-struct) if it exists.
func getIfExistsString(target any, fieldName string) string {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Ptr {
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

	// string
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}

	// *string
	if f.Kind() == reflect.Ptr && !f.IsNil() && f.Elem().Kind() == reflect.String {
		return strings.TrimSpace(f.Elem().String())
	}

	return ""
}
