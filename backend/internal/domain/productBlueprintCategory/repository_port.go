// backend/internal/domain/productBlueprintCategory/repository_port.go
package productBlueprintCategory

import (
	"context"
	"errors"

	"narratives/internal/domain/common"
)

// ======================================
// Repository errors
// ======================================

var (
	ErrRepositoryInvalidInput = errors.New("productBlueprintCategory repository: invalid input")
)

// ======================================
// Filter / Patch
// ======================================

// Filter は ProductBlueprintCategory 一覧取得用フィルタ。
// 共通の検索・作成日時・更新日時フィルタは common.FilterCommon を埋め込む。
//
// NOTE:
// この repository は productBlueprintCategories collection に保存される
// カテゴリマスタそのものを扱う。
// category ごとの入力項目定義は input_schema.go の静的 schema registry 側で管理し、
// repository の永続化対象にはしない。
type Filter struct {
	common.FilterCommon

	IDs []CategoryID

	Code *CategoryCode
	Kind *CategoryKind

	ParentID *CategoryID

	// ParentID が nil のトップ階層だけを取得したい場合に true。
	// ParentID と RootOnly が両方指定された場合は repository 実装側で ErrRepositoryInvalidInput を返す想定。
	RootOnly bool
}

// Patch は ProductBlueprintCategory の部分更新用。
// common.RepositoryCRUD の Update(ctx, id, patch) で使う。
//
// NOTE:
// Patch はカテゴリマスタの更新だけを扱う。
// 入力項目 schema は code に紐づく domain 定義として input_schema.go で管理するため、
// Patch には含めない。
type Patch struct {
	Code *CategoryCode

	NameJa *string
	NameEn *string

	ParentID *CategoryID
	Path     []string

	Kind *CategoryKind

	DisplayOrder *int

	Attributes *CategoryAttributes
}

// ======================================
// Sort
// ======================================

const (
	SortColumnDisplayOrder = "displayOrder"
	SortColumnCode         = "code"
	SortColumnNameJa       = "nameJa"
	SortColumnKind         = "kind"
	SortColumnCreatedAt    = "createdAt"
	SortColumnUpdatedAt    = "updatedAt"
)

func IsAllowedSortColumn(column string) bool {
	switch column {
	case SortColumnDisplayOrder,
		SortColumnCode,
		SortColumnNameJa,
		SortColumnKind,
		SortColumnCreatedAt,
		SortColumnUpdatedAt:
		return true
	default:
		return false
	}
}

// ======================================
// RepositoryPort
// ======================================

// RepositoryPort は ProductBlueprintCategory の永続化境界。
// Firestore などの具体的な保存先は adapter/out 側で実装する。
//
// NOTE:
// この port はカテゴリマスタのみを扱う。
// カテゴリごとの入力項目定義は repository から取得せず、
// domain の input_schema.go に定義する GetCategoryInputSchema / HasModelFields 等を利用する。
type RepositoryPort interface {
	common.Repository[ProductBlueprintCategory, Filter, Patch]

	GetByCode(ctx context.Context, code CategoryCode) (ProductBlueprintCategory, error)

	// ListTree はフロントのカテゴリ選択 UI 向け。
	// displayOrder 昇順で返す想定。
	ListTree(ctx context.Context) ([]ProductBlueprintCategory, error)

	ListCursor(
		ctx context.Context,
		filter Filter,
		page common.CursorPage,
	) (common.CursorPageResult[ProductBlueprintCategory], error)

	ExistsByID(ctx context.Context, id string) (bool, error)
	ExistsByCode(ctx context.Context, code CategoryCode) (bool, error)
}
