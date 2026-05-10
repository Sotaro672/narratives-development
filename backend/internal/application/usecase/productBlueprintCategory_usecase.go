// backend/internal/application/usecase/productBlueprintCategory_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ProductBlueprintCategoryRepository は application 層が利用する
// productBlueprintCategory 永続化ポートです。
type ProductBlueprintCategoryRepository interface {
	GetByID(ctx context.Context, id string) (categorydom.ProductBlueprintCategory, error)
	GetByCode(ctx context.Context, code categorydom.CategoryCode) (categorydom.ProductBlueprintCategory, error)

	List(
		ctx context.Context,
		filter categorydom.Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[categorydom.ProductBlueprintCategory], error)

	ListTree(ctx context.Context) ([]categorydom.ProductBlueprintCategory, error)

	Create(
		ctx context.Context,
		entity categorydom.ProductBlueprintCategory,
	) (categorydom.ProductBlueprintCategory, error)

	Update(
		ctx context.Context,
		id string,
		patch categorydom.Patch,
	) (categorydom.ProductBlueprintCategory, error)

	Delete(ctx context.Context, id string) error

	ExistsByID(ctx context.Context, id string) (bool, error)
	ExistsByCode(ctx context.Context, code categorydom.CategoryCode) (bool, error)
}

// ProductBlueprintCategoryUsecase は商品設計書カテゴリの application service です。
type ProductBlueprintCategoryUsecase struct {
	repo ProductBlueprintCategoryRepository
}

func NewProductBlueprintCategoryUsecase(
	repo ProductBlueprintCategoryRepository,
) *ProductBlueprintCategoryUsecase {
	return &ProductBlueprintCategoryUsecase{
		repo: repo,
	}
}

// ------------------------------------------------------------
// Input DTOs
// ------------------------------------------------------------

type CreateProductBlueprintCategoryCommand struct {
	ID string

	Code   string
	NameJa string
	NameEn string

	ParentID *string
	Path     []string

	Kind string

	DisplayOrder int

	Attributes categorydom.CategoryAttributes

	Now *time.Time
}

type UpdateProductBlueprintCategoryCommand struct {
	Code *string

	NameJa *string
	NameEn *string

	ParentID *string
	Path     []string

	Kind *string

	DisplayOrder *int

	Attributes *categorydom.CategoryAttributes
}

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
	if strings.TrimSpace(id) == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidID
	}

	return u.repo.GetByID(ctx, id)
}

func (u *ProductBlueprintCategoryUsecase) GetByCode(
	ctx context.Context,
	code string,
) (categorydom.ProductBlueprintCategory, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidCode
	}

	return u.repo.GetByCode(ctx, categorydom.CategoryCode(code))
}

func (u *ProductBlueprintCategoryUsecase) List(
	ctx context.Context,
	q ListProductBlueprintCategoriesQuery,
) (common.PageResult[categorydom.ProductBlueprintCategory], error) {
	filter, err := buildProductBlueprintCategoryFilter(q)
	if err != nil {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, err
	}

	sort := common.Sort{
		Column: q.SortColumn,
		Order:  q.SortOrder,
	}
	if sort.Column == "" {
		sort.Column = categorydom.SortColumnDisplayOrder
	}
	if sort.Order == "" {
		sort.Order = common.SortAsc
	}
	if !categorydom.IsAllowedSortColumn(sort.Column) {
		return common.PageResult[categorydom.ProductBlueprintCategory]{}, categorydom.ErrRepositoryInvalidInput
	}

	page := common.Page{
		Number:  q.Page,
		PerPage: q.PerPage,
	}
	if page.Number <= 0 {
		page.Number = 1
	}

	return u.repo.List(ctx, filter, sort, page)
}

// ListTree はフロントのカテゴリ選択 UI 用に、
// カテゴリ一覧を displayOrder 順で返します。
// tree への整形は handler/response mapper 側で行ってもよいです。
func (u *ProductBlueprintCategoryUsecase) ListTree(
	ctx context.Context,
) ([]categorydom.ProductBlueprintCategory, error) {
	return u.repo.ListTree(ctx)
}

// ------------------------------------------------------------
// Write methods
// ------------------------------------------------------------

func (u *ProductBlueprintCategoryUsecase) Create(
	ctx context.Context,
	cmd CreateProductBlueprintCategoryCommand,
) (categorydom.ProductBlueprintCategory, error) {
	id := categorydom.CategoryID(strings.TrimSpace(cmd.ID))
	code := categorydom.CategoryCode(strings.TrimSpace(cmd.Code))
	nameJa := strings.TrimSpace(cmd.NameJa)
	nameEn := strings.TrimSpace(cmd.NameEn)
	kind := categorydom.CategoryKind(strings.TrimSpace(cmd.Kind))

	var parentID *categorydom.CategoryID
	if cmd.ParentID != nil && strings.TrimSpace(*cmd.ParentID) != "" {
		v := categorydom.CategoryID(strings.TrimSpace(*cmd.ParentID))
		parentID = &v
	}

	now := time.Now().UTC()
	if cmd.Now != nil && !cmd.Now.IsZero() {
		now = cmd.Now.UTC()
	}

	if exists, err := u.repo.ExistsByID(ctx, string(id)); err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	} else if exists {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrConflict
	}

	if exists, err := u.repo.ExistsByCode(ctx, code); err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	} else if exists {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrConflict
	}

	category, err := categorydom.New(
		id,
		code,
		nameJa,
		nameEn,
		parentID,
		cmd.Path,
		kind,
		cmd.DisplayOrder,
		cmd.Attributes,
		now,
	)
	if err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	return u.repo.Create(ctx, category)
}

