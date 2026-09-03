// backend/internal/application/usecase/cart_item_cleanup.go
package usecase

import "context"

// CartItemCleanup removes cart items that became unavailable because the
// underlying List or Resale was suspended or deleted.
type CartItemCleanup interface {
	RemoveItemsByListID(ctx context.Context, listID string) error
	RemoveItemsByResaleID(ctx context.Context, resaleID string) error
}
