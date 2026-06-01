// backend/internal/domain/productBlueprintCategory/repository_port.go
package productBlueprintCategory

import (
	"context"

	"narratives/internal/domain/common"
)

// ======================================
// Filter
// ======================================

// Filter は ProductBlueprintCategory 一覧取得用フィルタ。
//
// NOTE:
// productBlueprintCategories collection は seed_category によって投入される
// 読み取り専用のカテゴリマスタとして扱う。
// Console API からカテゴリを作成・更新・削除しない。
//
// category ごとの入力項目定義は input_schema.go の静的 schema registry 側で管理し、
// repository / Firestore の永続化対象にはしない。
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
// ReadOnlyRepositoryPort
// ======================================

// ReadOnlyRepositoryPort は ProductBlueprintCategory の読み取り専用境界。
//
// NOTE:
// productBlueprintCategories collection は seed_category で投入する。
// Console API では読み取りのみ行い、Create / Update / Delete は提供しない。
//
// カテゴリごとの入力項目定義は repository から取得せず、
// domain の input_schema.go に定義する GetCategoryInputSchema / HasModelFields 等を利用する。
type ReadOnlyRepositoryPort interface {
	GetByID(ctx context.Context, id string) (ProductBlueprintCategory, error)

	GetByCode(ctx context.Context, code CategoryCode) (ProductBlueprintCategory, error)

	List(
		ctx context.Context,
		filter Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[ProductBlueprintCategory], error)

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

// RepositoryPort は後方互換用 alias。
//
// NOTE:
// 新規実装では ReadOnlyRepositoryPort を優先する。
// 将来的に管理画面からカテゴリを編集する場合のみ、
// write 用 port を別途定義する。
type RepositoryPort = ReadOnlyRepositoryPort
