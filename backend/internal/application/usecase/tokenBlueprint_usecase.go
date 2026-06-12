// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

type arweaveUploader interface {
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

type TokenBlueprintUsecase struct {
	tbRepo       tbdom.RepositoryPort
	tbReviewRepo tbReview.RepositoryPort

	metadata *tokenBlueprintMetadataUsecase
	command  *tokenBlueprintCommandUsecase
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
	uploader arweaveUploader,
) *TokenBlueprintUsecase {
	if tbRepo == nil {
		panic(fmt.Errorf("NewTokenBlueprintUsecase: tbRepo is nil"))
	}

	return &TokenBlueprintUsecase{
		tbRepo:       tbRepo,
		tbReviewRepo: tbReviewRepo,
		metadata:     newTokenBlueprintMetadataUsecase(tbRepo, uploader),
		command:      newTokenBlueprintCommandUsecase(tbRepo),
	}
}

type CreateBlueprintRequest struct {
	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string
	Description string

	IconURL         string
	IconObjectPath  string
	IconFileName    string
	IconContentType string
	IconSize        int64

	ContentFiles []tbdom.ContentFile

	AssigneeID string
	CreatedBy  string
}

func (u *TokenBlueprintUsecase) Create(
	ctx context.Context,
	in CreateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	createdBy := in.CreatedBy
	if createdBy == "" {
		return nil, tbdom.ErrInvalidCreatedBy
	}

	if err := tbdom.ValidateContentFiles(in.ContentFiles); err != nil {
		return nil, err
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:        in.Name,
		Symbol:      in.Symbol,
		BrandID:     in.BrandID,
		CompanyID:   in.CompanyID,
		Description: in.Description,

		IconURL:         in.IconURL,
		IconObjectPath:  in.IconObjectPath,
		IconFileName:    in.IconFileName,
		IconContentType: in.IconContentType,
		IconSize:        in.IconSize,

		ContentFiles: in.ContentFiles,

		AssigneeID: in.AssigneeID,

		CreatedAt: nil,
		CreatedBy: createdBy,
		UpdatedAt: nil,
		UpdatedBy: createdBy,

		MetadataURI: "",
	})
	if err != nil {
		return nil, err
	}

	if tb == nil || tb.ID == "" {
		return nil, fmt.Errorf("tokenBlueprint create returned empty id")
	}

	if u.tbReviewRepo != nil {
		aggRepo := u.tbReviewRepo.TokenBlueprintAggregates()

		now := time.Now().UTC()
		agg, aerr := tbReview.NewTokenBlueprintReviewAggregate(tb.ID, now)
		if aerr == nil && aggRepo != nil {
			_, _ = aggRepo.Create(ctx, *agg)
		}
	}

	return tb, nil
}

type UpdateBlueprintRequest struct {
	ID          string
	Name        *string
	Symbol      *string
	BrandID     *string
	Description *string
	AssigneeID  *string

	IconURL         *string
	IconObjectPath  *string
	IconFileName    *string
	IconContentType *string
	IconSize        *int64

	ContentFiles *[]tbdom.ContentFile

	MetadataURI *string
	Minted      *bool
	UpdatedBy   string
}

func (u *TokenBlueprintUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	id := in.ID
	if id == "" {
		return nil, tbdom.ErrInvalidID
	}

	updatedBy := in.UpdatedBy
	if updatedBy == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
	}

	if in.ContentFiles != nil {
		if err := tbdom.ValidateContentFiles(*in.ContentFiles); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		Name:        in.Name,
		Symbol:      in.Symbol,
		BrandID:     in.BrandID,
		Description: in.Description,
		AssigneeID:  in.AssigneeID,

		IconURL:         in.IconURL,
		IconObjectPath:  in.IconObjectPath,
		IconFileName:    in.IconFileName,
		IconContentType: in.IconContentType,
		IconSize:        in.IconSize,

		ContentFiles: in.ContentFiles,

		MetadataURI: in.MetadataURI,
		Minted:      in.Minted,

		UpdatedAt: &now,
		UpdatedBy: ptr(updatedBy),
		DeletedAt: nil,
		DeletedBy: nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	if id == "" {
		return tbdom.ErrInvalidID
	}

	return u.tbRepo.Delete(ctx, id)
}

