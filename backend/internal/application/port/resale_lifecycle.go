// backend/internal/application/port/resale_lifecycle.go
package port

import "context"

// ResaleProductListingLockReleaser releases the active listing lock held by a
// completed resale.
//
// The lock represents only the currently active resale listing for one physical
// product. Historical sold resale records must remain persisted independently
// from this lock.
//
// Implementations must be idempotent and safe against stale completion calls:
//
// - If no lock exists for productID, return nil.
// - If the current lock belongs to resaleID, remove it.
// - If the current lock belongs to another resale, do not remove it and return
//   nil.
//
// The resaleID ownership check is required so a delayed completion or retry from
// an older resale can never remove the lock of a newer resale listing.
type ResaleProductListingLockReleaser interface {
	ReleaseProductListingLock(
		ctx context.Context,
		productID string,
		resaleID string,
	) error
}
