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
		if t == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		// best-effort
		s := fmt.Sprint(v)
		if s == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(s, "%d", &n)
		return n
	}
}

func asBool(v any) bool {
	if v == nil {
		return false
	}

	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	case int:
		return t != 0
	case int8:
		return t != 0
	case int16:
		return t != 0
	case int32:
		return t != 0
	case int64:
		return t != 0
	case uint:
		return t != 0
	case uint8:
		return t != 0
	case uint16:
		return t != 0
	case uint32:
		return t != 0
	case uint64:
		return t != 0
	case float32:
		return t != 0
	case float64:
		return t != 0
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		return strings.EqualFold(s, "true") || s == "1"
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
	if rv.Kind() == reflect.Ptr {
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

func timeFromMap(m map[string]any, key string) time.Time {
	t, _ := asTime(m[key])
	return t.UTC()
}

func ptrTimeFromMap(m map[string]any, key string) *time.Time {
	t, ok := asTime(m[key])
	if !ok || t.IsZero() {
		return nil
	}

	utc := t.UTC()
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
