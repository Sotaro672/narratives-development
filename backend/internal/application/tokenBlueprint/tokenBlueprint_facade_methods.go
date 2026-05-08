// backend/internal/application/tokenBlueprint/tokenBlueprint_facade_methods.go
package tokenBlueprint

import (
	"context"
	"fmt"

	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// =========================
// Create
// =========================
//
// Firebase Storage 移行後:
// - backend は GCS signed URL を発行しない
// - backend は GCS bucket / .keep object を作成しない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が取得した downloadURL を iconUrl / contentFiles[].url として渡す
// - backend は Firestore に保存するだけ

func (u *TokenBlueprintUsecase) Create(
	ctx context.Context,
	in CreateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	tb, err := u.crud.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	if tb == nil || tb.ID == "" {
		return nil, fmt.Errorf("tokenBlueprint create returned empty id")
	}

	return tb, nil
}

// =========================
// Read
// =========================

func (u *TokenBlueprintUsecase) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.GetByID(ctx, id)
}

func (u *TokenBlueprintUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	if u == nil || u.query == nil {
		return nil, "", fmt.Errorf("tokenBlueprint usecase/query is nil")
	}

	return u.query.GetByIDWithCreatorName(ctx, id)
}

func (u *TokenBlueprintUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if u == nil || u.crud == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.ListByCompanyID(ctx, companyID, page)
}

func (u *TokenBlueprintUsecase) ListByBrandID(
	ctx context.Context,
	brandID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if u == nil || u.crud == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.ListByBrandID(ctx, brandID, page)
}

func (u *TokenBlueprintUsecase) ListMintedCompleted(
	ctx context.Context,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if u == nil || u.crud == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.ListMintedCompleted(ctx, page)
}

func (u *TokenBlueprintUsecase) ResolveNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if u == nil || u.query == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/query is nil")
	}

	return u.query.ResolveNames(ctx, ids)
}

// =========================
// Update
// =========================
//
// Firebase Storage 移行後:
// - icon 変更時は frontend が Firebase Storage に upload 後、iconUrl を渡す
// - contents 変更時は frontend が Firebase Storage に upload 後、contentFiles を渡す

func (u *TokenBlueprintUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Update(ctx, in)
}

// =========================
// Convenience helpers (contents)
// =========================

func (u *TokenBlueprintUsecase) ReplaceContentFiles(
	ctx context.Context,
	blueprintID string,
	files []tbdom.ContentFile,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.content == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/content is nil")
	}

	return u.content.ReplaceContentFiles(ctx, blueprintID, files, actorID)
}

func (u *TokenBlueprintUsecase) AddContentFiles(
	ctx context.Context,
	blueprintID string,
	files []tbdom.ContentFile,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.content == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/content is nil")
	}

	return u.content.AddContentFiles(ctx, blueprintID, files, actorID)
}

func (u *TokenBlueprintUsecase) RemoveContentFile(
	ctx context.Context,
	blueprintID string,
	contentID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.content == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/content is nil")
	}

	return u.content.RemoveContentFile(ctx, blueprintID, contentID, actorID)
}

func (u *TokenBlueprintUsecase) SetContentVisibility(
	ctx context.Context,
	blueprintID string,
	contentID string,
	v tbdom.ContentVisibility,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.content == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/content is nil")
	}

	return u.content.SetContentVisibility(ctx, blueprintID, contentID, v, actorID)
}

// =========================
// Icon helpers
// =========================

func (u *TokenBlueprintUsecase) AttachTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
	iconURL string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.icon == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/icon is nil")
	}

	return u.icon.AttachTokenIconURL(ctx, tokenBlueprintID, iconURL, actorID)
}

func (u *TokenBlueprintUsecase) ClearTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.icon == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/icon is nil")
	}

	return u.icon.ClearTokenIconURL(ctx, tokenBlueprintID, actorID)
}

func (u *TokenBlueprintUsecase) GetTokenIconURL(
	ctx context.Context,
	tokenBlueprintID string,
) (string, error) {
	if u == nil || u.icon == nil {
		return "", fmt.Errorf("tokenBlueprint usecase/icon is nil")
	}

	return u.icon.GetTokenIconURL(ctx, tokenBlueprintID)
}

// =========================
// Delete
// =========================

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.crud == nil {
		return fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Delete(ctx, id)
}

// =========================
// Additional API: minted
// =========================

func (u *TokenBlueprintUsecase) MarkTokenBlueprintMinted(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.command == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/command is nil")
	}

	return u.command.MarkTokenBlueprintMinted(ctx, tokenBlueprintID, actorID)
}
