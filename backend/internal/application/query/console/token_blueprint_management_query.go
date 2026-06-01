// backend/internal/application/query/console/token_blueprint_management_query.go
package query

import (
	"context"
	"fmt"
	"strings"

	"narratives/internal/application/resolver"
	domcommon "narratives/internal/domain/common"
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
	tbRepo       tbdom.RepositoryPort
	nameResolver *resolver.NameResolver
}

func NewTokenBlueprintManagementQuery(
	tbRepo tbdom.RepositoryPort,
	nameResolver *resolver.NameResolver,
) *TokenBlueprintManagementQuery {
	return &TokenBlueprintManagementQuery{
		tbRepo:       tbRepo,
		nameResolver: nameResolver,
	}
}

func (q *TokenBlueprintManagementQuery) ListForCompany(
	ctx context.Context,
	companyID string,
	brandID string,
	page domcommon.Page,
) (TokenBlueprintWithMemberNamesPage, error) {
	if q == nil || q.tbRepo == nil {
		return TokenBlueprintWithMemberNamesPage{}, fmt.Errorf("tokenBlueprint management query/repo is nil")
	}

	companyID = strings.Trim(companyID, " \t\r\n")
	if companyID == "" {
		return TokenBlueprintWithMemberNamesPage{}, tbdom.ErrInvalidCompanyID
	}

	brandID = strings.Trim(brandID, " \t\r\n")

	var (
		result domcommon.PageResult[tbdom.TokenBlueprint]
		err    error
	)

	if brandID != "" {
		result, err = q.tbRepo.ListByBrandID(ctx, brandID, page)
		if err != nil {
			return TokenBlueprintWithMemberNamesPage{}, err
		}

		filtered := make([]tbdom.TokenBlueprint, 0, len(result.Items))
		for i := range result.Items {
			if strings.Trim(result.Items[i].CompanyID, " \t\r\n") == companyID {
				filtered = append(filtered, result.Items[i])
			}
		}
		result.Items = filtered
	} else {
		result, err = q.tbRepo.ListByCompanyID(ctx, companyID, page)
		if err != nil {
			return TokenBlueprintWithMemberNamesPage{}, err
		}
	}

	return q.attachMemberNames(ctx, result)
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
		id = strings.Trim(id, " \t\r\n")
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

		result[id] = strings.Trim(tb.Name, " \t\r\n")
	}

	return result, nil
}

func (q *TokenBlueprintManagementQuery) attachMemberNames(
	ctx context.Context,
	result domcommon.PageResult[tbdom.TokenBlueprint],
) (TokenBlueprintWithMemberNamesPage, error) {
	ids := make([]string, 0, len(result.Items)*3)
	for i := range result.Items {
		ids = append(ids,
			result.Items[i].AssigneeID,
			result.Items[i].CreatedBy,
			result.Items[i].UpdatedBy,
		)
	}

	nameByMemberID := q.resolveMemberNames(ctx, ids)

	items := make([]TokenBlueprintWithMemberNames, 0, len(result.Items))
	for i := range result.Items {
		tb := result.Items[i]

		assigneeID := strings.Trim(tb.AssigneeID, " \t\r\n")
		createdBy := strings.Trim(tb.CreatedBy, " \t\r\n")
		updatedBy := strings.Trim(tb.UpdatedBy, " \t\r\n")

		items = append(items, TokenBlueprintWithMemberNames{
			TokenBlueprint: tb,
			MemberNames: TokenBlueprintMemberNames{
				AssigneeName:  strings.Trim(nameByMemberID[assigneeID], " \t\r\n"),
				CreatedByName: strings.Trim(nameByMemberID[createdBy], " \t\r\n"),
				UpdatedByName: strings.Trim(nameByMemberID[updatedBy], " \t\r\n"),
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
		id = strings.Trim(id, " \t\r\n")
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if q.nameResolver == nil {
		for _, id := range uniq {
			out[id] = ""
		}
		return out
	}

	for _, id := range uniq {
		out[id] = q.nameResolver.ResolveMemberName(ctx, id)
	}

	return out
}
