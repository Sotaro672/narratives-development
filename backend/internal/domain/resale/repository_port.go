// backend/internal/domain/resale/repository_port.go
package resale

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ResaleImagePatch is a partial update input for resale image records.
// nil fields are not updated.
type ResaleImagePatch struct {
	URL          *string
	DisplayOrder *int

	UpdatedAt *time.Time
	UpdatedBy *string
}

// Filter is the query condition for Resale repository.
type Filter struct {
	// Free text search.
	// Implementation may interpret this as partial match against id,
	// mintAddress, description, etc.
	SearchQuery string

	IDs []string

	MintAddresses []string

	TokenBlueprintIDs   []string
	ProductIDs          []string
	BrandIDs            []string
	ProductBlueprintIDs []string
	AvatarIDs           []string

	Status   *ResaleStatus
	Statuses []ResaleStatus

	Condition  *ResaleCondition
	Conditions []ResaleCondition

	MinPrice *int
	MaxPrice *int
}

// ResaleImageFilter is the query condition for ResaleImage repository.
type ResaleImageFilter struct {
	ResaleID  *string
	ResaleIDs []string

	IDs []string
	URL *string
}

// Common type aliases.
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// Contract errors.
var (
	ErrNotFound = errors.New("resale: not found")
	ErrConflict = errors.New("resale: conflict")
)

// Repository is the repository port for Resale aggregate root.
type Repository interface {
	// Resale query.
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Resale], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Resale], error)

	// Read.
	GetByID(ctx context.Context, id string) (Resale, error)
	ListByAvatarID(ctx context.Context, avatarID string) ([]Resale, error)

	// Write.
	Create(ctx context.Context, r Resale) (Resale, error)

	// Update replaces/upserts the mutable fields of a persisted Resale.
	//
	// Expected implementation policy:
	// - id is the target document id.
	// - r.ID may be empty or equal to id.
	// - immutable fields such as MintAddress, CreatedAt, CreatedBy should not be overwritten
	//   unless the implementation intentionally treats Update as full replacement.
	Update(ctx context.Context, id string, r Resale) (Resale, error)

	// Delete physically deletes a resale document.
	// Implementations may also delete child image records if the storage supports subcollections.
	Delete(ctx context.Context, id string) error
}

// ImageRepository is the repository port for resale image records.
//
// Image policy:
// - Backend stores only Firebase Storage download URL.
// - Backend does not manage objectPath, fileName, contentType, or size.
// - Image record is scoped by resaleId.
// - imageID alone should not be used as a global lookup key.
type ImageRepository interface {
	// Query.
	ListByResaleID(ctx context.Context, resaleID string) ([]ResaleImage, error)

	// Write.
	Create(ctx context.Context, img ResaleImage) (ResaleImage, error)
	Update(ctx context.Context, resaleID string, imageID string, patch ResaleImagePatch) (ResaleImage, error)

	// Delete physically deletes a resale image record.
	Delete(ctx context.Context, resaleID string, imageID string) error
}
