// backend/internal/domain/member/repository_port.go
package member

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// Filter defines list conditions for members.
// NOTE: Field names/semantics align with entity.Member (camelCase in JSON/Firestore).
type Filter struct {
	// Free-text query for name/kana/email, etc.
	SearchQuery string

	// Brand filters (alias supported for backward compatibility).
	BrandIDs []string // preferred
	Brands   []string // legacy alias of BrandIDs

	// Company scope & status
	CompanyID string // owning company to scope results
	Status    string // "", "active", "inactive"

	// Ranges
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// Permission names (AND)
	Permissions []string
}

// Sort describes ordering for list results.
// Column uses symbolic names consumed by adapters/handlers.
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	// Historical naming kept for compatibility with handlers/adapters.
	// Handlers typically map "joinedAt" -> entity.CreatedAt.
	SortByJoinedAt      SortColumn = "joinedAt"
	SortByPermissions   SortColumn = "permissions"
	SortByAssigneeCount SortColumn = "assigneeCount"

	// Common columns
	SortByName      SortColumn = "name"
	SortByEmail     SortColumn = "email"
	SortByUpdatedAt SortColumn = "updatedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Common aliases
type Page = common.Page
type PageResult = common.PageResult[Member]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Member]
type SaveOptions = common.SaveOptions

// Repository is the persistence port for the Member aggregate.
type Repository interface {
	// Common CRUD/List
	common.RepositoryCRUD[Member, MemberPatch]
	common.RepositoryList[Member, Filter]

	// Additional requirements
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult, error)
	GetByID(ctx context.Context, id string) (Member, error)
	GetByEmail(ctx context.Context, email string) (Member, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Save(ctx context.Context, m Member, opts *SaveOptions) (Member, error)
	Reset(ctx context.Context) error
}
