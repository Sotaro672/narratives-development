// backend/internal/platform/di/console/helpers_reflect.go
package console

import (
	"reflect"
	"strings"
)

// callOptionalMethod calls obj.<methodName>(arg) when such method exists (best-effort).
func callOptionalMethod(obj any, methodName string, arg any) {
	if obj == nil || strings.TrimSpace(methodName) == "" || arg == nil {
		return
	}
	rv := reflect.ValueOf(obj)
	m := rv.MethodByName(methodName)
	if !m.IsValid() {
		return
	}
	if m.Type().NumIn() != 1 {
		return
	}
	av := reflect.ValueOf(arg)
	if !av.IsValid() {
		return
	}
	if !av.Type().AssignableTo(m.Type().In(0)) {
		if m.Type().In(0).Kind() == reflect.Interface && av.Type().Implements(m.Type().In(0)) {
			m.Call([]reflect.Value{av})
		}
		return
	}
	m.Call([]reflect.Value{av})
}
