// backend/internal/application/mint/presenter/list.go
package presenter

import (
	"context"
	"strings"
	"time"

	"narratives/internal/application/mint/dto"
	"narratives/internal/application/resolver"
	mintdom "narratives/internal/domain/mint"
)

type ListPresenter struct {
	NameResolver *resolver.NameResolver
}

func NewListPresenter(r *resolver.NameResolver) *ListPresenter {
	return &ListPresenter{NameResolver: r}
}

// ToRowDTO converts Mint(domain) -> MintListRowDTO (list screen DTO).
// - dto.MintListRowDTO の定義に合わせて string / *string を厳密に合わせる
// - MintedAt は RFC3339 (nil なら未mint)
func (p *ListPresenter) ToRowDTO(ctx context.Context, m mintdom.Mint) dto.MintListRowDTO {
	inspectionID := strings.TrimSpace(m.InspectionID)
	mintID := strings.TrimSpace(m.ID)
	tokenBlueprintID := strings.TrimSpace(m.TokenBlueprintID)

	// tokenName（名前解決）
	tokenName := ""
	if p != nil && p.NameResolver != nil && tokenBlueprintID != "" {
		tokenName = strings.TrimSpace(p.NameResolver.ResolveTokenName(ctx, tokenBlueprintID))
	}

	// createdByName（名前解決。取れなければ createdBy をそのまま返す）
	createdBy := strings.TrimSpace(m.CreatedBy)
	createdByName := ""
	if p != nil && p.NameResolver != nil && createdBy != "" {
		if name := strings.TrimSpace(p.NameResolver.ResolveMemberName(ctx, createdBy)); name != "" {
			createdByName = name
		}
	}
	if createdByName == "" {
		createdByName = createdBy
	}

	// mintedAt（RFC3339）
	var mintedAt *string
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		s := m.MintedAt.UTC().Format(time.RFC3339)
		mintedAt = &s
	}

	return dto.MintListRowDTO{
		InspectionID:   inspectionID,
		MintID:         mintID,
		TokenBlueprint: tokenBlueprintID,

		TokenName:     tokenName,
		CreatedByName: createdByName,
		MintedAt:      mintedAt,
	}
}
