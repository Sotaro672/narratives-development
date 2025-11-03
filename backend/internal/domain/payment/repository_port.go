package payment

import (
	"context"
	"errors"
	"time"
)

// PaymentMethod - 支払い方法（必要に応じて定義を拡張）
type PaymentMethod string

// CreatePaymentInput - 支払い作成入力（ドメイン契約）
type CreatePaymentInput struct {
	InvoiceID        string        `json:"invoiceId"`
	BillingAddressID string        `json:"billingAddressId"`
	Amount           int           `json:"amount"`
	Status           PaymentStatus `json:"status"`
	ErrorType        *string       `json:"errorType,omitempty"`
}

// UpdatePaymentInput - Payment部分更新（nilは未更新）
type UpdatePaymentInput struct {
	InvoiceID        *string        `json:"invoiceId,omitempty"`
	BillingAddressID *string        `json:"billingAddressId,omitempty"`
	Amount           *int           `json:"amount,omitempty"`
	Status           *PaymentStatus `json:"status,omitempty"`
	ErrorType        *string        `json:"errorType,omitempty"` // 空文字は実装側でnull化など判断
}

// Filter - 検索条件
type Filter struct {
	ID               string
	InvoiceID        string
	BillingAddressID string
	Statuses         []PaymentStatus
	ErrorType        string

	MinAmount *int
	MaxAmount *int

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	// nil: 全件, true: 削除済のみ, false: 未削除のみ
	Deleted *bool
}

// Sort - 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByUpdatedAt SortColumn = "updatedAt"
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
type RepositoryPort interface {
	// 取得
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByInvoiceID(ctx context.Context, invoiceID string) ([]Payment, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, in CreatePaymentInput) (*Payment, error)
	Update(ctx context.Context, id string, patch UpdatePaymentInput) (*Payment, error)
	Delete(ctx context.Context, id string) error

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
