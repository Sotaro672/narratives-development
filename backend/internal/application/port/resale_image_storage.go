// backend/internal/application/port/resale_image_storage.go
package port

import "context"

// ResaleImageStorage manages storage objects associated with a resale.
type ResaleImageStorage interface {
	DeleteAll(
		ctx context.Context,
		resaleID string,
	) error
}
