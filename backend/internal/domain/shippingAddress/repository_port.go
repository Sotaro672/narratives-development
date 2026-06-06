// backend/internal/domain/shippingAddress/repository_port.go
package shippingAddress

import (
	"context"
	"errors"
)

// ========================================
// Repository Port
// - entity.go (ShippingAddress) を single source とする
// - docId = ShippingAddress.ID = UUID
// - UserID = owner uid
// - docId と UserID は異なる
// - 1 user can have many shipping addresses
// ========================================

type RepositoryPort interface {
	// Read
	GetByID(ctx context.Context, id string) (*ShippingAddress, error)
	Exists(ctx context.Context, id string) (bool, error)
	ListByUserID(ctx context.Context, userID string) ([]ShippingAddress, error)

	// Write
	// Create creates a new document.
	// - ShippingAddress.ID must already be assigned by usecase.
	// - If the document already exists, return ErrConflict.
	Create(ctx context.Context, a ShippingAddress) (*ShippingAddress, error)

	// Update updates an existing document.
	// - No upsert.
	// - If the document does not exist, return ErrNotFound.
	Update(ctx context.Context, a ShippingAddress) (*ShippingAddress, error)

	Delete(ctx context.Context, id string) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("shippingAddress: not found")
	ErrConflict = errors.New("shippingAddress: conflict")
)
