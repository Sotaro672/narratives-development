// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
	tcdom "narratives/internal/domain/tokenContents"
	tidom "narratives/internal/domain/tokenIcon"
)

// TokenBlueprintUsecase coordinates TokenBlueprint, TokenContents, and TokenIcon domains.
type TokenBlueprintUsecase struct {
	tbRepo tbdom.RepositoryPort
	tcRepo tcdom.RepositoryPort
	tiRepo tidom.RepositoryPort
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tcRepo tcdom.RepositoryPort,
	tiRepo tidom.RepositoryPort,
) *TokenBlueprintUsecase {
	return &TokenBlueprintUsecase{
		tbRepo: tbRepo,
		tcRepo: tcRepo,
		tiRepo: tiRepo,
	}
}

// Upload DTOs

type IconUpload struct {
	FileName    string
	ContentType string
	Reader      io.Reader
}

type ContentUpload struct {
	Name        string
	Type        tcdom.ContentType
	FileName    string
	ContentType string
	Reader      io.Reader
}

// Create

type CreateBlueprintRequest struct {
	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string // ★ 追加: テナント
	Description string

	AssigneeID string
	ActorID    string // use for CreatedBy / UpdatedBy

	Icon     *IconUpload
	Contents []ContentUpload
}

func (u *TokenBlueprintUsecase) CreateWithUploads(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	var iconIDPtr *string
	if in.Icon != nil {
		iconURL, size, err := u.tiRepo.UploadIcon(ctx, in.Icon.FileName, in.Icon.ContentType, in.Icon.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload icon: %w", err)
		}
		icon, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
			URL:      strings.TrimSpace(iconURL),
			FileName: strings.TrimSpace(in.Icon.FileName),
			Size:     size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token icon: %w", err)
		}
		iconID := strings.TrimSpace(icon.ID)
		if iconID != "" {
			iconIDPtr = &iconID
		}
	}

	contentIDs := make([]string, 0, len(in.Contents))
	for _, c := range in.Contents {
		url, size, err := u.tcRepo.UploadContent(ctx, c.FileName, c.ContentType, c.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload content(%s): %w", c.FileName, err)
		}
		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(c.Name),
			Type: c.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token content(%s): %w", c.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			contentIDs = append(contentIDs, id)
		}
	}
	contentIDs = dedupStrings(contentIDs)

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         strings.TrimSpace(in.Name),
		Symbol:       strings.TrimSpace(in.Symbol),
		BrandID:      strings.TrimSpace(in.BrandID),
		CompanyID:    strings.TrimSpace(in.CompanyID), // ★ 追加
		Description:  strings.TrimSpace(in.Description),
		IconID:       iconIDPtr,
		ContentFiles: contentIDs,
		AssigneeID:   strings.TrimSpace(in.AssigneeID),

		CreatedAt: nil,
		CreatedBy: strings.TrimSpace(in.ActorID),
		UpdatedAt: nil,
		UpdatedBy: strings.TrimSpace(in.ActorID),
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

// Read

func (u *TokenBlueprintUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	return u.tbRepo.GetByID(ctx, strings.TrimSpace(id))
}

// sort を廃止し、List からも除去
func (u *TokenBlueprintUsecase) List(ctx context.Context, filter tbdom.Filter, page tbdom.Page) (tbdom.PageResult, error) {
	return u.tbRepo.List(ctx, filter, page)
}

// ★ 追加: currentMember の companyId を指定して一覧取得するユースケース
func (u *TokenBlueprintUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return tbdom.PageResult{}, fmt.Errorf("companyId is empty")
	}

	filter := tbdom.Filter{
		CompanyIDs: []string{cid},
	}

	return u.tbRepo.List(ctx, filter, page)
}

// Update

type UpdateBlueprintRequest struct {
	ID           string
	Name         *string
	Symbol       *string
	BrandID      *string
	Description  *string
	AssigneeID   *string
	IconID       *string   // set empty string "" to clear
	ContentFiles *[]string // full replacement list (IDs)
	ActorID      string
}

func (u *TokenBlueprintUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	return u.tbRepo.Update(ctx, strings.TrimSpace(in.ID), tbdom.UpdateTokenBlueprintInput{
		Name:         trimPtr(in.Name),
		Symbol:       trimPtr(in.Symbol),
		BrandID:      trimPtr(in.BrandID),
		Description:  trimPtr(in.Description),
		IconID:       normalizeEmptyToNil(in.IconID),
		ContentFiles: normalizeSlicePtr(in.ContentFiles),
		AssigneeID:   trimPtr(in.AssigneeID),

		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(in.ActorID)),
		DeletedAt: nil,
		DeletedBy: nil,
	})
}

// Convenient helpers

// ReplaceIconWithUpload uploads a new icon, creates TokenIcon, and sets IconID on the blueprint.
func (u *TokenBlueprintUsecase) ReplaceIconWithUpload(ctx context.Context, blueprintID string, icon IconUpload, actorID string) (*tbdom.TokenBlueprint, error) {
	url, size, err := u.tiRepo.UploadIcon(ctx, icon.FileName, icon.ContentType, icon.Reader)
	if err != nil {
		return nil, fmt.Errorf("upload icon: %w", err)
	}
	ti, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
		URL:      strings.TrimSpace(url),
		FileName: strings.TrimSpace(icon.FileName),
		Size:     size,
	})
	if err != nil {
		return nil, fmt.Errorf("create token icon: %w", err)
	}
	iconID := strings.TrimSpace(ti.ID)
	return u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &iconID,
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
}

// AddContentsWithUploads uploads and creates contents, then appends their IDs to ContentFiles.
func (u *TokenBlueprintUsecase) AddContentsWithUploads(ctx context.Context, blueprintID string, uploads []ContentUpload, actorID string) (*tbdom.TokenBlueprint, error) {
	if len(uploads) == 0 {
		return u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	}

	ids := make([]string, 0, len(uploads))
	for _, up := range uploads {
		url, size, err := u.tcRepo.UploadContent(ctx, up.FileName, up.ContentType, up.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload content(%s): %w", up.FileName, err)
		}
		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(up.Name),
			Type: up.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token content(%s): %w", up.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			ids = append(ids, id)
		}
	}

	// Fetch current blueprint to merge
	current, err := u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	if err != nil {
		return nil, err
	}
	merged := append([]string{}, current.ContentFiles...)
	merged = append(merged, ids...)
	merged = dedupStrings(merged)

	return u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &merged,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
	})
}

// ClearIcon clears IconID.
func (u *TokenBlueprintUsecase) ClearIcon(ctx context.Context, blueprintID string, actorID string) (*tbdom.TokenBlueprint, error) {
	empty := ""
	return u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &empty, // normalizeEmptyToNil will set NULL
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
}

// RemoveContentIDs replaces ContentFiles with given list (caller computes new list).
func (u *TokenBlueprintUsecase) ReplaceContentIDs(ctx context.Context, blueprintID string, contentIDs []string, actorID string) (*tbdom.TokenBlueprint, error) {
	clean := dedupStrings(contentIDs)
	return u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &clean,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
	})
}

// Delete

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.tbRepo.Delete(ctx, strings.TrimSpace(id))
}

// Helpers は common_usecase.go に移動しました（trimPtr を使用してください）。
