// backend/internal/platform/di/sns_adapters.go
package di

import (
	"context"
	"errors"
	"reflect"
	"strings"

	snsdto "narratives/internal/application/query/sns/dto"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// sns catalog adapters (DI-only helpers)
// - sns/application/query 側が要求する “小さな port” を、既存 repo に best-effort で接続するためのアダプタ群
// ============================================================

type snsCatalogInventoryRepoAdapter struct {
	repo any
}

func (a *snsCatalogInventoryRepoAdapter) GetByID(ctx context.Context, id string) (*snsdto.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

func (a *snsCatalogInventoryRepoAdapter) GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*snsdto.SNSCatalogInventoryDTO, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog inventory repo: repo is nil")
	}
	pb := strings.TrimSpace(productBlueprintID)
	tb := strings.TrimSpace(tokenBlueprintID)

	methods := []string{
		"GetByProductAndTokenBlueprintID",
		"GetByProductAndTokenBlueprintId",
		"GetByProductAndTokenBlueprintIDs",
		"GetByProductAndTokenBlueprintIds",
	}
	v, err := callRepo(a.repo, methods, ctx, pb, tb)
	if err != nil {
		return nil, err
	}
	return toSNSCatalogInventoryDTO(v)
}

type snsCatalogProductBlueprintRepoAdapter struct {
	repo any
}

func (a *snsCatalogProductBlueprintRepoAdapter) GetByID(ctx context.Context, id string) (*pbdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("sns catalog product repo: repo is nil")
	}
	v, err := callRepo(a.repo, []string{"GetByID", "GetById"}, ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("productBlueprint is nil")
	}
	if pb, ok := v.(*pbdom.ProductBlueprint); ok {
		return pb, nil
	}
	if pb, ok := v.(pbdom.ProductBlueprint); ok {
		cp := pb
		return &cp, nil
	}

	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
		if x, ok := rv.Interface().(*pbdom.ProductBlueprint); ok {
			return x, nil
		}
	}
	return nil, errors.New("unexpected productBlueprint type")
}

func callRepo(repo any, methodNames []string, args ...any) (any, error) {
	rv := reflect.ValueOf(repo)
	if !rv.IsValid() {
		return nil, errors.New("repo is invalid")
	}

	for _, name := range methodNames {
		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		in := make([]reflect.Value, 0, len(args))
		for _, a := range args {
			in = append(in, reflect.ValueOf(a))
		}

		out := m.Call(in)
		if len(out) == 0 {
			return nil, nil
		}

		if len(out) >= 2 {
			if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
				return nil, e
			}
		}
		return out[0].Interface(), nil
	}

	return nil, errors.New("method not found on repo")
}

func toSNSCatalogInventoryDTO(v any) (*snsdto.SNSCatalogInventoryDTO, error) {
	if v == nil {
		return nil, errors.New("inventory is nil")
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, errors.New("inventory is invalid")
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, errors.New("inventory is nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("inventory is not struct")
	}

	getStr := func(names ...string) string {
		for _, n := range names {
			f := rv.FieldByName(n)
			if !f.IsValid() {
				continue
			}
			if f.Kind() == reflect.String {
				return strings.TrimSpace(f.String())
			}
		}
		return ""
	}

	id := getStr("ID", "Id", "InventoryID", "InventoryId")
	pbID := getStr("ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	tbID := getStr("TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId")

	stock := map[string]snsdto.SNSCatalogInventoryModelStockDTO{}

	var sf reflect.Value
	for _, n := range []string{"Stock", "Stocks", "stock"} {
		f := rv.FieldByName(n)
		if f.IsValid() {
			sf = f
			break
		}
	}

	if sf.IsValid() {
		if sf.Kind() == reflect.Pointer {
			if !sf.IsNil() {
				sf = sf.Elem()
			}
		}

		switch sf.Kind() {
		case reflect.Map:
			if sf.Type().Key().Kind() == reflect.String {
				iter := sf.MapRange()
				for iter.Next() {
					modelID := strings.TrimSpace(iter.Key().String())
					if modelID == "" {
						continue
					}

					ids := extractProductIDsFromStockValue(iter.Value())
					products := make(map[string]bool, len(ids))
					for _, pid := range ids {
						pid = strings.TrimSpace(pid)
						if pid == "" {
							continue
						}
						products[pid] = true
					}

					stock[modelID] = snsdto.SNSCatalogInventoryModelStockDTO{
						Products: products,
					}
				}
			}

		case reflect.Slice, reflect.Array:
			for i := 0; i < sf.Len(); i++ {
				it := sf.Index(i)
				if it.Kind() == reflect.Interface && !it.IsNil() {
					it = it.Elem()
				}
				if it.Kind() == reflect.Pointer && !it.IsNil() {
					it = it.Elem()
				}
				if !it.IsValid() {
					continue
				}
				if it.Kind() == reflect.String {
					modelID := strings.TrimSpace(it.String())
					if modelID == "" {
						continue
					}
					if _, ok := stock[modelID]; !ok {
						stock[modelID] = snsdto.SNSCatalogInventoryModelStockDTO{Products: map[string]bool{}}
					}
				}
			}
		}
	}

	return &snsdto.SNSCatalogInventoryDTO{
		ID:                 id,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		Stock:              stock,
	}, nil
}

// stock reflection helpers (modelId -> products)

func extractProductIDsFromStockValue(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		pf := v.FieldByName("Products")
		if pf.IsValid() {
			return extractStringIDs(pf)
		}
	}

	return extractStringIDs(v)
}

func extractStringIDs(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if e.Kind() == reflect.Interface && !e.IsNil() {
				e = e.Elem()
			}
			if e.Kind() == reflect.Pointer {
				if e.IsNil() {
					continue
				}
				e = e.Elem()
			}
			if e.Kind() == reflect.String {
				s := strings.TrimSpace(e.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}
