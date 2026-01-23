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

	return tb, name, nil
}

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
