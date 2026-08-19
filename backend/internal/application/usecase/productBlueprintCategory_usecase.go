// backend/internal/application/usecase/productBlueprintCategory_usecase.go
package usecase

import (
	"context"

	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ProductBlueprintCategoryUsecase は、
// 商品設計カテゴリマスタを読み取り専用で扱う application service です。
type ProductBlueprintCategoryUsecase struct {
	repo categorydom.ReadOnlyRepositoryPort
}

func NewProductBlueprintCategoryUsecase(
	repo categorydom.ReadOnlyRepositoryPort,
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

	Paths [][]string

	SortColumn string
	SortOrder  common.SortOrder

	Page    int
	PerPage int
}

// ------------------------------------------------------------
// Read methods
// ------------------------------------------------------------

func (u *ProductBlueprintCategoryUsecase) GetByPath(
	ctx context.Context,
	path []string,
) (categorydom.ProductBlueprintCategory, error) {
	if u == nil || u.repo == nil {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrRepositoryInvalidInput
	}

	if !isValidProductBlueprintCategoryPath(path) {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidPath
	}

	return u.repo.GetByPath(
		ctx,
		append(
			[]string(nil),
			path...,
		),
	)
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
		sortSpec.Column = categorydom.SortColumnPath
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

	return u.repo.List(
		ctx,
		filter,
		sortSpec,
		page,
	)
}

// ListTree はフロントのカテゴリ選択 UI 向けに、
// productBlueprintCategoryPath の階層順のカテゴリ一覧を返します。
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

	return u.repo.ListCursor(
		ctx,
		filter,
		page,
	)
}

func (u *ProductBlueprintCategoryUsecase) ExistsByPath(
	ctx context.Context,
	path []string,
) (bool, error) {
	if u == nil || u.repo == nil {
		return false, categorydom.ErrRepositoryInvalidInput
	}

	if !isValidProductBlueprintCategoryPath(path) {
		return false, categorydom.ErrInvalidPath
	}

	return u.repo.ExistsByPath(
		ctx,
		append(
			[]string(nil),
			path...,
		),
	)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func buildProductBlueprintCategoryFilter(
	q ListProductBlueprintCategoriesQuery,
) (categorydom.Filter, error) {
	paths := make([][]string, 0, len(q.Paths))

	for _, path := range q.Paths {
		if !isValidProductBlueprintCategoryPath(path) {
			return categorydom.Filter{}, categorydom.ErrInvalidPath
		}

		paths = append(
			paths,
			append(
				[]string(nil),
				path...,
			),
		)
	}

	return categorydom.Filter{
		FilterCommon: common.FilterCommon{
			SearchQuery: q.SearchQuery,
		},
		Paths: paths,
	}, nil
}

func isValidProductBlueprintCategoryPath(
	path []string,
) bool {
	if len(path) == 0 {
		return false
	}

	for _, segment := range path {
		if segment == "" {
			return false
		}
	}

	return true
}
