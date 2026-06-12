// backend/internal/application/usecase/productBlueprintCategory_usecase.go
package usecase

import (
	"context"
	"time"

	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ProductBlueprintCategoryReadRepository は、
// seed 済み productBlueprintCategories collection を読み取るための repository port です。
//
// NOTE:
// - productBlueprintCategories は backend/cmd/seed_category で投入する
// - Console API から category master の作成・更新・削除は行わない
// - category ごとの入力項目定義は Firestore ではなく domain/input_schema.go 側で管理する
type ProductBlueprintCategoryReadRepository interface {
	GetByID(
		ctx context.Context,
		id string,
	) (categorydom.ProductBlueprintCategory, error)

	List(
		ctx context.Context,
		filter categorydom.Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[categorydom.ProductBlueprintCategory], error)

	// ListTree はフロントのカテゴリ選択 UI 向け。
	// displayOrder 昇順で返す想定。
	ListTree(
		ctx context.Context,
	) ([]categorydom.ProductBlueprintCategory, error)

	ListCursor(
		ctx context.Context,
		filter categorydom.Filter,
		page common.CursorPage,
	) (common.CursorPageResult[categorydom.ProductBlueprintCategory], error)

	ExistsByID(
		ctx context.Context,
		id string,
	) (bool, error)
}

// ProductBlueprintCategoryUsecase は、
// 商品設計カテゴリマスタを読み取り専用で扱う application service です。
type ProductBlueprintCategoryUsecase struct {
	repo ProductBlueprintCategoryReadRepository
}

func NewProductBlueprintCategoryUsecase(
	repo ProductBlueprintCategoryReadRepository,
) *ProductBlueprintCategoryUsecase {
	return &ProductBlueprintCategoryUsecase{
		repo: repo,
	}
}

// ------------------------------------------------------------
// Input DTOs
// ------------------------------------------------------------

type ListProductBlueprintCategoriesQuery struct {
	SearchQuery string

	IDs []string

	Code *string
	Kind *string

	ParentID *string
	RootOnly bool

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	SortColumn string
	SortOrder  common.SortOrder

	Page    int
	PerPage int
}

// ------------------------------------------------------------
// Read methods
// ------------------------------------------------------------

func (u *ProductBlueprintCategoryUsecase) GetByID(
	ctx context.Context,
	id string,
) (categorydom.ProductBlueprintCategory, error) {
	if u == nil || u.repo == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrRepositoryInvalidInput
	}

	if id == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidID
	}

	return u.repo.GetByID(ctx, id)
}

func (u *ProductBlueprintCategoryUsecase) List(
	ctx context.Context,
	q ListProductBlueprintCategoriesQuery,
) (common.PageResult[categorydom.ProductBlueprintCategory], error) {
	if u == nil || u.repo == nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	filter, err := buildProductBlueprintCategoryFilter(q)
	if err != nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	sortSpec := common.Sort{
		Column: q.SortColumn,
		Order:  q.SortOrder,
	}

	if sortSpec.Column == "" {
		sortSpec.Column = categorydom.SortColumnDisplayOrder
	}

	if sortSpec.Order == "" {
		sortSpec.Order = common.SortAsc
	}

	if !categorydom.IsAllowedSortColumn(sortSpec.Column) {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	page := common.Page{
		Number:  q.Page,
		PerPage: q.PerPage,
	}

	if page.Number <= 0 {
		page.Number = 1
	}

	if page.PerPage <= 0 {
		page.PerPage = 20
	}

	return u.repo.List(ctx, filter, sortSpec, page)
}

// ListTree はフロントのカテゴリ選択 UI 向けに、
// displayOrder 昇順のカテゴリ一覧を返します。
func (u *ProductBlueprintCategoryUsecase) ListTree(
	ctx context.Context,
) ([]categorydom.ProductBlueprintCategory, error) {
	if u == nil || u.repo == nil {
		return nil, categorydom.ErrRepositoryInvalidInput
	}

	return u.repo.ListTree(ctx)
}

func (u *ProductBlueprintCategoryUsecase) ListCursor(
	ctx context.Context,
	q ListProductBlueprintCategoriesQuery,
	page common.CursorPage,
) (common.CursorPageResult[categorydom.ProductBlueprintCategory], error) {
	if u == nil || u.repo == nil {
		return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	filter, err := buildProductBlueprintCategoryFilter(q)
	if err != nil {
		return common.CursorPageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	return u.repo.ListCursor(ctx, filter, page)
}

func (u *ProductBlueprintCategoryUsecase) ExistsByID(
	ctx context.Context,
	id string,
) (bool, error) {
	if u == nil || u.repo == nil {
		return false, categorydom.ErrRepositoryInvalidInput
	}

	if id == "" {
		return false, categorydom.ErrInvalidID
	}

	return u.repo.ExistsByID(ctx, id)
}

// BuildProductBlueprintCategorySnapshot は productBlueprint 作成/更新時に使う
// denormalized snapshot を作るための helper です。
func (u *ProductBlueprintCategoryUsecase) BuildProductBlueprintCategorySnapshot(
	ctx context.Context,
	categoryID string,
) (categorydom.Snapshot, error) {
	if categoryID == "" {
		return categorydom.Snapshot{}, categorydom.ErrInvalidID
	}

	category, err := u.GetByID(ctx, categoryID)
	if err != nil {
		return categorydom.Snapshot{}, err
	}

	return category.ToSnapshot(), nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func buildProductBlueprintCategoryFilter(
	q ListProductBlueprintCategoriesQuery,
) (categorydom.Filter, error) {
	ids := make([]categorydom.CategoryID, 0, len(q.IDs))
	for _, raw := range q.IDs {
		if raw == "" {
			continue
		}

		ids = append(ids, categorydom.CategoryID(raw))
	}

	var code *categorydom.CategoryCode
	if q.Code != nil && *q.Code != "" {
		v := categorydom.CategoryCode(*q.Code)
		code = &v
	}

	var kind *categorydom.CategoryKind
	if q.Kind != nil && *q.Kind != "" {
		v := categorydom.CategoryKind(*q.Kind)
		if !categorydom.IsValidCategoryKind(v) {
			return categorydom.Filter{}, categorydom.ErrInvalidKind
		}
		kind = &v
	}

	var parentID *categorydom.CategoryID
	if q.ParentID != nil && *q.ParentID != "" {
		v := categorydom.CategoryID(*q.ParentID)
		parentID = &v
	}

	if parentID != nil && q.RootOnly {
		return categorydom.Filter{}, categorydom.ErrRepositoryInvalidInput
	}

	return categorydom.Filter{
		FilterCommon: common.FilterCommon{
			SearchQuery: q.SearchQuery,
			Created: common.TimeRange{
				From: q.CreatedFrom,
				To:   q.CreatedTo,
			},
			Updated: common.TimeRange{
				From: q.UpdatedFrom,
				To:   q.UpdatedTo,
			},
		},
		IDs:      ids,
		Code:     code,
		Kind:     kind,
		ParentID: parentID,
		RootOnly: q.RootOnly,
	}, nil
}
