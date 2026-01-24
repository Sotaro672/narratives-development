// backend/internal/application/usecase/tokenBlueprint_query_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintQueryUsecase handles read-model conveniences (joins / name resolutions).
type TokenBlueprintQueryUsecase struct {
	tbRepo    tbdom.RepositoryPort
	memberSvc *memdom.Service
}

func NewTokenBlueprintQueryUsecase(tbRepo tbdom.RepositoryPort, memberSvc *memdom.Service) *TokenBlueprintQueryUsecase {
	return &TokenBlueprintQueryUsecase{
		tbRepo:    tbRepo,
		memberSvc: memberSvc,
	}
}

// TokenBlueprintMemberNames is a resolved name set for console response.
type TokenBlueprintMemberNames struct {
	AssigneeName  string `json:"assigneeName"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
}

// ResolveMemberNames resolves memberId -> display name (best-effort).
// - dedup ids
// - if memberSvc is nil, returns empty map
// - if resolution fails for an id, value becomes ""
func (u *TokenBlueprintQueryUsecase) ResolveMemberNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	out := make(map[string]string, len(ids))
	if u.memberSvc == nil {
		// memberSvc が無い構成でも落とさない（空で返す）
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			out[id] = ""
		}
		return out, nil
	}

	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	// ベストエフォート：1件失敗しても他は返す
	for _, mid := range uniq {
		name, err := u.memberSvc.GetNameLastFirstByID(ctx, mid)
		if err != nil {
			out[mid] = ""
			continue
		}
		out[mid] = strings.TrimSpace(name)
	}

	return out, nil
}

// GetByIDWithCreatorName keeps backward-compat method (optional).
func (u *TokenBlueprintQueryUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	if u == nil || u.tbRepo == nil {
		return nil, "", fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	tid := strings.TrimSpace(id)

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, "", err
	}

	if u.memberSvc == nil {
		return tb, "", nil
	}

	memberID := strings.TrimSpace(tb.CreatedBy)
	if memberID == "" {
		return tb, "", nil
	}

	name, err := u.memberSvc.GetNameLastFirstByID(ctx, memberID)
	if err != nil {
		return tb, "", nil
	}

	return tb, strings.TrimSpace(name), nil
}

// GetByIDWithMemberNames returns tb and resolved member names for console response.
func (u *TokenBlueprintQueryUsecase) GetByIDWithMemberNames(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, TokenBlueprintMemberNames, error) {
	if u == nil || u.tbRepo == nil {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	tid := strings.TrimSpace(id)
	if tid == "" {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("id is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, TokenBlueprintMemberNames{}, err
	}
	if tb == nil {
		return nil, TokenBlueprintMemberNames{}, tbdom.ErrNotFound
	}

	ids := []string{
		strings.TrimSpace(tb.AssigneeID),
		strings.TrimSpace(tb.CreatedBy),
		strings.TrimSpace(tb.UpdatedBy),
	}

	m, _ := u.ResolveMemberNames(ctx, ids)

	return tb, TokenBlueprintMemberNames{
		AssigneeName:  m[strings.TrimSpace(tb.AssigneeID)],
		CreatedByName: m[strings.TrimSpace(tb.CreatedBy)],
		UpdatedByName: m[strings.TrimSpace(tb.UpdatedBy)],
	}, nil
}

// ResolveNames resolves tokenBlueprint id -> tokenBlueprint name (as-is).
func (u *TokenBlueprintQueryUsecase) ResolveNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {

	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	result := make(map[string]string, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		name, err := u.tbRepo.GetNameByID(ctx, id)
		if err != nil {
			result[id] = ""
			continue
		}

		result[id] = strings.TrimSpace(name)
	}

	return result, nil
}