func (u *ProductBlueprintCategoryUsecase) Update(
	ctx context.Context,
	id string,
	cmd UpdateProductBlueprintCategoryCommand,
) (categorydom.ProductBlueprintCategory, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidID
	}

	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return categorydom.ProductBlueprintCategory{}, err
	}

	patch := categorydom.Patch{}

	if cmd.Code != nil {
		code := categorydom.CategoryCode(strings.TrimSpace(*cmd.Code))
		if code == "" {
			return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidCode
		}

		if code != current.Code {
			if exists, err := u.repo.ExistsByCode(ctx, code); err != nil {
				return categorydom.ProductBlueprintCategory{}, err
			} else if exists {
				return categorydom.ProductBlueprintCategory{}, categorydom.ErrConflict
			}
		}

		patch.Code = &code
	}

	if cmd.NameJa != nil {
		nameJa := strings.TrimSpace(*cmd.NameJa)
		if nameJa == "" {
			return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidNameJa
		}
		patch.NameJa = &nameJa
	}

	if cmd.NameEn != nil {
		nameEn := strings.TrimSpace(*cmd.NameEn)
		patch.NameEn = &nameEn
	}

	if cmd.ParentID != nil {
		parentIDValue := strings.TrimSpace(*cmd.ParentID)
		if parentIDValue == "" {
			patch.ParentID = nil
		} else {
			parentID := categorydom.CategoryID(parentIDValue)
			patch.ParentID = &parentID
		}
	}

	if cmd.Path != nil {
		patch.Path = normalizeStringList(cmd.Path)
	}

	if cmd.Kind != nil {
		kind := categorydom.CategoryKind(strings.TrimSpace(*cmd.Kind))
		if !categorydom.IsValidCategoryKind(kind) {
			return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidKind
		}
		patch.Kind = &kind
	}

	if cmd.DisplayOrder != nil {
		if *cmd.DisplayOrder <= 0 {
			return categorydom.ProductBlueprintCategory{}, categorydom.ErrInvalidDisplayOrder
		}
		displayOrder := *cmd.DisplayOrder
		patch.DisplayOrder = &displayOrder
	}

	if cmd.Attributes != nil {
		attrs := *cmd.Attributes
		patch.Attributes = &attrs
	}

	return u.repo.Update(ctx, id, patch)
}

func (u *ProductBlueprintCategoryUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if strings.TrimSpace(id) == "" {
		return categorydom.ErrInvalidID
	}

	return u.repo.Delete(ctx, id)
}

// ------------------------------------------------------------
// Snapshot helper
// ------------------------------------------------------------

// BuildProductBlueprintCategorySnapshot は productBlueprint 作成/更新時に使う
// denormalized snapshot を作るための helper です。
func (u *ProductBlueprintCategoryUsecase) BuildProductBlueprintCategorySnapshot(
	ctx context.Context,
	categoryID string,
) (categorydom.Snapshot, error) {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return categorydom.Snapshot{}, categorydom.ErrInvalidID
	}

	category, err := u.repo.GetByID(ctx, categoryID)
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
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		ids = append(ids, categorydom.CategoryID(raw))
	}

	var code *categorydom.CategoryCode
	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		v := categorydom.CategoryCode(strings.TrimSpace(*q.Code))
		code = &v
	}

	var kind *categorydom.CategoryKind
	if q.Kind != nil && strings.TrimSpace(*q.Kind) != "" {
		v := categorydom.CategoryKind(strings.TrimSpace(*q.Kind))
		if !categorydom.IsValidCategoryKind(v) {
			return categorydom.Filter{}, categorydom.ErrInvalidKind
		}
		kind = &v
	}

	var parentID *categorydom.CategoryID
	if q.ParentID != nil && strings.TrimSpace(*q.ParentID) != "" {
		v := categorydom.CategoryID(strings.TrimSpace(*q.ParentID))
		parentID = &v
	}

	if parentID != nil && q.RootOnly {
		return categorydom.Filter{}, categorydom.ErrRepositoryInvalidInput
	}

	filter := categorydom.Filter{
		FilterCommon: common.FilterCommon{
			SearchQuery: strings.TrimSpace(q.SearchQuery),
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
	}

	return filter, nil
}

func normalizeStringList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))

	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}
