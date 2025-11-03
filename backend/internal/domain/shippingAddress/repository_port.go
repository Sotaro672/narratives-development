package shippingAddress

import (
    "context"
    "errors"
    "time"
)

// Note: ShippingAddress はドメインエンティティです（entity.go 等で定義されている想定）。
// このファイルでは再定義しません。

// ========================================
// 外部DTO（API/ストレージとの境界）
// ========================================

type ShippingAddressDTO struct {
	ID            string  `json:"id"`
	UserID        string  `json:"userId"`
	RecipientName string  `json:"recipientName"`
	PostalCode    string  `json:"postalCode"`
	Prefecture    string  `json:"prefecture"`
	City          string  `json:"city"`
	AddressLine1  string  `json:"addressLine1"`
	AddressLine2  *string `json:"addressLine2,omitempty"`
	Country       string  `json:"country"`
	PhoneNumber   *string `json:"phoneNumber,omitempty"`
	IsDefault     bool    `json:"isDefault"`
	CreatedAt     string  `json:"createdAt"` // ISO8601
	UpdatedAt     string  `json:"updatedAt"` // ISO8601
}

// ========================================
// 入出力DTO（UseCase/Service -> Repository）
// ========================================

type CreateShippingAddressInput struct {
	UserID        string  `json:"userId"`
	RecipientName string  `json:"recipientName"`
	PostalCode    string  `json:"postalCode"`
	Prefecture    string  `json:"prefecture"`
	City          string  `json:"city"`
	AddressLine1  string  `json:"addressLine1"`
	AddressLine2  *string `json:"addressLine2,omitempty"`
	Country       string  `json:"country"`
	PhoneNumber   *string `json:"phoneNumber,omitempty"`
	IsDefault     bool    `json:"isDefault"`
}

type UpdateShippingAddressInput struct {
	RecipientName *string `json:"recipientName,omitempty"`
	PostalCode    *string `json:"postalCode,omitempty"`
	Prefecture    *string `json:"prefecture,omitempty"`
	City          *string `json:"city,omitempty"`
	AddressLine1  *string `json:"addressLine1,omitempty"`
	AddressLine2  *string `json:"addressLine2,omitempty"`
	Country       *string `json:"country,omitempty"`
	PhoneNumber   *string `json:"phoneNumber,omitempty"`
	IsDefault     *bool   `json:"isDefault,omitempty"`
}

// ========================================
// 検索条件/ソート/ページング（契約のみ）
// ========================================

type Filter struct {
    // 識別子
    ID     string
    UserID string

    // 住所系
    City    string
    State   string
    ZipCode string
    Country string

    // 期間
    CreatedFrom *time.Time
    CreatedTo   *time.Time
    UpdatedFrom *time.Time
    UpdatedTo   *time.Time
}

type Sort struct {
    Column SortColumn
    Order  SortOrder
}

type SortColumn string

const (
    SortByCreatedAt SortColumn = "createdAt"
    SortByUpdatedAt SortColumn = "updatedAt"
    SortByCity      SortColumn = "city"
    SortByState     SortColumn = "state"
    SortByZipCode   SortColumn = "zipCode"
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
    Items      []ShippingAddress
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
    GetByID(ctx context.Context, id string) (*ShippingAddress, error)
    List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
    Count(ctx context.Context, filter Filter) (int, error)

    // 変更系
    Create(ctx context.Context, in CreateShippingAddressInput) (*ShippingAddress, error)
    Update(ctx context.Context, id string, in UpdateShippingAddressInput) (*ShippingAddress, error)
    Delete(ctx context.Context, id string) error

    // 管理（開発/テスト用）
    Reset(ctx context.Context) error

    // 任意: トランザクション境界
    WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// 共通エラー（契約）
var (
    ErrNotFound = errors.New("shippingAddress: not found")
    ErrConflict = errors.New("shippingAddress: conflict")
)
