// backend/internal/application/tokenBlueprint/tokenBlueprint_content_usecase.go
package tokenBlueprint

import (
	"context"
	"fmt"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Usecase: Content
// ============================================================
//
// Firebase Storage 移行後の責務:
// - backend は upload URL を発行しない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が getDownloadURL() で取得した downloadURL を contentFiles[].url に入れる
// - frontend が Firebase Storage object path を contentFiles[].objectPath に入れる
// - backend は contentFiles を TokenBlueprint に保存・置換する

type TokenBlueprintContentUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintContentUsecase(
	tbRepo tbdom.RepositoryPort,
) *TokenBlueprintContentUsecase {
	return &TokenBlueprintContentUsecase{
		tbRepo: tbRepo,
	}
}

// ============================================================
// Embedded contents operations
// ============================================================

// ReplaceContentFiles replaces all embedded contents.
//
// Firebase Storage 前提:
// - files[].ObjectPath は Firebase Storage object path
// - files[].URL は Firebase Storage downloadURL
// - backend は file upload / delete を行わない
func (u *TokenBlueprintContentUsecase) ReplaceContentFiles(
	ctx context.Context,
	blueprintID string,
	files []tbdom.ContentFile,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	if blueprintID == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	if err := tbdom.ValidateContentFiles(files); err != nil {
		return nil, err
	}

	validFiles := copyContentFiles(files)

	now := time.Now().UTC()

	tb, err := u.tbRepo.Update(ctx, blueprintID, tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &validFiles,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(actorID),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

// AddContentFiles appends new content files to existing embedded contents.
func (u *TokenBlueprintContentUsecase) AddContentFiles(
	ctx context.Context,
	blueprintID string,
	files []tbdom.ContentFile,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	if blueprintID == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	next := make([]tbdom.ContentFile, 0, len(tb.ContentFiles)+len(files))
	next = append(next, tb.ContentFiles...)
	next = append(next, files...)

	if err := tbdom.ValidateContentFiles(next); err != nil {
		return nil, err
	}

	validFiles := copyContentFiles(next)

	now := time.Now().UTC()

	updated, err := u.tbRepo.Update(ctx, blueprintID, tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &validFiles,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(actorID),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// RemoveContentFile removes a content file from embedded contents.
func (u *TokenBlueprintContentUsecase) RemoveContentFile(
	ctx context.Context,
	blueprintID string,
	contentID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	if blueprintID == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	if contentID == "" {
		return nil, fmt.Errorf("contentID is empty")
	}

	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	found := false
	next := make([]tbdom.ContentFile, 0, len(tb.ContentFiles))

	for _, f := range tb.ContentFiles {
		if f.ID == contentID {
			found = true
			continue
		}

		next = append(next, f)
	}

	if !found {
		return nil, tbdom.WrapNotFound(nil, "content file not found")
	}

	if err := tbdom.ValidateContentFiles(next); err != nil {
		return nil, err
	}

	validFiles := copyContentFiles(next)

	now := time.Now().UTC()

	updated, err := u.tbRepo.Update(ctx, blueprintID, tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &validFiles,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(actorID),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// SetContentVisibility updates visibility for a specific contentId.
func (u *TokenBlueprintContentUsecase) SetContentVisibility(
	ctx context.Context,
	blueprintID string,
	contentID string,
	v tbdom.ContentVisibility,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	if blueprintID == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	if contentID == "" {
		return nil, fmt.Errorf("contentID is empty")
	}

	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	now := time.Now().UTC()

	if err := tb.SetContentVisibility(contentID, v, actorID, now); err != nil {
		return nil, err
	}

	if err := tbdom.ValidateContentFiles(tb.ContentFiles); err != nil {
		return nil, err
	}

	validFiles := copyContentFiles(tb.ContentFiles)

	updated, err := u.tbRepo.Update(ctx, blueprintID, tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &validFiles,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(actorID),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// ============================================================
// internal helpers
// ============================================================

func copyContentFiles(files []tbdom.ContentFile) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	out := make([]tbdom.ContentFile, len(files))
	copy(out, files)

	return out
}
