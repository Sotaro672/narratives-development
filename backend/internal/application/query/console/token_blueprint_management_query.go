// backend/internal/application/query/console/token_blueprint_management_query.go
package query

import (
	"context"
	"fmt"

	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintWithMemberNames struct {
	TokenBlueprint tbdom.TokenBlueprint
	MemberNames    TokenBlueprintMemberNames
}

type TokenBlueprintWithMemberNamesPage struct {
	Items      []TokenBlueprintWithMemberNames
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

type TokenBlueprintManagementQuery struct {
	tbRepo     tbdom.RepositoryPort
	memberRepo memberdom.Repository
	brandRepo  branddom.Repository
}

func NewTokenBlueprintManagementQuery(
	tbRepo tbdom.RepositoryPort,
	memberRepo memberdom.Repository,
	brandRepo branddom.Repository,
) *TokenBlueprintManagementQuery {
	return &TokenBlueprintManagementQuery{
		tbRepo:     tbRepo,
		memberRepo: memberRepo,
		brandRepo:  brandRepo,
	}
}

func (q *TokenBlueprintManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page domcommon.Page,
) (TokenBlueprintWithMemberNamesPage, error) {
	if q == nil || q.tbRepo == nil {
		return TokenBlueprintWithMemberNamesPage{}, fmt.Errorf("tokenBlueprint management query/repo is nil")
	}

	if companyID == "" {
		return TokenBlueprintWithMemberNamesPage{}, tbdom.ErrInvalidCompanyID
	}

	result, err := q.tbRepo.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		return TokenBlueprintWithMemberNamesPage{}, err
	}

	return q.attachResolvedNames(ctx, result)
}

func (q *TokenBlueprintManagementQuery) ResolveTokenBlueprintNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if q == nil || q.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint management query/repo is nil")
	}

	result := make(map[string]string, len(ids))

	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}

		tb, err := q.tbRepo.GetByID(ctx, id)
		if err != nil || tb == nil {
			result[id] = ""
			continue
		}

		result[id] = tb.Name
	}

	return result, nil
}

func (q *TokenBlueprintManagementQuery) attachResolvedNames(
	ctx context.Context,
	result domcommon.PageResult[tbdom.TokenBlueprint],
) (TokenBlueprintWithMemberNamesPage, error) {
	memberIDs := make([]string, 0, len(result.Items)*3)
	brandIDs := make([]string, 0, len(result.Items))

	for i := range result.Items {
		memberIDs = append(memberIDs,
			result.Items[i].AssigneeID,
			result.Items[i].CreatedBy,
			result.Items[i].UpdatedBy,
		)

		brandIDs = append(brandIDs, result.Items[i].BrandID)
	}

	nameByMemberID := q.resolveMemberNames(ctx, memberIDs)
	nameByBrandID := q.resolveBrandNames(ctx, brandIDs)

	items := make([]TokenBlueprintWithMemberNames, 0, len(result.Items))
	for i := range result.Items {
		tb := result.Items[i]

		brandID := tb.BrandID
		assigneeID := tb.AssigneeID
		createdBy := tb.CreatedBy
		updatedBy := tb.UpdatedBy

		items = append(items, TokenBlueprintWithMemberNames{
			TokenBlueprint: tb,
			MemberNames: TokenBlueprintMemberNames{
				BrandName:     nameByBrandID[brandID],
				AssigneeName:  nameByMemberID[assigneeID],
				CreatedByName: nameByMemberID[createdBy],
				UpdatedByName: nameByMemberID[updatedBy],
			},
		})
	}

	return TokenBlueprintWithMemberNamesPage{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}, nil
}

func (q *TokenBlueprintManagementQuery) resolveMemberNames(
	ctx context.Context,
	ids []string,
) map[string]string {
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
		return out
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

	return out
}

func (q *TokenBlueprintManagementQuery) resolveBrandNames(
	ctx context.Context,
	ids []string,
) map[string]string {
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
		return out
	}

	for _, id := range uniq {
		brand, err := q.brandRepo.GetByID(ctx, id)
		if err != nil {
			out[id] = ""
			continue
		}

		out[id] = brand.Name
	}

	return out
}
