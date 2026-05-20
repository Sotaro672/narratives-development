// backend/internal/application/tokenBlueprint/tokenBlueprint_icon_usecase.go
package tokenBlueprint

import (
	"context"
	"fmt"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Usecase: TokenBlueprint Icon
// ============================================================
//
// Firebase Storage 移行後の責務:
// - backend は GCS signed URL を発行しない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が getDownloadURL() で取得した downloadURL を backend に渡す
// - backend はその downloadURL を tokenBlueprint.iconUrl として保存する
//
// 旧GCS責務として削除したもの:
// - GCS_SIGNER_EMAIL
// - TOKEN_ICON_BUCKET
// - storage.SignedURL
// - iamcredentials.SignBlob
// - TokenIconUploadURL
// - IssueTokenIconUploadURL
// - gcsObjectPublicURL
// - backend 側での GCS object path 生成

type TokenBlueprintIconUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintIconUsecase(
	tbRepo tbdom.RepositoryPort,
) *TokenBlueprintIconUsecase {
	return &TokenBlueprintIconUsecase{
		tbRepo: tbRepo,
	}
}

// AttachTokenIconURL stores a Firebase Storage downloadURL as tokenBlueprint.iconUrl.
func (u *TokenBlueprintIconUsecase) AttachTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
	iconURL string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint icon usecase/repo is nil")
	}

	id := tokenBlueprintID
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	url := iconURL
	if url == "" {
		return nil, fmt.Errorf("iconURL is empty")
	}

	actor := actorID
	if actor == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	// ensure blueprint exists
	if _, err := u.tbRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		IconURL:   &url,
		UpdatedAt: &now,
		UpdatedBy: &actor,
	})
}

// ClearTokenIconURL clears tokenBlueprint.iconUrl.
func (u *TokenBlueprintIconUsecase) ClearTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint icon usecase/repo is nil")
	}

	id := tokenBlueprintID
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	actor := actorID
	if actor == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	// ensure blueprint exists
	if _, err := u.tbRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	empty := ""
	now := time.Now().UTC()

	return u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		IconURL:   &empty,
		UpdatedAt: &now,
		UpdatedBy: &actor,
	})
}

// GetTokenIconURL returns the stored Firebase Storage downloadURL.
func (u *TokenBlueprintIconUsecase) GetTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
) (string, error) {
	if u == nil || u.tbRepo == nil {
		return "", fmt.Errorf("tokenBlueprint icon usecase/repo is nil")
	}

	id := tokenBlueprintID
	if id == "" {
		return "", fmt.Errorf("tokenBlueprintID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if tb == nil {
		return "", tbdom.ErrNotFound
	}

	return tb.IconURL, nil
}
