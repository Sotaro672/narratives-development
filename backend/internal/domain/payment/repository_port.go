// backend\internal\domain\payment\repository_port.go
package payment

import (
	"context"
	"errors"
	"time"
)

// PaymentMethod - 支払い方法（必要に応じて定義を拡張）
type PaymentMethod string

// CreatePaymentInput - 支払い作成入力（ドメイン契約）
//
// ✅ docId = invoiceId のため、ID は受け取らない
type CreatePaymentInput struct {
	InvoiceID        string        `json:"invoiceId"`
	BillingAddressID string        `json:"billingAddressId"`
	Amount           int           `json:"amount"`
	Status           PaymentStatus `json:"status"`
	ErrorType        *string       `json:"errorType,omitempty"`
}

// UpdatePaymentInput - Payment部分更新（nilは未更新）
//
// ✅ docId = invoiceId のため、Update は invoiceId をキーに行う前提。
//
//	また、Payment.Entity から ID/UpdatedAt/DeletedAt を削除したため、
//	フィルタ/更新項目からもそれらに依存する要素を削除。
type UpdatePaymentInput struct {
	// InvoiceID は docId と同一のため更新不可（必要なら “移設” 扱いで別操作にする）
	BillingAddressID *string        `json:"billingAddressId,omitempty"`
	Amount           *int           `json:"amount,omitempty"`
	Status           *PaymentStatus `json:"status,omitempty"`
	ErrorType        *string        `json:"errorType,omitempty"` // 空文字は実装側でnull化など判断
}

// Filter - 検索条件
//
// ✅ entity.go 準拠（Updated/Deleted 系を削除）
type Filter struct {
	InvoiceID        string
	BillingAddressID string
	Statuses         []PaymentStatus
	ErrorType        string

	MinAmount *int
	MaxAmount *int

	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// Sort - 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByAmount    SortColumn = "amount"
	SortByStatus    SortColumn = "status"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Page - ページ指定
type Page struct {
	Number  int
	PerPage int
}

// PageResult - ページ結果
type PageResult struct {
	Items      []Payment
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// RepositoryPort - ドメインのリポジトリ契約
//
// ✅ docId = invoiceId を前提に、ID 引数は invoiceId として扱うよう整理。
//
//	（互換のため引数名は id のままでも良いが、ここでは明確化のため invoiceId に統一）
type RepositoryPort interface {
	// 取得
	GetByInvoiceID(ctx context.Context, invoiceID string) (*Payment, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, in CreatePaymentInput) (*Payment, error)
	UpdateByInvoiceID(ctx context.Context, invoiceID string, patch UpdatePaymentInput) (*Payment, error)
	DeleteByInvoiceID(ctx context.Context, invoiceID string) error

	// 開発/テスト補助
	Reset(ctx context.Context) error
}

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
)

// 共通エラー
var (
	ErrNotFound = errors.New("payment: not found")
	ErrConflict = errors.New("payment: conflict")
)
