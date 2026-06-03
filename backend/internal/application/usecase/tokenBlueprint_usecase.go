// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

type arweaveUploader interface {
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

type TokenBlueprintUsecase struct {
	crud     *tokenBlueprintCRUDUsecase
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
		crud:     newTokenBlueprintCRUDUsecase(tbRepo, tbReviewRepo),
		metadata: newTokenBlueprintMetadataUsecase(tbRepo, uploader),
		command:  newTokenBlueprintCommandUsecase(tbRepo),
	}
}

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

	if tb == nil || strings.Trim(tb.ID, " \t\r\n") == "" {
		return nil, fmt.Errorf("tokenBlueprint create returned empty id")
	}

	return tb, nil
}

func (u *TokenBlueprintUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Update(ctx, in)
}

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.crud == nil {
		return fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Delete(ctx, id)
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

type tokenBlueprintCRUDUsecase struct {
	tbRepo       tbdom.RepositoryPort
	tbReviewRepo tbReview.RepositoryPort
}

func newTokenBlueprintCRUDUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
) *tokenBlueprintCRUDUsecase {
	return &tokenBlueprintCRUDUsecase{
		tbRepo:       tbRepo,
		tbReviewRepo: tbReviewRepo,
	}
}

type CreateBlueprintRequest struct {
	Name         string
	Symbol       string
	BrandID      string
	CompanyID    string
	Description  string
	IconURL      string
	ContentFiles []tbdom.ContentFile
	AssigneeID   string
	CreatedBy    string
}

func (u *tokenBlueprintCRUDUsecase) Create(
	ctx context.Context,
	in CreateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	createdBy := strings.Trim(in.CreatedBy, " \t\r\n")
	if createdBy == "" {
		return nil, tbdom.ErrInvalidCreatedBy
	}

	if err := tbdom.ValidateContentFiles(in.ContentFiles); err != nil {
		return nil, err
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         strings.Trim(in.Name, " \t\r\n"),
		Symbol:       strings.Trim(in.Symbol, " \t\r\n"),
		BrandID:      strings.Trim(in.BrandID, " \t\r\n"),
		CompanyID:    strings.Trim(in.CompanyID, " \t\r\n"),
		Description:  strings.Trim(in.Description, " \t\r\n"),
		IconURL:      strings.Trim(in.IconURL, " \t\r\n"),
		ContentFiles: in.ContentFiles,
		AssigneeID:   strings.Trim(in.AssigneeID, " \t\r\n"),
		CreatedAt:    nil,
		CreatedBy:    createdBy,
		UpdatedAt:    nil,
		UpdatedBy:    createdBy,
		MetadataURI:  "",
	})
	if err != nil {
		return nil, err
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
	ID           string
	Name         *string
	Symbol       *string
	BrandID      *string
	Description  *string
	AssigneeID   *string
	IconURL      *string
	ContentFiles *[]tbdom.ContentFile
	MetadataURI  *string
	Minted       *bool
	UpdatedBy    string
}

func (u *tokenBlueprintCRUDUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	id := strings.Trim(in.ID, " \t\r\n")
	if id == "" {
		return nil, tbdom.ErrInvalidID
	}

	updatedBy := strings.Trim(in.UpdatedBy, " \t\r\n")
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
		Name:         in.Name,
		Symbol:       in.Symbol,
		BrandID:      in.BrandID,
		Description:  in.Description,
		AssigneeID:   in.AssigneeID,
		IconURL:      in.IconURL,
		ContentFiles: in.ContentFiles,
		MetadataURI:  in.MetadataURI,
		Minted:       in.Minted,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(updatedBy),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

func (u *tokenBlueprintCRUDUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	id = strings.Trim(id, " \t\r\n")
	if id == "" {
		return tbdom.ErrInvalidID
	}

	return u.tbRepo.Delete(ctx, id)
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
	if strings.Trim(tb.ID, " \t\r\n") == "" {
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

	uri = strings.Trim(uri, " \t\r\n")
	if uri == "" {
		return nil, fmt.Errorf("metadataUri is empty after upload")
	}

	now := time.Now().UTC()
	actorID = strings.Trim(actorID, " \t\r\n")

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

	id := strings.Trim(tb.ID, " \t\r\n")
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	name := strings.Trim(tb.Name, " \t\r\n")
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := strings.Trim(tb.Symbol, " \t\r\n")
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	desc := strings.Trim(tb.Description, " \t\r\n")

	imageURL := strings.Trim(tb.IconURL, " \t\r\n")
	if imageURL == "" {
		return nil, fmt.Errorf("tokenBlueprint.iconUrl is empty")
	}

	files := make([]map[string]any, 0, 1+len(tb.ContentFiles))

	files = append(files, map[string]any{
		"uri":  imageURL,
		"type": "image/*",
	})

	seen := make(map[string]struct{}, len(tb.ContentFiles))

	for _, f := range tb.ContentFiles {
		cid := strings.Trim(f.ID, " \t\r\n")
		if cid == "" {
			continue
		}

		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		uri := strings.Trim(f.URL, " \t\r\n")
		if uri == "" {
			return nil, fmt.Errorf("tokenBlueprint.contentFiles[%s].url is empty", cid)
		}

		ct := strings.Trim(f.ContentType, " \t\r\n")
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

	id := strings.Trim(tokenBlueprintID, " \t\r\n")
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	actorID = strings.Trim(actorID, " \t\r\n")
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
