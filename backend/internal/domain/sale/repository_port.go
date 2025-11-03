package sale

import (
    "context"
    "errors"
)

// ========================================
// 外部DTO（必要なら利用）
// ========================================

type SaleDTO struct {
    ID          string  `json:"id"`
    ListID      string  `json:"listId"`
    MemberID    string  `json:"memberId"`
    ModelNumber string  `json:"modelNumber"`
    Price       float64 `json:"price"`
    Currency    string  `json:"currency"`
    Status      string  `json:"status"`
    CreatedAt   string  `json:"createdAt"` // ISO8601
    UpdatedAt   string  `json:"updatedAt"` // ISO8601
}

// ========================================
// 入出力DTO（UseCase/Service -> Repository）
// ========================================

type CreateSaleInput struct {
    ListID     string      `json:"listId"`
    DiscountID *string     `json:"discountId,omitempty"`
    Prices     []SalePrice `json:"prices"` // 少なくとも1件
}

type UpdateSaleInput struct {
    ListID     *string      `json:"listId,omitempty"`
    DiscountID *string      `json:"discountId,omitempty"` // 空文字→null化等はユースケースで吸収
    Prices     *[]SalePrice `json:"prices,omitempty"`
}

// ========================================
// 価格・在庫関連（必要に応じて利用）
// ========================================

type ModelPriceInfo struct {
    ModelNumber string  `json:"modelNumber"`
    Price       float64 `json:"price"`
}

type PriceResult struct {
    SaleID      string   `json:"saleId"`
    ModelNumber string   `json:"modelNumber"`
    Price       *float64 `json:"price,omitempty"`
    Error       *string  `json:"error,omitempty"`
}

type ModelInventoryInfo struct {
    ModelNumber string                 `json:"modelNumber"`
    Attrs       map[string]interface{} `json:"attrs,omitempty"`
}

// ========================================
// 検索条件/ソート/ページング（契約のみ）
// ========================================

type Filter struct {
    // 識別子
    ID     string
    ListID string

    // 割引の有無（nil: 全件, true: 割引あり, false: 割引なし）
    HasDiscount *bool

    // この modelNumber の価格を持つ Sale を検索
    ModelNumber string

    // 価格帯でのフィルタ（いずれかの価格が範囲に入るもの）
    MinAnyPrice *int
    MaxAnyPrice *int
}

type Sort struct {
    Column SortColumn
    Order  SortOrder
}

type SortColumn string

const (
    SortByID     SortColumn = "id"
    SortByListID SortColumn = "listId"
)

type SortOrder string

const (
    SortAsc  SortOrder = "asc"
    SortDesc SortOrder = "desc"
)

type Page struct {
    Number  int
    PerPage int
}

type PageResult struct {
    Items      []Sale
    TotalCount int
    TotalPages int
    Page       int
    PerPage    int
}

// ========================================
// Repository Port（契約のみ）
// ========================================

type RepositoryPort interface {
    // 取得系
    GetByID(ctx context.Context, id string) (*Sale, error)
    List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
    Count(ctx context.Context, filter Filter) (int, error)

    // 変更系
    Create(ctx context.Context, in CreateSaleInput) (*Sale, error)
    Update(ctx context.Context, id string, in UpdateSaleInput) (*Sale, error)
    Delete(ctx context.Context, id string) error

    // 開発/テスト補助（任意）
    Reset(ctx context.Context) error
}

// 共通エラー（契約）
var (
    ErrNotFound = errors.New("sale: not found")
    ErrConflict = errors.New("sale: conflict")
)
