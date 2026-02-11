// backend/internal/domain/billingAddress/repository_port.go
package billingAddress

import (
	"context"
	"errors"
	"time"
)

// 契約（インターフェース）のみ定義。実装はインフラ層に委譲します。

// Create input (IDは実装側で採番可)
// ✅ entity.go（billing_address.dart 入力）準拠:
// - cardNumber（クレジットカード番号）
// - cardholderName（契約者名義）
// - cvc（裏の3桁コード）
type CreateBillingAddressInput struct {
	UserID         string     `json:"userId"`
	CardNumber     string     `json:"cardNumber"`
	CardholderName string     `json:"cardholderName"`
	CVC            string     `json:"cvc"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

// Update input (部分更新)
type UpdateBillingAddressInput struct {
	CardNumber     *string    `json:"cardNumber,omitempty"`
	CardholderName *string    `json:"cardholderName,omitempty"`
	CVC            *string    `json:"cvc,omitempty"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

// Repository Port（契約のみ）
// ✅ 旧式互換（GetDefaultByUser / SetDefault / List / Count / Filter / Sort / Page）は削除
type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*BillingAddress, error)
	GetByUser(ctx context.Context, userID string) ([]BillingAddress, error)

	// 変更系
	Create(ctx context.Context, in CreateBillingAddressInput) (*BillingAddress, error)
	Update(ctx context.Context, id string, in UpdateBillingAddressInput) (*BillingAddress, error)
	Delete(ctx context.Context, id string) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("billingAddress: not found")
	ErrConflict = errors.New("billingAddress: conflict")
)
