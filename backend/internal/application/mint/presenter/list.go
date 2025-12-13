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

func (p *ListPresenter) ToRowDTO(ctx context.Context, m mintdom.Mint) dto.MintListRowDTO {
	tokenName := ""
	if p != nil && p.NameResolver != nil {
		tokenName = strings.TrimSpace(p.NameResolver.ResolveTokenName(ctx, m.TokenBlueprintID))
	}

	var createdByName *string
	if p != nil && p.NameResolver != nil {
		// createdBy は string なので pointer 化して resolver の既存APIに合わせるならこう
		createdBy := strings.TrimSpace(m.CreatedBy)
		if createdBy != "" {
			name := strings.TrimSpace(p.NameResolver.ResolveMemberName(ctx, createdBy))
			if name != "" {
				createdByName = &name
			}
		}
	}

	var mintedAt *string
	if m.Minted && m.MintedAt != nil && !m.MintedAt.IsZero() {
		s := m.MintedAt.In(time.Local).Format("2006/01/02") // JST運用なら Local でOK。UTC固定なら UTC で。
		mintedAt = &s
	}

	return dto.MintListRowDTO{
		TokenName:     tokenName,
		CreatedByName: createdByName,
		MintedAt:      mintedAt,
	}
}
