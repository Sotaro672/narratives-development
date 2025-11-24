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

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ------------------------------------------------------------
// Commands (単体)
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	return u.repo.Create(ctx, v)
}

func (u *ProductBlueprintUsecase) Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	return u.repo.Save(ctx, v)
}

func (u *ProductBlueprintUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// ------------------------------------------------------------
// Variations の紐付け専用ユースケース
// ------------------------------------------------------------

// AttachVariations は既存の ProductBlueprint に VariationIDs を紐付けます。
// PATCH /product-blueprints/{id}/variations から呼ばれる想定です。
func (u *ProductBlueprintUsecase) AttachVariations(
	ctx context.Context,
	id string,
	variationIDs []string,
) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	pb, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	_, err = u.repo.Save(ctx, pb)
	return err
}

// CreateBlueprintAndModels は「Blueprint を作成し、その直後に VariationIDs を紐付ける」
// 高レベルユースケースです。
// - 先に ProductBlueprint を Create
// - その結果得られた ID に対して VariationIDs をセットして Save
// （Model ドキュメント自体の作成は別ユースケース／別リポジトリで行う前提）
func (u *ProductBlueprintUsecase) CreateBlueprintAndModels(
	ctx context.Context,
	v productbpdom.ProductBlueprint,
	variationIDs []string,
) (productbpdom.ProductBlueprint, error) {
	// 1. Blueprint 単体を作成
	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// Variation が無ければそのまま返す
	if len(variationIDs) == 0 {
		return created, nil
	}

	saved, err := u.repo.Save(ctx, created)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}
	return saved, nil
}
