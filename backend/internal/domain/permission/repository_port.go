package permission

import (
	"context"
	"errors"

	dcommon "narratives/internal/domain/common"
)

// ========================================
// Entity Patch (contract only)
// ========================================

type PermissionPatch struct {
	Name        *string
	Category    *PermissionCategory
	Description *string
}

// ========================================
// Common repository errors
// ========================================

var (
	ErrNotFound = errors.New("permission: not found")
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

type Repository interface {
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Permission], error)
	GetByID(ctx context.Context, id string) (Permission, error)
	Create(ctx context.Context, p Permission) (Permission, error)
	Update(ctx context.Context, id string, patch PermissionPatch) (Permission, error)
	Delete(ctx context.Context, id string) error
}

// Optional: expose common generic interfaces if your codebase uses them
type RepositoryCRUD[T any, P any] = dcommon.RepositoryCRUD[T, P]
type RepositoryList[T any, F any] = dcommon.RepositoryList[T, F]
