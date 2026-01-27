// backend/internal/application/usecase/tokenBlueprint_crud_usecase.go
package tokenBlueprint

import (
	"context"
	"fmt"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintCRUDUsecase focuses on persistence CRUD only.
type TokenBlueprintCRUDUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintCRUDUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintCRUDUsecase {
	return &TokenBlueprintCRUDUsecase{tbRepo: tbRepo}
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

	AssigneeID string
	CreatedBy  string

	// ★ objectPath 永続化（tokenIcon / tokenContents）
	// - tokenIconObjectPath: 例 "{id}/icon"（tokenIconObjectPath(id)）
	// - tokenContentsObjectPath: 例 "{id}/.keep"（keepObjectPath(id)）
	//   ※ token-contents は “参照パス” として .keep を採用する方針に合わせる
	TokenIconObjectPath     string
	TokenContentsObjectPath string

	// ActorID is intentionally not used by pure CRUD usecase (audit is handled by repo/inputs).
	ActorID string
}

func (u *TokenBlueprintCRUDUsecase) Create(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:        strings.TrimSpace(in.Name),
		Symbol:      strings.TrimSpace(in.Symbol),
		BrandID:     strings.TrimSpace(in.BrandID),
		CompanyID:   strings.TrimSpace(in.CompanyID),
		Description: strings.TrimSpace(in.Description),

		// entity.go 正: embedded contents
		ContentFiles: nil,

		AssigneeID: strings.TrimSpace(in.AssigneeID),

		CreatedAt: nil,
		CreatedBy: strings.TrimSpace(in.CreatedBy),
		UpdatedAt: nil,
		UpdatedBy: "",

		// create 時は metadataUri を作成しない（保存しない方針）
		MetadataURI: "",

		// ★ objectPath 永続化（create で保存）
		TokenIconObjectPath:     strings.TrimSpace(in.TokenIconObjectPath),
		TokenContentsObjectPath: strings.TrimSpace(in.TokenContentsObjectPath),
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

// ============================================================
// Read
// ============================================================

func (u *TokenBlueprintCRUDUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}
	tid := strings.TrimSpace(id)
	return u.tbRepo.GetByID(ctx, tid)
}

func (u *TokenBlueprintCRUDUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	if u == nil || u.tbRepo == nil {
		return tbdom.PageResult{}, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}

	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return tbdom.PageResult{}, fmt.Errorf("companyId is empty")
	}
	return u.tbRepo.ListByCompanyID(ctx, cid, page)
}

func (u *TokenBlueprintCRUDUsecase) ListByBrandID(ctx context.Context, brandID string, page tbdom.Page) (tbdom.PageResult, error) {
	if u == nil || u.tbRepo == nil {
		return tbdom.PageResult{}, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}

	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return tbdom.PageResult{}, fmt.Errorf("brandId is empty")
	}
	return tbdom.ListByBrandID(ctx, u.tbRepo, bid, page)
}

// ★ ListMintedNotYet は domain から削除されたため、この usecase からも削除
// func (u *TokenBlueprintCRUDUsecase) ListMintedNotYet(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
// 	if u == nil || u.tbRepo == nil {
// 		return tbdom.PageResult{}, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
// 	}
// 	return tbdom.ListMintedNotYet(ctx, u.tbRepo, page)
// }

func (u *TokenBlueprintCRUDUsecase) ListMintedCompleted(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	if u == nil || u.tbRepo == nil {
		return tbdom.PageResult{}, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
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

	// ★ objectPath 永続化（tokenIcon / tokenContents）
	// - tokenIconObjectPath: 例 "{id}/icon"
	// - tokenContentsObjectPath: 例 "{id}/.keep"
	TokenIconObjectPath     *string
	TokenContentsObjectPath *string

	// entity.go 正: embedded
	ContentFiles *[]tbdom.ContentFile // 全置換

	ActorID string
}

func (u *TokenBlueprintCRUDUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(in.ID), tbdom.UpdateTokenBlueprintInput{
		Name:        trimPtr(in.Name),
		Symbol:      trimPtr(in.Symbol),
		BrandID:     trimPtr(in.BrandID),
		Description: trimPtr(in.Description),
		AssigneeID:  trimPtr(in.AssigneeID),

		ContentFiles: normalizeContentFilesPtr(in.ContentFiles),

		// objectPath は update で更新されない方針なら、ここは nil 固定にする
		// （呼び出し側が値を入れても repo 側で無視する設計に合わせる）
		TokenIconObjectPath:     nil,
		TokenContentsObjectPath: nil,

		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(in.ActorID)),
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
		return fmt.Errorf("tokenBlueprint CRUD usecase/repo is nil")
	}
	tid := strings.TrimSpace(id)
	return u.tbRepo.Delete(ctx, tid)
}
