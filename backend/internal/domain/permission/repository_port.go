package permission

import (
	"context"
	"errors"

	dcommon "narratives/internal/domain/common"
)

// ========================================
// 読み取り専用 Permission コンテキスト
// ========================================
//
// Permission は「閲覧のみ」を前提としたドメインとして扱う。
// そのため、このパッケージの Repository では
//  - Create
//  - Update
//  - Delete
// といった変更系の操作は公開しない（他の初期化処理や Seeder などでのみ管理する）。

// ========================================
// Common repository errors
// ========================================

var (
	ErrNotFound = errors.New("permission: not found")
	// ErrConflict は書き込み系の共通エラーとして保持するが、
	// Repository インターフェースでは利用しない（将来のSeeder等で利用する可能性を考慮）。
	ErrConflict = errors.New("permission: conflict")
)

// ========================================
// Filters and shared types
// ========================================

type Filter struct {
	dcommon.FilterCommon
	Categories []PermissionCategory
}

// Aliases to shared domain types (sorting, paging, etc.)
type Sort = dcommon.Sort
type SortOrder = dcommon.SortOrder

const (
	SortAsc  = dcommon.SortAsc
	SortDesc = dcommon.SortDesc
)

type Page = dcommon.Page
type PageResult[T any] = dcommon.PageResult[T]

type Timestamps = dcommon.Timestamps
type TimeRange = dcommon.TimeRange
type CursorPage = dcommon.CursorPage
type CursorPageResult[T any] = dcommon.CursorPageResult[T]
type SaveOptions = dcommon.SaveOptions

// ========================================
// Repository Port (interface contracts only)
// ========================================
//
// 「閲覧のみ」を前提としているため、List / GetByID のみ公開する。
type Repository interface {
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Permission], error)
	GetByID(ctx context.Context, id string) (Permission, error)
}

// Optional: expose common generic interfaces if your codebase uses them
type RepositoryCRUD[T any, P any] = dcommon.RepositoryCRUD[T, P]
type RepositoryList[T any, F any] = dcommon.RepositoryList[T, F]
