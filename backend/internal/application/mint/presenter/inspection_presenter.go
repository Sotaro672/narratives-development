// backend/internal/application/mint/presenter/inspection_presenter.go
package presenter

import (
	"context"
	"strings"

	dto "narratives/internal/application/mint/dto"
	resolver "narratives/internal/application/resolver"
)

// PresentInspectionViews は、usecase が返した DTO（ProductBlueprintID まで埋まっている）に対して
// NameResolver を使って表示用の ProductName を埋めて返す presenter です。
func PresentInspectionViews(
	ctx context.Context,
	r *resolver.NameResolver,
	in []dto.MintInspectionView,
) []dto.MintInspectionView {
	if len(in) == 0 {
		return []dto.MintInspectionView{}
	}

	// resolver が無ければそのまま返す（productName 空のまま）
	if r == nil {
		out := make([]dto.MintInspectionView, len(in))
		copy(out, in)
		return out
	}

	out := make([]dto.MintInspectionView, 0, len(in))
	for _, v := range in {
		pbID := strings.TrimSpace(v.ProductBlueprintID)
		if pbID != "" {
			name := strings.TrimSpace(r.ResolveProductName(ctx, pbID))
			if name != "" {
				v.ProductName = name
			}
		}
		out = append(out, v)
	}

	return out
}
