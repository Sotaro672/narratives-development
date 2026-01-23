// backend/internal/application/usecase/tokenBlueprint_metadata_usecase.go
package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintMetadataUsecase handles metadataUri composition policy.
type TokenBlueprintMetadataUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintMetadataUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintMetadataUsecase {
	return &TokenBlueprintMetadataUsecase{tbRepo: tbRepo}
}

// EnsureMetadataURI sets metadataUri if empty. Intended to be called post-create.
func (u *TokenBlueprintMetadataUsecase) EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint metadata usecase/repo is nil")
	}
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}
	if strings.TrimSpace(tb.ID) == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}
	if strings.TrimSpace(tb.MetadataURI) != "" {
		return tb, nil
	}

	uri := buildMetadataResolverURL(tb.ID)
	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
		UpdatedAt:   nil,
		UpdatedBy:   ptr(strings.TrimSpace(actorID)),
		DeletedAt:   nil,
		DeletedBy:   nil,
	})
	if err != nil {
		// 呼び出し側で「create自体は成功」として扱いたいケースがあるため、そのまま返す
		return tb, err
	}
	if updated == nil {
		return tb, nil
	}
	return updated, nil
}

// buildMetadataResolverURL builds resolver URL for metadataUri.
// NOTE: base URL は環境差分が大きいので env で明示する。
func buildMetadataResolverURL(tokenBlueprintID string) string {
	base := strings.TrimSpace(os.Getenv("TOKEN_METADATA_BASE_URL"))
	if base == "" {
		// 例: "https://api.example.com" のように設定する想定。
		// 未設定の場合は空のままにする（validate では必須にしない方針）。
		return ""
	}
	base = strings.TrimSuffix(base, "/")
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("%s/v1/token-blueprints/%s/metadata", base, id)
}
