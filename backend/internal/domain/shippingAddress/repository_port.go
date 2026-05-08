// backend/internal/domain/shippingAddress/repository_port.go
package shippingAddress

import (
	"context"
	"errors"
	"time"
)

// ========================================
// 入出力（契約のみ）
// - entity.go (ShippingAddress) を single source とする
// - docId = uid (= ShippingAddress.ID)
// - UserID も uid 固定
// ========================================

type UpsertShippingAddressInput struct {
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"` // optional (may be "")
	Country string `json:"country"` // required (if UI has no input, caller sets "JP" etc.)
}

type UpdateShippingAddressInput struct {
	ZipCode   *string    `json:"zipCode,omitempty"`
	State     *string    `json:"state,omitempty"`
	City      *string    `json:"city,omitempty"`
	Street    *string    `json:"street,omitempty"`
	Street2   *string    `json:"street2,omitempty"`
	Country   *string    `json:"country,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"` // server can fill
}

// ========================================
// Repository Port（契約のみ）
// ========================================

type RepositoryPort interface {
	// Read
	GetByID(ctx context.Context, id string) (*ShippingAddress, error)

	// Write
	// ✅ CreateWithID: docId を caller が指定（id=uid）
	// - 既存なら ErrConflict
	CreateWithID(ctx context.Context, id string, a ShippingAddress) (*ShippingAddress, error)

	// ✅ Update: docId を指定して上書き（必須フィールドは usecase 側で保証）
	Update(ctx context.Context, id string, a ShippingAddress) (*ShippingAddress, error)

	Delete(ctx context.Context, id string) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("shippingAddress: not found")
	ErrConflict = errors.New("shippingAddress: conflict")
)
