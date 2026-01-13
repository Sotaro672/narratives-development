// backend/internal/application/usecase/payment_reflect_util.go
package usecase

/*
責任と機能:
- PaymentUsecase 内部で使う "best-effort reflection utility" を提供する。
- order/payment/items が struct / pointer / map など揺れても動くように、
  文字列/数値/スライスの抽出を安全に行う。
- ログ用の mask、trim などの軽量ユーティリティも集約する。
*/

import (
	"fmt"
	"reflect"
	"strings"
)

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func maskID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if len(id) <= 8 {
		return "***"
	}
	return id[:4] + "***" + id[len(id)-4:]
}

func getStringFieldBestEffort(v any, fieldNames ...string) string {
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
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}

		// direct string or named string type
		if f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}

		// pointer to string
		if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String && !f.IsNil() {
			s := strings.TrimSpace(f.Elem().String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}

		// fallback: fmt.Sprint for other kinds (rare)
		if f.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(f.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}

	return ""
}

func getSliceFieldBestEffort(v any, fieldNames ...string) (reflect.Value, bool) {
	if v == nil {
		return reflect.Value{}, false
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return reflect.Value{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
			return f, true
		}
	}
	return reflect.Value{}, false
}

func getStringFieldFromValueBestEffort(rv reflect.Value, fieldNames ...string) string {
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
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
		if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String && !f.IsNil() {
			return strings.TrimSpace(f.Elem().String())
		}
		if f.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(f.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func getIntFieldFromValueBestEffort(rv reflect.Value, fieldNames ...string) int {
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return 0
	}
	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(f.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(f.Uint())
		case reflect.Float32, reflect.Float64:
			return int(f.Float())
		}
		if f.CanInterface() {
			// very last resort
			if n, ok := parseIntFromAny(f.Interface()); ok {
				return n
			}
		}
	}
	return 0
}

func getStringMapValueBestEffort(mv reflect.Value, keys ...string) string {
	if !mv.IsValid() || mv.Kind() != reflect.Map {
		return ""
	}
	for _, k := range keys {
		kv := reflect.ValueOf(k)
		v := mv.MapIndex(kv)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.Kind() == reflect.String {
			s := strings.TrimSpace(v.String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}
		if v.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(v.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func getIntMapValueBestEffort(mv reflect.Value, keys ...string) int {
	if !mv.IsValid() || mv.Kind() != reflect.Map {
		return 0
	}
	for _, k := range keys {
		kv := reflect.ValueOf(k)
		v := mv.MapIndex(kv)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.CanInterface() {
			if n, ok := parseIntFromAny(v.Interface()); ok {
				return n
			}
		}
	}
	return 0
}

func parseIntFromAny(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		// allow "1.0" etc
		var n int
		_, err := fmt.Sscanf(s, "%d", &n)
		if err == nil {
			return n, true
		}
		var f float64
		_, err2 := fmt.Sscanf(s, "%f", &f)
		if err2 == nil {
			return int(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}
