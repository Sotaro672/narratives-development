// backend/internal/application/tokenBlueprint/tokenBlueprint_crud_usecase.go
package tokenBlueprint

import (
	"context"
	"time"

	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

// TokenBlueprintCRUDUsecase focuses on persistence CRUD only.
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

// ============================================================
// Create
// ============================================================

type CreateBlueprintRequest struct {
	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string
	Description string

	// Firebase Storage downloadURL.
	// frontend が Firebase Storage へ直接 upload し、getDownloadURL() の結果を渡す。
	IconURL string

	// Firebase Storage objectPath / downloadURL を含む embedded contentFiles。
	ContentFiles []tbdom.ContentFile

	AssigneeID string
	CreatedBy  string

	ActorID string
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

	contentFiles, err := validateAndCopyContentFiles(in.ContentFiles)
	if err != nil {
		return nil, err
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:        in.Name,
		Symbol:      in.Symbol,
		BrandID:     in.BrandID,
		CompanyID:   in.CompanyID,
		Description: in.Description,

		IconURL:      in.IconURL,
		ContentFiles: contentFiles,

		AssigneeID: in.AssigneeID,

		CreatedAt: nil,
		CreatedBy: in.CreatedBy,
		UpdatedAt: nil,
		UpdatedBy: in.CreatedBy,

		MetadataURI: "",
	})
	if err != nil {
		return nil, err
	}

	// tokenBlueprint 起票と同時に tokenBlueprint_review の aggregate も起票する。
	// 失敗しても tokenBlueprint 本体は作成済みなので best-effort。
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

// ============================================================
// Read
// ============================================================

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

// ============================================================
// Update
// ============================================================

type UpdateBlueprintRequest struct {
	ID          string
	Name        *string
	Symbol      *string
	BrandID     *string
	Description *string
	AssigneeID  *string

	// Firebase Storage downloadURL.
	IconURL *string

	// Firebase Storage objectPath / downloadURL を含む embedded contentFiles。
	ContentFiles *[]tbdom.ContentFile

	MetadataURI *string
	Minted      *bool

	ActorID string
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

	contentFiles, err := normalizeContentFilesPtr(in.ContentFiles)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.Update(ctx, in.ID, tbdom.UpdateTokenBlueprintInput{
		Name:        in.Name,
		Symbol:      in.Symbol,
		BrandID:     in.BrandID,
		Description: in.Description,
		AssigneeID:  in.AssigneeID,

		IconURL:      in.IconURL,
		ContentFiles: contentFiles,

		MetadataURI: in.MetadataURI,
		Minted:      in.Minted,

		UpdatedAt: &now,
		UpdatedBy: ptr(in.ActorID),
		DeletedAt: nil,
		DeletedBy: nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

// ============================================================
// Delete
// ============================================================

func (u *TokenBlueprintCRUDUsecase) Delete(ctx context.Context, id string) error {
	if u == nil || u.tbRepo == nil {
		return tbdom.ErrInvalid
	}

	if id == "" {
		return tbdom.ErrInvalidID
	}

	return u.tbRepo.Delete(ctx, id)
}
