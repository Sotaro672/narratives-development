// backend/internal/application/usecase/productBlueprint_usecase.go
package usecase

import (
	"context"
	"strings"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepo defines the minimal persistence port needed by ProductBlueprintUsecase.
type ProductBlueprintRepo interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 一覧取得用（companyId による絞り込みは repository 側の実装に委譲）
	List(ctx context.Context) ([]productbpdom.ProductBlueprint, error)

	Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Delete(ctx context.Context, id string) error
}

// ProductBlueprintUsecase orchestrates productBlueprint operations.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo
}

func NewProductBlueprintUsecase(repo ProductBlueprintRepo) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{repo: repo}
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// List
// handler 側の GET /product-blueprints から利用される一覧取得。
// companyId でのテナント絞り込みは、現状は repository 実装に委譲する形にしています。
// （もし usecase 層で companyId を強制したい場合は、BrandUsecase 同様に
//
//	Filter 型を導入していく想定）
func (u *ProductBlueprintUsecase) List(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	return u.repo.List(ctx)
}

// ------------------------------------------------------------
// Commands (単体)
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) Create(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	// ★ BrandUsecase と同様:
	//   context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}
	return u.repo.Create(ctx, v)
}

func (u *ProductBlueprintUsecase) Save(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	// ★ BrandUsecase と同様:
	//   context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		v.CompanyID = strings.TrimSpace(cid)
	}
	return u.repo.Save(ctx, v)
}

func (u *ProductBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
