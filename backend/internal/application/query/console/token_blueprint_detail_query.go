// backend/internal/application/query/console/token_blueprint_detail_query.go
package query

import (
	"context"
	"fmt"
	"strings"

	"narratives/internal/application/resolver"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintMemberNames struct {
	AssigneeName  string `json:"assigneeName"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
}

type TokenBlueprintDetailQuery struct {
	tbRepo       tbdom.RepositoryPort
	nameResolver *resolver.NameResolver
}

func NewTokenBlueprintDetailQuery(
	tbRepo tbdom.RepositoryPort,
	nameResolver *resolver.NameResolver,
) *TokenBlueprintDetailQuery {
	return &TokenBlueprintDetailQuery{
		tbRepo:       tbRepo,
		nameResolver: nameResolver,
	}
}

func (q *TokenBlueprintDetailQuery) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, TokenBlueprintMemberNames, error) {
	if q == nil || q.tbRepo == nil {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("tokenBlueprint detail query/repo is nil")
	}

	id = strings.Trim(id, " \t\r\n")
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

	names := q.resolveMemberNamesForTokenBlueprint(ctx, tb)

	return tb, names, nil
}

func (q *TokenBlueprintDetailQuery) ResolveMemberNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if q == nil || q.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint detail query/repo is nil")
	}

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
		return out, nil
	}

	for _, id := range uniq {
		out[id] = q.nameResolver.ResolveMemberName(ctx, id)
	}

	return out, nil
}

func (q *TokenBlueprintDetailQuery) resolveMemberNamesForTokenBlueprint(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
) TokenBlueprintMemberNames {
	if tb == nil {
		return TokenBlueprintMemberNames{}
	}

	m, _ := q.ResolveMemberNames(ctx, []string{
		tb.AssigneeID,
		tb.CreatedBy,
		tb.UpdatedBy,
	})

	assigneeID := strings.Trim(tb.AssigneeID, " \t\r\n")
	createdBy := strings.Trim(tb.CreatedBy, " \t\r\n")
	updatedBy := strings.Trim(tb.UpdatedBy, " \t\r\n")

	return TokenBlueprintMemberNames{
		AssigneeName:  strings.Trim(m[assigneeID], " \t\r\n"),
		CreatedByName: strings.Trim(m[createdBy], " \t\r\n"),
		UpdatedByName: strings.Trim(m[updatedBy], " \t\r\n"),
	}
}
