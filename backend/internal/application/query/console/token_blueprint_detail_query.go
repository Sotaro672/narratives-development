// backend/internal/application/query/console/token_blueprint_detail_query.go
package query

import (
	"context"
	"fmt"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintMemberNames struct {
	BrandName     string `json:"brandName"`
	AssigneeName  string `json:"assigneeName"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
}

type TokenBlueprintDetailQuery struct {
	tbRepo     tbdom.RepositoryPort
	memberRepo memberdom.Repository
	brandRepo  branddom.Repository
}

func NewTokenBlueprintDetailQuery(
	tbRepo tbdom.RepositoryPort,
	memberRepo memberdom.Repository,
	brandRepo branddom.Repository,
) *TokenBlueprintDetailQuery {
	return &TokenBlueprintDetailQuery{
		tbRepo:     tbRepo,
		memberRepo: memberRepo,
		brandRepo:  brandRepo,
	}
}

func (q *TokenBlueprintDetailQuery) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, TokenBlueprintMemberNames, error) {
	if q == nil || q.tbRepo == nil {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("tokenBlueprint detail query/repo is nil")
	}

	if id == "" {
		return nil, TokenBlueprintMemberNames{}, tbdom.ErrInvalidID
	}

	tb, err := q.tbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, TokenBlueprintMemberNames{}, err
	}
	if tb == nil {
		return nil, TokenBlueprintMemberNames{}, tbdom.ErrNotFound
	}

	names := q.resolveNamesForTokenBlueprint(ctx, tb)

	return tb, names, nil
}

func (q *TokenBlueprintDetailQuery) ResolveMemberNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if q == nil {
		return nil, fmt.Errorf("tokenBlueprint detail query is nil")
	}

	out := make(map[string]string, len(ids))

	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))

	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if q.memberRepo == nil {
		for _, id := range uniq {
			out[id] = ""
		}
		return out, nil
	}

	for _, id := range uniq {
		rec, err := q.memberRepo.GetByID(ctx, id)
		if err != nil {
			out[id] = ""
			continue
		}

		out[id] = memberdom.FormatLastFirst(
			rec.Member.LastName,
			rec.Member.FirstName,
		)
	}

	return out, nil
}

func (q *TokenBlueprintDetailQuery) ResolveBrandNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if q == nil {
		return nil, fmt.Errorf("tokenBlueprint detail query is nil")
	}

	out := make(map[string]string, len(ids))

	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))

	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if q.brandRepo == nil {
		for _, id := range uniq {
			out[id] = ""
		}
		return out, nil
	}

	for _, id := range uniq {
		brand, err := q.brandRepo.GetByID(ctx, id)
		if err != nil {
			out[id] = ""
			continue
		}

		out[id] = brand.Name
	}

	return out, nil
}

func (q *TokenBlueprintDetailQuery) resolveNamesForTokenBlueprint(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
) TokenBlueprintMemberNames {
	if tb == nil {
		return TokenBlueprintMemberNames{}
	}

	memberNameByID, _ := q.ResolveMemberNames(ctx, []string{
		tb.AssigneeID,
		tb.CreatedBy,
		tb.UpdatedBy,
	})

	brandNameByID, _ := q.ResolveBrandNames(ctx, []string{
		tb.BrandID,
	})

	return TokenBlueprintMemberNames{
		BrandName:     brandNameByID[tb.BrandID],
		AssigneeName:  memberNameByID[tb.AssigneeID],
		CreatedByName: memberNameByID[tb.CreatedBy],
		UpdatedByName: memberNameByID[tb.UpdatedBy],
	}
}
