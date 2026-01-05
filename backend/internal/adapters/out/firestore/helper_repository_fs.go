package firestore

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(v)
	}
}

func asInt(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case string:
		tt := strings.TrimSpace(t)
		if tt == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(tt, "%d", &n)
		return n
	default:
		// best-effort
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(s, "%d", &n)
		return n
	}
}

// asTime returns (time, ok)
func asTime(v any) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	switch t := v.(type) {
	case time.Time:
		return t, true
	default:
		return time.Time{}, false
	}
}
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
func containsString(xs []string, v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(xs) == 0 {
		return false
	}
	for _, x := range xs {
		if strings.TrimSpace(x) == v {
			return true
		}
	}
	return false
}
