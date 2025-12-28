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

// クエリ条件（entity.go 準拠：最小）
// - 互換のためIDsは保持
// - 旧: billingType/cardBrand/isDefault/住所系フィールドは削除
type Filter struct {
	IDs     []string
	UserIDs []string

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// 部分一致（実装依存）
	CardholderNameLike *string
}

// 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByUpdatedAt SortColumn = "updatedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ページング
type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []BillingAddress
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// Repository Port（契約のみ）
type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*BillingAddress, error)
	GetByUser(ctx context.Context, userID string) ([]BillingAddress, error)

	// 互換: 旧契約に存在していたため残す（entity.go では default の概念なし）
	// 実装側では「最新1件」などの規約で返す。
	GetDefaultByUser(ctx context.Context, userID string) (*BillingAddress, error)

	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, in CreateBillingAddressInput) (*BillingAddress, error)
	Update(ctx context.Context, id string, in UpdateBillingAddressInput) (*BillingAddress, error)
	Delete(ctx context.Context, id string) error

	// 互換: 旧契約に存在していたため残す（entity.go では default の概念なし）
	// 実装側では no-op または独自規約で担保。
	SetDefault(ctx context.Context, id string) error

	// 任意: トランザクション境界/メンテ
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	Reset(ctx context.Context) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("billingAddress: not found")
	ErrConflict = errors.New("billingAddress: conflict")
)
