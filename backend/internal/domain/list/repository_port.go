package list

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ListImagePatch is a partial update input for list image records.
// nil fields are not updated.
type ListImagePatch struct {
	URL          *string
	DisplayOrder *int

	UpdatedAt *time.Time
	UpdatedBy *string
}

// Filter is the query condition for List repository.
//
// NOTE:
// - ModelNumbers remains for backward compatibility.
// - Its actual meaning is modelId collection.
// - Price conditions are applied to Prices[] rows.
type Filter struct {
	// Free text search.
	// Implementation may interpret this as partial match against id,
	// readableId, title, description, etc.
	SearchQuery string

	IDs         []string
	ReadableIDs []string

	AssigneeID *string
	Status     *ListStatus
	Statuses   []ListStatus

	// Price conditions.
	ModelNumbers []string
	MinPrice     *int
	MaxPrice     *int

	// Filter by inventory ids.
	InventoryIDs []string
}

// ListImageFilter is the query condition for ListImage repository.
type ListImageFilter struct {
	ListID  *string
	ListIDs []string

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
	ErrNotFound = errors.New("list: not found")
	ErrConflict = errors.New("list: conflict")
)

// Repository is the repository port for List aggregate root.
type Repository interface {
	// List query.
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[List], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[List], error)

	// Count for pagination.
	// The filter interpretation must be same as List.
	Count(ctx context.Context, filter Filter) (int, error)

	// Read.
	GetByID(ctx context.Context, id string) (List, error)
	ListIDsByInventoryID(ctx context.Context, inventoryID string) ([]string, error)

	// Write.
	Create(ctx context.Context, l List) (List, error)

	// Update replaces/upserts the mutable fields of a persisted List.
	//
	// Expected implementation policy:
	// - id is the target document id.
	// - l.ID may be empty or equal to id.
	// - immutable fields such as InventoryID, CreatedAt, CreatedBy should not be overwritten
	//   unless the implementation intentionally treats Update as full replacement.
	Update(ctx context.Context, id string, l List) (List, error)

	// Delete physically deletes a list document.
	// Implementations may also delete child image records if the storage supports subcollections.
	Delete(ctx context.Context, id string) error
}

// ImageRepository is the repository port for list image records.
//
// Image policy:
// - Backend stores only Firebase Storage download URL.
// - Backend does not manage objectPath, fileName, contentType, or size.
// - Image record is scoped by listId.
// - imageID alone should not be used as a global lookup key.
type ImageRepository interface {
	// Query.
	ListByListID(ctx context.Context, listID string) ([]ListImage, error)

	// Read.
	GetByListIDAndID(ctx context.Context, listID string, imageID string) (ListImage, error)

	// Write.
	Create(ctx context.Context, img ListImage) (ListImage, error)
	Update(ctx context.Context, listID string, imageID string, patch ListImagePatch) (ListImage, error)

	// Delete physically deletes a list image record.
	Delete(ctx context.Context, listID string, imageID string) error
}
