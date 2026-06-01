// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
	"narratives/internal/infra/arweave"
)

type ArweaveUploader interface {
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

type TokenBlueprintUsecase struct {
	crud     *TokenBlueprintCRUDUsecase
	command  *TokenBlueprintCommandUsecase
	metadata *TokenBlueprintMetadataUsecase
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
) *TokenBlueprintUsecase {
	if tbRepo == nil {
		panic(fmt.Errorf("NewTokenBlueprintUsecase: tbRepo is nil"))
	}

	crud := NewTokenBlueprintCRUDUsecase(tbRepo, tbReviewRepo)
	command := NewTokenBlueprintCommandUsecase(tbRepo)

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	metadata := NewTokenBlueprintMetadataUsecase(tbRepo, uploader)

	return &TokenBlueprintUsecase{
		crud:     crud,
		command:  command,
		metadata: metadata,
	}
}

type TokenBlueprintCRUDUsecase struct {
	tbRepo       tbdom.RepositoryPort
	tbReviewRepo tbReview.RepositoryPort
}

func NewTokenBlueprintCRUDUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
) *TokenBlueprintCRUDUsecase {
	return &TokenBlueprintCRUDUsecase{
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

func (u *TokenBlueprintCRUDUsecase) Create(
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

	contentFiles := normalizeContentFilesForCommand(in.ContentFiles, createdBy)
	if err := tbdom.ValidateContentFiles(contentFiles); err != nil {
		return nil, err
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         strings.Trim(in.Name, " \t\r\n"),
		Symbol:       strings.Trim(in.Symbol, " \t\r\n"),
		BrandID:      strings.Trim(in.BrandID, " \t\r\n"),
		CompanyID:    strings.Trim(in.CompanyID, " \t\r\n"),
		Description:  strings.Trim(in.Description, " \t\r\n"),
		IconURL:      strings.Trim(in.IconURL, " \t\r\n"),
		ContentFiles: contentFiles,
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

func (u *TokenBlueprintCRUDUsecase) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	id = strings.Trim(id, " \t\r\n")
	if id == "" {
		return nil, tbdom.ErrInvalidID
	}

	return u.tbRepo.GetByID(ctx, id)
}

func (u *TokenBlueprintCRUDUsecase) GetByIDForCompany(
	ctx context.Context,
	id string,
	companyID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	companyID = strings.Trim(companyID, " \t\r\n")
	if companyID == "" {
		return nil, tbdom.ErrInvalidCompanyID
	}

	tb, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	if strings.Trim(tb.CompanyID, " \t\r\n") != companyID {
		return nil, tbdom.ErrNotFound
	}

	return tb, nil
}

func (u *TokenBlueprintCRUDUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	var empty domcommon.PageResult[tbdom.TokenBlueprint]

	if u == nil || u.tbRepo == nil {
		return empty, tbdom.ErrInvalid
	}

	companyID = strings.Trim(companyID, " \t\r\n")
	if companyID == "" {
		return empty, tbdom.ErrInvalidCompanyID
	}

	return u.tbRepo.ListByCompanyID(ctx, companyID, page)
}

func (u *TokenBlueprintCRUDUsecase) ListByBrandID(
	ctx context.Context,
	brandID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	var empty domcommon.PageResult[tbdom.TokenBlueprint]

	if u == nil || u.tbRepo == nil {
		return empty, tbdom.ErrInvalid
	}

	brandID = strings.Trim(brandID, " \t\r\n")
	if brandID == "" {
		return empty, tbdom.ErrInvalidBrandID
	}

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
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

func (u *TokenBlueprintCRUDUsecase) Update(
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

	var contentFiles *[]tbdom.ContentFile
	if in.ContentFiles != nil {
		normalized := normalizeContentFilesForCommand(*in.ContentFiles, updatedBy)
		if err := tbdom.ValidateContentFiles(normalized); err != nil {
			return nil, err
		}

		contentFiles = &normalized
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		Name:         trimStringPtr(in.Name),
		Symbol:       trimStringPtr(in.Symbol),
		BrandID:      trimStringPtr(in.BrandID),
		Description:  trimStringPtr(in.Description),
		AssigneeID:   trimStringPtr(in.AssigneeID),
		IconURL:      trimStringPtr(in.IconURL),
		ContentFiles: contentFiles,
		MetadataURI:  trimStringPtr(in.MetadataURI),
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

func (u *TokenBlueprintCRUDUsecase) UpdateForCompany(
	ctx context.Context,
	companyID string,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	if _, err := u.GetByIDForCompany(ctx, in.ID, companyID); err != nil {
		return nil, err
	}

	return u.Update(ctx, in)
}

func (u *TokenBlueprintCRUDUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	id = strings.Trim(id, " \t\r\n")
	if id == "" {
		return tbdom.ErrInvalidID
	}

	return u.tbRepo.Delete(ctx, id)
}

func (u *TokenBlueprintCRUDUsecase) DeleteForCompany(
	ctx context.Context,
	companyID string,
	id string,
) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	if _, err := u.GetByIDForCompany(ctx, id, companyID); err != nil {
		return err
	}

	return u.Delete(ctx, id)
}

type TokenBlueprintCommandUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintCommandUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintCommandUsecase {
	return &TokenBlueprintCommandUsecase{tbRepo: tbRepo}
}

func (u *TokenBlueprintCommandUsecase) MarkTokenBlueprintMinted(
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

type TokenBlueprintMetadataUsecase struct {
	tbRepo   tbdom.RepositoryPort
	uploader ArweaveUploader
}

func NewTokenBlueprintMetadataUsecase(
	tbRepo tbdom.RepositoryPort,
	uploader ArweaveUploader,
) *TokenBlueprintMetadataUsecase {
	return &TokenBlueprintMetadataUsecase{
		tbRepo:   tbRepo,
		uploader: uploader,
	}
}

func (u *TokenBlueprintMetadataUsecase) EnsureMetadataURI(
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

func (u *TokenBlueprintUsecase) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.GetByID(ctx, id)
}

func (u *TokenBlueprintUsecase) GetByIDForCompany(
	ctx context.Context,
	id string,
	companyID string,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.GetByIDForCompany(ctx, id, companyID)
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

func (u *TokenBlueprintUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Update(ctx, in)
}

func (u *TokenBlueprintUsecase) UpdateForCompany(
	ctx context.Context,
	companyID string,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.crud == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.UpdateForCompany(ctx, companyID, in)
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

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.crud == nil {
		return fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.Delete(ctx, id)
}

func (u *TokenBlueprintUsecase) DeleteForCompany(
	ctx context.Context,
	companyID string,
	id string,
) error {
	if u == nil || u.crud == nil {
		return fmt.Errorf("tokenBlueprint usecase/crud is nil")
	}

	return u.crud.DeleteForCompany(ctx, companyID, id)
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

func normalizeContentFilesForCommand(files []tbdom.ContentFile, actorID string) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	actorID = strings.Trim(actorID, " \t\r\n")
	now := time.Now().UTC()
	out := make([]tbdom.ContentFile, 0, len(files))

	for _, f := range files {
		f.ID = strings.Trim(f.ID, " \t\r\n")
		f.Type = tbdom.ContentFileType(strings.Trim(string(f.Type), " \t\r\n"))
		f.ContentType = strings.Trim(f.ContentType, " \t\r\n")
		f.URL = strings.Trim(f.URL, " \t\r\n")
		f.Visibility = tbdom.ContentVisibility(strings.Trim(string(f.Visibility), " \t\r\n"))

		if f.ContentType == "" {
			f.ContentType = "application/octet-stream"
		}

		if f.Visibility == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}

		if f.CreatedAt.IsZero() {
			f.CreatedAt = now
		}

		if strings.Trim(f.CreatedBy, " \t\r\n") == "" {
			f.CreatedBy = actorID
		}

		if f.UpdatedAt.IsZero() {
			f.UpdatedAt = now
		}

		if strings.Trim(f.UpdatedBy, " \t\r\n") == "" {
			f.UpdatedBy = actorID
		}

		if f.ID == "" || f.URL == "" {
			continue
		}

		out = append(out, f)
	}

	return out
}

func trimStringPtr(v *string) *string {
	if v == nil {
		return nil
	}

	x := strings.Trim(*v, " \t\r\n")
	return &x
}

func ptr[T any](v T) *T {
	return &v
}
