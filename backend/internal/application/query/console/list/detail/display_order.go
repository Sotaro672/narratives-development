// backend/internal/application/query/console/list/detail/display_order.go
//
// 機能: productBlueprintPatch から displayOrder を抽出する
// 責任:
// - Patch.ModelRefs の正規構造から modelID/displayOrder を読み取る
// - modelID -> *displayOrder の辞書を生成する（0 は未設定として nil）
//
// NOTE:
// - pbPatchRepo は (productBlueprint.Patch, error) を返す前提（typed）
// - productBlueprint.Patch.ModelRefs は *[]productBlueprint.ModelRef
package detail

import (
	"context"
)

func (q *ListDetailQuery) buildDisplayOrderByModelID(ctx context.Context, productBlueprintID string) map[string]*int {
	out := map[string]*int{}

	if q == nil || q.pbPatchRepo == nil {
		return out
	}

	pbID := productBlueprintID
	if pbID == "" {
		return out
	}

	patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID)
	if err != nil {
		return out
	}

	if patch.ModelRefs == nil || len(*patch.ModelRefs) == 0 {
		return out
	}

	seen := map[string]struct{}{}
	for _, r := range *patch.ModelRefs {
		mid := r.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}

		var ptr *int
		if r.DisplayOrder != 0 {
			x := r.DisplayOrder
			ptr = &x
		}
		out[mid] = ptr
	}

	return out
}
