package fulfillment

import (
	"context"
	"errors"
	"time"
)

// 契約（インターフェース）のみ定義。実装はインフラ層に委譲します。

// 入力DTO（IDは実装側で採番可）
type CreateFulfillmentInput struct {
	OrderID   string            `json:"orderId"`
	PaymentID string            `json:"paymentId"`
	Status    FulfillmentStatus `json:"status"`

	// nil の場合は実装側で現在時刻を補完可
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// 部分更新
type UpdateFulfillmentInput struct {
	Status    *FulfillmentStatus `json:"status,omitempty"`
	UpdatedAt *time.Time         `json:"updatedAt,omitempty"`
}

// 検索条件
type Filter struct {
	IDs         []string
	OrderIDs    []string
	PaymentIDs  []string
	Statuses    []FulfillmentStatus
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
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
	SortByStatus    SortColumn = "status"
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
	Items      []Fulfillment
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// Repository Port（契約）
type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*Fulfillment, error)
	GetByOrderID(ctx context.Context, orderID string) ([]Fulfillment, error)
	GetLatestByOrderID(ctx context.Context, orderID string) (*Fulfillment, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, in CreateFulfillmentInput) (*Fulfillment, error)
	Update(ctx context.Context, id string, in UpdateFulfillmentInput) (*Fulfillment, error)
	Delete(ctx context.Context, id string) error

	// 任意: トランザクション境界/メンテ
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	Reset(ctx context.Context) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("fulfillment: not found")
	ErrConflict = errors.New("fulfillment: conflict")
)
