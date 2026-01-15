// backend/internal/application/mint/product_blueprint_resolver.go
package mint

import (
	"context"
	"errors"
	"reflect"
	"strings"
)

// production から ProductBlueprintID を（存在すれば）取り出す
// ※ prodRepo の具体型に依存しないため、GetByID/Get を reflect で試す
func (u *MintUsecase) resolveProductBlueprintIDFromProduction(ctx context.Context, productionID string) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}

	call := func(methodName string) (any, error) {
		rv := reflect.ValueOf(u.prodRepo)
		m := rv.MethodByName(methodName)
		if !m.IsValid() {
			return nil, errors.New("method not found: " + methodName)
		}
		out := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(productionID)})
		if len(out) != 2 {
			return nil, errors.New("unexpected return values from " + methodName)
		}
		if !out[1].IsNil() {
			if err, ok := out[1].Interface().(error); ok {
				return nil, err
			}
			return nil, errors.New("non-error type returned as error")
		}
		return out[0].Interface(), nil
	}

	var prod any
	if p, err := call("GetByID"); err == nil {
		prod = p
	} else if p, err := call("Get"); err == nil {
		prod = p
	} else {
		return ""
	}

	v := reflect.ValueOf(prod)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range []string{"ProductBlueprintID", "ProductBlueprintId"} {
		f := v.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
		if f.Kind() == reflect.Ptr && !f.IsNil() && f.Elem().Kind() == reflect.String {
			return strings.TrimSpace(f.Elem().String())
		}
	}

	return ""
}
