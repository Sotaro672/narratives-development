// backend/internal/domain/cart/repository_port.go
package cart

import "context"

// Repository is a persistence port for Cart.
//
// Storage recommendation (Firestore):
// - collection: carts
// - docId: avatarId
// - fields: avatarId, items(map), createdAt, updatedAt, expiresAt
//
// TTL:
// - Configure Firestore TTL on the "expiresAt" field.
// - expiresAt should be refreshed on each cart mutation (handled by domain via touch()).
type Repository interface {
	// GetByAvatarID returns the cart for the avatar.
	// Not-found handling policy:
	// - If your infra layer has ErrNotFound, return (nil, ErrNotFound)
	// - Or return (nil, nil) and let application layer treat nil as "empty cart"
	GetByAvatarID(ctx context.Context, avatarID string) (*Cart, error)

	// Upsert saves the cart (create or update).
	// Implementations may enforce optimistic concurrency.
	Upsert(ctx context.Context, c *Cart) error

	// DeleteByAvatarID deletes the cart for the avatar (e.g., after order).
	DeleteByAvatarID(ctx context.Context, avatarID string) error
}
