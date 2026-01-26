// backend/internal/application/usecase/tokenBlueprint_facade_methods.go
package tokenBlueprint

import (
	"context"
	"fmt"
	"log"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// =========================
// Signed URL : token_icon
// =========================

func (u *TokenBlueprintUsecase) IssueTokenIconUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	fileName string,
	contentType string,
) (*TokenIconUploadURL, error) {
	if u == nil || u.icon == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/icon usecase is nil")
	}
	return u.icon.IssueTokenIconUploadURL(ctx, tokenBlueprintID, fileName, contentType)
}

// =========================
// Signed URL : token_contents
// =========================

// IssueTokenContentUploadURL is a facade method used by handlers.
// It delegates to TokenBlueprintContentUsecase.IssueTokenContentsUploadURL.
func (u *TokenBlueprintUsecase) IssueTokenContentUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	fileName string,
	contentType string,
) (*TokenContentsUploadURL, error) {
	if u == nil || u.content == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/content usecase is nil")
	}
	return u.content.IssueTokenContentsUploadURL(ctx, tokenBlueprintID, fileName, contentType)
}

// =========================
// Create
// =========================

func (u *TokenBlueprintUsecase) Create(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	// --- guard ---
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}
	if u.buckets == nil {
		return nil, fmt.Errorf("tokenBlueprint buckets usecase is nil")
	}

	// --- Firestore create (need ID for "{id}/.keep") ---
	tb, err := u.crud.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	if tb == nil || tb.ID == "" {
		return nil, fmt.Errorf("tokenBlueprint create returned empty id")
	}

	// --- 必須: 起票後に .keep を保証（icon/contents 両方） ---
	log.Printf(
		"[TokenBlueprintBucket] ensure keep start id=%q (iconBucket=%q contentsBucket=%q)",
		tb.ID,
		tokenIconBucketName(),
		tokenContentsBucketName(),
	)

	if err := u.buckets.EnsureKeepObjects(ctx, tb.ID); err != nil {
		log.Printf("[TokenBlueprintBucket] ensure keep ERROR id=%q: %v", tb.ID, err)
		return nil, err
	}

	log.Printf("[TokenBlueprintBucket] ensure keep success id=%q", tb.ID)

	// Backward compatible behavior: fill metadataUri if empty (policy component).
	if u.metadata != nil {
		updated, uerr := u.metadata.EnsureMetadataURI(ctx, tb, in.ActorID)
		if uerr == nil && updated != nil {
			return updated, nil
		}
		// EnsureMetadataURI failure should not break create result in current policy.
	}

	return tb, nil
}

// =========================
// Read
// =========================

func (u *TokenBlueprintUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	return u.crud.GetByID(ctx, id)
}

func (u *TokenBlueprintUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	return u.query.GetByIDWithCreatorName(ctx, id)
}

func (u *TokenBlueprintUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	return u.crud.ListByCompanyID(ctx, companyID, page)
}

func (u *TokenBlueprintUsecase) ListByBrandID(ctx context.Context, brandID string, page tbdom.Page) (tbdom.PageResult, error) {
	return u.crud.ListByBrandID(ctx, brandID, page)
}

func (u *TokenBlueprintUsecase) ListMintedNotYet(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return u.crud.ListMintedNotYet(ctx, page)
}

func (u *TokenBlueprintUsecase) ListMintedCompleted(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return u.crud.ListMintedCompleted(ctx, page)
}

func (u *TokenBlueprintUsecase) ResolveNames(ctx context.Context, ids []string) (map[string]string, error) {
	return u.query.ResolveNames(ctx, ids)
}

// =========================
// Update
// =========================

func (u *TokenBlueprintUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	return u.crud.Update(ctx, in)
}

// =========================
// Convenience helpers (contents)
// =========================

func (u *TokenBlueprintUsecase) ReplaceContentFiles(ctx context.Context, blueprintID string, files []tbdom.ContentFile, actorID string) (*tbdom.TokenBlueprint, error) {
	return u.content.ReplaceContentFiles(ctx, blueprintID, files, actorID)
}

func (u *TokenBlueprintUsecase) SetContentVisibility(ctx context.Context, blueprintID string, contentID string, v tbdom.ContentVisibility, actorID string) (*tbdom.TokenBlueprint, error) {
	return u.content.SetContentVisibility(ctx, blueprintID, contentID, v, actorID)
}

// =========================
// Delete
// =========================

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.crud.Delete(ctx, id)
}

// =========================
// Additional API: minted
// =========================

func (u *TokenBlueprintUsecase) MarkTokenBlueprintMinted(ctx context.Context, tokenBlueprintID string, actorID string) (*tbdom.TokenBlueprint, error) {
	return u.command.MarkTokenBlueprintMinted(ctx, tokenBlueprintID, actorID)
}
