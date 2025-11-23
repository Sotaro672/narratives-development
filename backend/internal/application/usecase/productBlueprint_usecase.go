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

// Queries

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *ProductBlueprintUsecase) Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	return u.repo.Create(ctx, v)
}

// CreateWithVariations は Handler から呼ばれる拡張メソッドです。
// 現状は Variations を含んだ ProductBlueprint を そのまま repo.Create に委譲します。
// 将来的に「モデルテーブルの同時作成」などが必要になった場合は、ここで
// 他の Usecase/Repo を呼び出す形に拡張できます。
func (u *ProductBlueprintUsecase) CreateWithVariations(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	return u.repo.Create(ctx, v)
}

func (u *ProductBlueprintUsecase) Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	return u.repo.Save(ctx, v)
}

func (u *ProductBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
