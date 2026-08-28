// backend/internal/application/port/list_save_operation.go
package port

import (
	"context"
	"time"
)

// ListSaveOperationStorage defines storage operations required by the
// ListSaveOperation application flow.
type ListSaveOperationStorage interface {
	Exists(
		ctx context.Context,
		storagePath string,
	) (bool, error)

	Delete(
		ctx context.Context,
		storagePath string,
	) error
}

// ListSaveOperationRetryQueue defines asynchronous retry scheduling required by
// the ListSaveOperation application flow.
type ListSaveOperationRetryQueue interface {
	EnqueueRetry(
		ctx context.Context,
		operationID string,
		scheduledAt time.Time,
	) error
}
