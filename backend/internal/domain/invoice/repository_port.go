package invoice

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
type InvoicePatch struct {
	// 金額系
	Subtotal       *int
	DiscountAmount *int
	TaxAmount      *int
	ShippingCost   *int
	TotalAmount    *int

	// その他
	Currency         *string
	BillingAddressID *string

	UpdatedAt *time.Time
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	SearchQuery string // orderId, currency などへの部分一致等は実装側で解釈

	OrderID  *string
	OrderIDs []string
	Currency *string

	// 金額レンジ（いずれか、または複数指定可）
	MinSubtotal       *int
	MaxSubtotal       *int
	MinDiscountAmount *int
	MaxDiscountAmount *int
	MinTaxAmount      *int
	MaxTaxAmount      *int
	MinShippingCost   *int
	MaxShippingCost   *int
	MinTotalAmount    *int
	MaxTotalAmount    *int

	// 日付レンジ
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
}

// 共通型エイリアス（インフラ非依存）
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("invoice: not found")
	ErrConflict = errors.New("invoice: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Invoice], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Invoice], error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 取得
	GetByOrderID(ctx context.Context, orderID string) (Invoice, error)
	Exists(ctx context.Context, orderID string) (bool, error)

	// 変更
	Create(ctx context.Context, inv Invoice) (Invoice, error)
	Update(ctx context.Context, orderID string, patch InvoicePatch) (Invoice, error)
	Delete(ctx context.Context, orderID string) error

	// 任意: Upsert 等
	Save(ctx context.Context, inv Invoice, opts *SaveOptions) (Invoice, error)

	// 補助: 注文アイテム単位の請求情報
	GetOrderItemInvoiceByOrderItemID(ctx context.Context, orderItemID string) (OrderItemInvoice, error)
	ListOrderItemInvoicesByOrderItemIDs(ctx context.Context, orderItemIDs []string) ([]OrderItemInvoice, error)
}
