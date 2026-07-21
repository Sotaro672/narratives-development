// backend/internal/domain/list/repository_port.go
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
// Model filter policy:
// - ModelIDs contains modelId values.
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
	ModelIDs []string
	MinPrice *int
	MaxPrice *int

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

// SaveOperationRetryFilter is the query condition used by retry workers.
//
// Retry workers should normally select failed_retryable operations whose
// UpdatedAt is earlier than or equal to UpdatedBefore.
type SaveOperationRetryFilter struct {
	Statuses []SaveOperationStatus

	UpdatedBefore *time.Time

	Limit int
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

	ErrSaveOperationNotFound = errors.New(
		"list save operation: not found",
	)

	ErrSaveOperationConflict = errors.New(
		"list save operation: conflict",
	)

	ErrSaveOperationIdempotencyConflict = errors.New(
		"list save operation: idempotency key conflict",
	)
)

// Repository is the repository port for List aggregate root.
type Repository interface {
	// List query.
	List(
		ctx context.Context,
		filter Filter,
		sort Sort,
		page Page,
	) (PageResult[List], error)

	ListByCursor(
		ctx context.Context,
		filter Filter,
		sort Sort,
		cpage CursorPage,
	) (CursorPageResult[List], error)

	// Read.
	GetByID(
		ctx context.Context,
		id string,
	) (List, error)

	ListByInventoryID(
		ctx context.Context,
		inventoryID string,
	) ([]List, error)

	// Write.
	Create(
		ctx context.Context,
		l List,
	) (List, error)

	// Update replaces/upserts the mutable fields of a persisted List.
	//
	// Expected implementation policy:
	// - id is the target document id.
	// - l.ID may be empty or equal to id.
	// - immutable fields such as InventoryID, CreatedAt, CreatedBy should not
	//   be overwritten unless the implementation intentionally treats Update
	//   as full replacement.
	Update(
		ctx context.Context,
		id string,
		l List,
	) (List, error)

	// Delete physically deletes a list document.
	// Implementations may also delete child image records if the storage
	// supports subcollections.
	Delete(
		ctx context.Context,
		id string,
	) error
}

// ImageRepository is the repository port for list image records.
//
// Image policy:
//   - Backend stores only Firebase Storage download URL.
//   - Backend does not store objectPath, fileName, contentType, or size in
//     ListImage.
//   - StoragePath may be retained only by SaveOperation for compensation.
//   - Image records are scoped by listId.
//   - imageID alone must not be used as a global lookup key.
//
// Idempotency policy:
//   - Create must use listID + imageID as the unique scoped key.
//   - Repeating Create with the same equivalent record should return the
//     existing record.
//   - Repeating Create with conflicting values should return ErrConflict.
type ImageRepository interface {
	// Query.
	GetByID(
		ctx context.Context,
		listID string,
		imageID string,
	) (ListImage, error)

	ListByListID(
		ctx context.Context,
		listID string,
	) ([]ListImage, error)

	// Write.
	Create(
		ctx context.Context,
		img ListImage,
	) (ListImage, error)

	Update(
		ctx context.Context,
		listID string,
		imageID string,
		patch ListImagePatch,
	) (ListImage, error)

	// Delete physically deletes a list image record.
	//
	// Delete should be idempotent. Deleting an already absent image record
	// should be treated as success.
	Delete(
		ctx context.Context,
		listID string,
		imageID string,
	) error
}

// SaveOperationRepository persists the List save Saga state.
//
// Idempotency policy:
//   - IdempotencyKey must be unique.
//   - Create with an unused IdempotencyKey creates a new operation.
//   - Create with an existing equivalent IdempotencyKey returns the existing
//     operation.
//   - Create with an existing key for a different request returns
//     ErrSaveOperationIdempotencyConflict.
//
// Concurrency policy:
// - Update uses optimistic concurrency control.
// - expectedVersion must match the persisted Version.
// - A successful update increments Version.
// - A version mismatch returns ErrSaveOperationConflict.
type SaveOperationRepository interface {
	// Create persists a new pending save operation.
	Create(
		ctx context.Context,
		operation SaveOperation,
	) (SaveOperation, error)

	// GetByID returns a save operation by operation ID.
	GetByID(
		ctx context.Context,
		operationID string,
	) (SaveOperation, error)

	// GetByIdempotencyKey returns the operation associated with the key.
	GetByIdempotencyKey(
		ctx context.Context,
		idempotencyKey string,
	) (SaveOperation, error)

	// Update persists the current operation state using optimistic locking.
	//
	// expectedVersion is the Version value read before applying the update.
	// The returned operation must contain the incremented Version.
	Update(
		ctx context.Context,
		operation SaveOperation,
		expectedVersion int64,
	) (SaveOperation, error)

	// ListRetryable returns operations eligible for retry processing.
	//
	// Results should be ordered by UpdatedAt ascending so older failures are
	// retried first.
	ListRetryable(
		ctx context.Context,
		filter SaveOperationRetryFilter,
	) ([]SaveOperation, error)
}