func (u *TokenBlueprintUsecase) EnsureMetadataURI(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.metadata == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/metadata is nil")
	}

	return u.metadata.EnsureMetadataURI(ctx, tb, actorID)
}

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

type tokenBlueprintMetadataUsecase struct {
	tbRepo   tbdom.RepositoryPort
	uploader arweaveUploader
}

func newTokenBlueprintMetadataUsecase(
	tbRepo tbdom.RepositoryPort,
	uploader arweaveUploader,
) *tokenBlueprintMetadataUsecase {
	return &tokenBlueprintMetadataUsecase{
		tbRepo:   tbRepo,
		uploader: uploader,
	}
}

func (u *tokenBlueprintMetadataUsecase) EnsureMetadataURI(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint metadata usecase/repo is nil")
	}
	if u.uploader == nil {
		return nil, fmt.Errorf("tokenBlueprint metadata uploader is nil")
	}
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}
	if tb.ID == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	if tb.MetadataURI != "" {
		return tb, nil
	}

	data, err := buildTokenBlueprintMetadataJSON(tb)
	if err != nil {
		return nil, err
	}

	uri, err := u.uploader.UploadMetadata(ctx, data)
	if err != nil {
		return nil, err
	}

	if uri == "" {
		return nil, fmt.Errorf("metadataUri is empty after upload")
	}

	now := time.Now().UTC()

	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
		UpdatedAt:   &now,
		UpdatedBy:   ptr(actorID),
		DeletedAt:   nil,
		DeletedBy:   nil,
	})
	if err != nil {
		return nil, err
	}

	if updated == nil {
		tb.MetadataURI = uri
		tb.UpdatedAt = now
		tb.UpdatedBy = actorID
		return tb, nil
	}

	return updated, nil
}

func buildTokenBlueprintMetadataJSON(tb *tbdom.TokenBlueprint) ([]byte, error) {
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}

	id := tb.ID
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	name := tb.Name
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := tb.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	desc := tb.Description

	imageURL := tb.IconURL
	if imageURL == "" {
		return nil, fmt.Errorf("tokenBlueprint.iconUrl is empty")
	}

	files := make([]map[string]any, 0, 1+len(tb.ContentFiles))

	iconContentType := tb.IconContentType
	if iconContentType == "" {
		iconContentType = "image/*"
	}

	files = append(files, map[string]any{
		"uri":  imageURL,
		"type": iconContentType,
	})

	seen := make(map[string]struct{}, len(tb.ContentFiles))

	for _, f := range tb.ContentFiles {
		cid := f.ID
		if cid == "" {
			continue
		}

		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		uri := f.URL
		if uri == "" {
			return nil, fmt.Errorf("tokenBlueprint.contentFiles[%s].url is empty", cid)
		}

		objectPath := f.ObjectPath
		if objectPath == "" {
			return nil, fmt.Errorf("tokenBlueprint.contentFiles[%s].objectPath is empty", cid)
		}

		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}

		files = append(files, map[string]any{
			"uri":  uri,
			"type": ct,
		})
	}

	payload := map[string]any{
		"name":        name,
		"symbol":      symbol,
		"description": desc,
		"image":       imageURL,
		"attributes": []map[string]any{
			{
				"trait_type": "TokenBlueprintID",
				"value":      id,
			},
		},
		"properties": map[string]any{
			"category": "image",
			"files":    files,
		},
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata json: %w", err)
	}

	return b, nil
}

type tokenBlueprintCommandUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func newTokenBlueprintCommandUsecase(tbRepo tbdom.RepositoryPort) *tokenBlueprintCommandUsecase {
	return &tokenBlueprintCommandUsecase{tbRepo: tbRepo}
}

func (u *tokenBlueprintCommandUsecase) MarkTokenBlueprintMinted(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil {
		return nil, fmt.Errorf("tokenBlueprint command usecase is nil")
	}
	if u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint repo is nil")
	}

	id := tokenBlueprintID
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	if tb.Minted {
		return tb, nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	updated, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		ContentFiles: nil,
		AssigneeID:   nil,
		Description:  nil,
		Minted:       &minted,
		UpdatedAt:    &now,
		UpdatedBy:    &updatedBy,
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func ptr[T any](v T) *T {
	return &v
}
