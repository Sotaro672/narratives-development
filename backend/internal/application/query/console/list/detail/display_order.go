// backend/internal/application/query/console/list/detail/display_order.go
//
// 機能: productBlueprintPatch から displayOrder を抽出する
// 責任:
// - Patch(ModelRefs) の構造差分に強い reflection で modelID/displayOrder を読み取る
// - modelID -> *displayOrder の辞書を生成する（0 は未設定として nil）
//
// NOTE:
// - pbPatchRepo は (any, error) を返す前提（DI側で any へ合わせる）
package detail

import (
	"context"
	"reflect"
	"strings"
)

func (q *ListDetailQuery) buildDisplayOrderByModelID(ctx context.Context, productBlueprintID string) map[string]*int {
	out := map[string]*int{}

	if q == nil || q.pbPatchRepo == nil {
		return out
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return out
	}

	patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID)
	if err != nil {
		return out
	}

	refs := extractModelRefsFromPBPatchAny(patch)
	if len(refs) == 0 {
		return out
	}

	seen := map[string]struct{}{}
	for _, r := range refs {
		mid := strings.TrimSpace(r.modelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}

		var ptr *int
		if r.displayOrder != 0 {
			x := r.displayOrder
			ptr = &x
		}
		out[mid] = ptr
	}

	return out
}

type modelRefAny struct {
	modelID      string
	displayOrder int
}

// reflection: patch.ModelRefs の型が何であっても拾えるようにする（DisplayOrder が *int の場合も拾う）
func extractModelRefsFromPBPatchAny(patch any) []modelRefAny {
	rv := reflect.ValueOf(patch)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	f := rv.FieldByName("ModelRefs")
	if !f.IsValid() {
		f = rv.FieldByName("modelRefs")
	}
	if !f.IsValid() {
		return nil
	}

	// ModelRefs: *[]T or []T
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil
		}
		f = f.Elem()
	}
	if f.Kind() != reflect.Slice {
		return nil
	}

	out := make([]modelRefAny, 0, f.Len())

	for i := 0; i < f.Len(); i++ {
		it := f.Index(i)
		if it.Kind() == reflect.Pointer {
			if it.IsNil() {
				continue
			}
			it = it.Elem()
		}
		if it.Kind() != reflect.Struct {
			continue
		}

		// ---- modelId ----
		mid := ""
		if mf := it.FieldByName("ModelID"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("ModelId"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("modelId"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("modelID"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		}
		if mid == "" {
			continue
		}

		// ---- displayOrder (int/uint/*int/*uint) ----
		ord := 0
		of := it.FieldByName("DisplayOrder")
		if !of.IsValid() {
			of = it.FieldByName("displayOrder")
		}
		if of.IsValid() && of.CanInterface() {
			if of.Kind() == reflect.Pointer {
				if !of.IsNil() {
					ev := of.Elem()
					switch ev.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						ord = int(ev.Int())
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						ord = int(ev.Uint())
					}
				}
			} else {
				switch of.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					ord = int(of.Int())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					ord = int(of.Uint())
				}
			}
		}

		out = append(out, modelRefAny{modelID: mid, displayOrder: ord})
	}

	return out
}
