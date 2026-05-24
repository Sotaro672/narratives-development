package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"narratives/internal/application/resolver"
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
	icon     *TokenBlueprintIconUsecase
	query    *TokenBlueprintQueryUsecase
	content  *TokenBlueprintContentUsecase
	command  *TokenBlueprintCommandUsecase
	metadata *TokenBlueprintMetadataUsecase
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
	nameResolver *resolver.NameResolver,
) *TokenBlueprintUsecase {
	if tbRepo == nil {
		panic(fmt.Errorf("NewTokenBlueprintUsecase: tbRepo is nil"))
	}

	crud := NewTokenBlueprintCRUDUsecase(tbRepo, tbReviewRepo)
	icon := NewTokenBlueprintIconUsecase(tbRepo)
	query := NewTokenBlueprintQueryUsecase(tbRepo, nameResolver)
	content := NewTokenBlueprintContentUsecase(tbRepo)
	command := NewTokenBlueprintCommandUsecase(tbRepo)

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	metadata := NewTokenBlueprintMetadataUsecase(tbRepo, uploader)

	return &TokenBlueprintUsecase{
		crud:     crud,
		icon:     icon,
		query:    query,
		content:  content,
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
	ActorID      string
}

func (u *TokenBlueprintCRUDUsecase) Create(
	ctx context.Context,
	in CreateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	if in.CreatedBy == "" {
		return nil, tbdom.ErrInvalidCreatedBy
	}

	if err := tbdom.ValidateContentFiles(in.ContentFiles); err != nil {
		return nil, err
	}

	contentFiles := copyContentFiles(in.ContentFiles)

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         in.Name,
		Symbol:       in.Symbol,
		BrandID:      in.BrandID,
		CompanyID:    in.CompanyID,
		Description:  in.Description,
		IconURL:      in.IconURL,
		ContentFiles: contentFiles,
		AssigneeID:   in.AssigneeID,
		CreatedAt:    nil,
		CreatedBy:    in.CreatedBy,
		UpdatedAt:    nil,
		UpdatedBy:    in.CreatedBy,
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

	if id == "" {
		return nil, tbdom.ErrInvalidID
	}

	return u.tbRepo.GetByID(ctx, id)
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

	if brandID == "" {
		return empty, tbdom.ErrInvalidBrandID
	}

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}

func (u *TokenBlueprintCRUDUsecase) ListMintedCompleted(
	ctx context.Context,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	var empty domcommon.PageResult[tbdom.TokenBlueprint]

	if u == nil || u.tbRepo == nil {
		return empty, tbdom.ErrInvalid
	}

	return tbdom.ListMintedCompleted(ctx, u.tbRepo, page)
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
	ActorID      string
}

func (u *TokenBlueprintCRUDUsecase) Update(
	ctx context.Context,
	in UpdateBlueprintRequest,
) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, tbdom.ErrInvalid
	}

	if in.ID == "" {
		return nil, tbdom.ErrInvalidID
	}

	if in.ActorID == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
	}

	var contentFiles *[]tbdom.ContentFile
	if in.ContentFiles != nil {
		if err := tbdom.ValidateContentFiles(*in.ContentFiles); err != nil {
			return nil, err
		}

		copied := copyContentFiles(*in.ContentFiles)
		contentFiles = &copied
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.Update(ctx, in.ID, tbdom.UpdateTokenBlueprintInput{
		Name:         in.Name,
		Symbol:       in.Symbol,
		BrandID:      in.BrandID,
		Description:  in.Description,
		AssigneeID:   in.AssigneeID,
		IconURL:      in.IconURL,
		ContentFiles: contentFiles,
		MetadataURI:  in.MetadataURI,
		Minted:       in.Minted,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(in.ActorID),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

func (u *TokenBlueprintCRUDUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	if id == "" {
		return tbdom.ErrInvalidID
	}

	return u.tbRepo.Delete(ctx, id)
}

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

type TokenBlueprintQueryUsecase struct {
	tbRepo       tbdom.RepositoryPort
	nameResolver *resolver.NameResolver
}

func NewTokenBlueprintQueryUsecase(
	tbRepo tbdom.RepositoryPort,
	nameResolver *resolver.NameResolver,
) *TokenBlueprintQueryUsecase {
	return &TokenBlueprintQueryUsecase{
		tbRepo:       tbRepo,
		nameResolver: nameResolver,
	}
}

type TokenBlueprintMemberNames struct {
	AssigneeName  string `json:"assigneeName"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
}

func (u *TokenBlueprintQueryUsecase) ResolveMemberNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	out := make(map[string]string, len(ids))

	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if u.nameResolver == nil {
		for _, mid := range uniq {
			out[mid] = ""
		}
		return out, nil
	}

	for _, mid := range uniq {
		out[mid] = u.nameResolver.ResolveMemberName(ctx, mid)
	}

	return out, nil
}

func (u *TokenBlueprintQueryUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	if u == nil || u.tbRepo == nil {
		return nil, "", fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	tid := id

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, "", err
	}
	if tb == nil {
		return nil, "", tbdom.ErrNotFound
	}

	memberID := tb.CreatedBy
	if memberID == "" || u.nameResolver == nil {
		return tb, "", nil
	}

	return tb, u.nameResolver.ResolveMemberName(ctx, memberID), nil
}

func (u *TokenBlueprintQueryUsecase) GetByIDWithMemberNames(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, TokenBlueprintMemberNames, error) {
	if u == nil || u.tbRepo == nil {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("tokenBlueprint query usecase/repo is nil")
	}

	tid := id
	if tid == "" {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("id is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, TokenBlueprintMemberNames{}, err
	}
	if tb == nil {
		return nil, TokenBlueprintMemberNames{}, tbdom.ErrNotFound
	}

	ids := []string{
		tb.AssigneeID,
		tb.CreatedBy,
		tb.UpdatedBy,
	}

	m, _ := u.ResolveMemberNames(ctx, ids)

	return tb, TokenBlueprintMemberNames{
		AssigneeName:  m[tb.AssigneeID],
		CreatedByName: m[tb.CreatedBy],
		UpdatedByName: m[tb.UpdatedBy],
	}, nil
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
		if id == "" {
			continue
		}

		name, err := u.tbRepo.GetNameByID(ctx, id)
		if err != nil {
			result[id] = ""
			continue
		}

		result[id] = name
	}

	return result, nil
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
	if strings.TrimSpace(tb.ID) == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	if strings.TrimSpace(tb.MetadataURI) != "" {
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

	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, fmt.Errorf("metadataUri is empty after upload")
	}

	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
		UpdatedAt:   nil,
		UpdatedBy:   ptr(actorID),
		DeletedAt:   nil,
		DeletedBy:   nil,
	})
	if err != nil {
		return nil, err
	}

	if updated == nil {
		tb.MetadataURI = uri
		return tb, nil
	}

	return updated, nil
}

func buildTokenBlueprintMetadataJSON(tb *tbdom.TokenBlueprint) ([]byte, error) {
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}

	id := strings.TrimSpace(tb.ID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	name := strings.TrimSpace(tb.Name)
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := strings.TrimSpace(tb.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	desc := strings.TrimSpace(tb.Description)

	imageURL := strings.TrimSpace(tb.IconURL)
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
		cid := strings.TrimSpace(f.ID)
		if cid == "" {
			continue
		}

		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		uri := strings.TrimSpace(f.URL)
		if uri == "" {
			return nil, fmt.Errorf("tokenBlueprint.contentFiles[%s].url is empty", cid)
		}

		ct := strings.TrimSpace(f.ContentType)
		if ct == "" {
			ct = "application/octet-stream"
		}

		file := map[string]any{
			"uri":  uri,
			"type": ct,
		}

		if name := strings.TrimSpace(f.Name); name != "" {
			file["name"] = name
		}

		files = append(files, file)
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

	if tb == nil || tb.ID == "" {
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

func (u *TokenBlueprintUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	if u == nil || u.query == nil {
		return nil, "", fmt.Errorf("tokenBlueprint usecase/query is nil")
	}

	return u.query.GetByIDWithCreatorName(ctx, id)
}

func (u *TokenBlueprintUsecase) GetByIDWithMemberNames(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, TokenBlueprintMemberNames, error) {
	if u == nil || u.query == nil {
		return nil, TokenBlueprintMemberNames{}, fmt.Errorf("tokenBlueprint usecase/query is nil")
	}

	return u.query.GetByIDWithMemberNames(ctx, id)
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

func (u *TokenBlueprintUsecase) ResolveMemberNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if u == nil || u.query == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/query is nil")
	}

	return u.query.ResolveMemberNames(ctx, ids)
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

func copyContentFiles(files []tbdom.ContentFile) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	out := make([]tbdom.ContentFile, len(files))
	copy(out, files)

	return out
}

func ptr[T any](v T) *T {
	return &v
}
