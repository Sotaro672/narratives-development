// backend/internal/adapters/out/firestore/productBlueprint/repository_helpers_modelrefs.go
// Responsibility: modelRefs の正規化・マージ・再採番・クローン等、modelRefs 操作の共通処理を提供する。
package productBlueprint

import (
	"sort"
	"strings"

	pbdom "narratives/internal/domain/productBlueprint"
)

func sanitizeModelRefs(in []pbdom.ModelRef) ([]pbdom.ModelRef, error) {
	// displayOrder 順で安定化し、modelId の重複は先勝ちで除外、最後に 1..N で再採番
	tmp := make([]pbdom.ModelRef, 0, len(in))
	seen := make(map[string]struct{}, len(in))

	// まずは入力を displayOrder で安定ソート（同順位は入力順）
	withIdx := make([]struct {
		ref pbdom.ModelRef
		idx int
	}, 0, len(in))
	for i, r := range in {
		withIdx = append(withIdx, struct {
			ref pbdom.ModelRef
			idx int
		}{ref: r, idx: i})
	}
	sort.SliceStable(withIdx, func(i, j int) bool {
		ri, rj := withIdx[i].ref, withIdx[j].ref
		if ri.DisplayOrder == rj.DisplayOrder {
			return withIdx[i].idx < withIdx[j].idx
		}
		return ri.DisplayOrder < rj.DisplayOrder
	})

	ids := make([]string, 0, len(in))
	for _, w := range withIdx {
		mid := strings.TrimSpace(w.ref.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	// 1..N で再採番
	for i, mid := range ids {
		tmp = append(tmp, pbdom.ModelRef{
			ModelID:      mid,
			DisplayOrder: i + 1,
		})
	}
	return tmp, nil
}

func mergeAndRenumberModelRefs(existing []pbdom.ModelRef, appendRefs []pbdom.ModelRef) []pbdom.ModelRef {
	seen := make(map[string]struct{}, len(existing)+len(appendRefs))
	ids := make([]string, 0, len(existing)+len(appendRefs))

	// existing は displayOrder で安定化してから取り込む
	ex := cloneModelRefs(existing)
	sort.SliceStable(ex, func(i, j int) bool { return ex[i].DisplayOrder < ex[j].DisplayOrder })

	for _, r := range ex {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	// appendRefs は displayOrder で安定化してから末尾に追加
	ap := cloneModelRefs(appendRefs)
	sort.SliceStable(ap, func(i, j int) bool { return ap[i].DisplayOrder < ap[j].DisplayOrder })

	for _, r := range ap {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		ids = append(ids, mid)
	}

	out := make([]pbdom.ModelRef, 0, len(ids))
	for i, mid := range ids {
		out = append(out, pbdom.ModelRef{
			ModelID:      mid,
			DisplayOrder: i + 1,
		})
	}
	return out
}

func cloneModelRefs(in []pbdom.ModelRef) []pbdom.ModelRef {
	if in == nil {
		return nil
	}
	out := make([]pbdom.ModelRef, len(in))
	copy(out, in)
	return out
}
