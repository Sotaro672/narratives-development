// backend\internal\domain\billingAddress\repository_port.go
package billingAddress

import (
	"context"
	"errors"
	"time"
)

// 契約（インターフェース）のみ定義。実装はインフラ層に委譲します。

// Create input (IDは実装側で採番可)
type CreateBillingAddressInput struct {
	UserID        string     `json:"userId"`
	BillingType   string     `json:"billingType"`
	NameOnAccount *string    `json:"nameOnAccount,omitempty"`
	CardBrand     *string    `json:"cardBrand,omitempty"`
	CardLast4     *string    `json:"cardLast4,omitempty"`
	CardExpMonth  *int       `json:"cardExpMonth,omitempty"`
	CardExpYear   *int       `json:"cardExpYear,omitempty"`
	CardToken     *string    `json:"cardToken,omitempty"`
	PostalCode    *int       `json:"postalCode,omitempty"`
	State         *string    `json:"state,omitempty"`
	City          *string    `json:"city,omitempty"`
	Street        *string    `json:"street,omitempty"`
	Country       *string    `json:"country,omitempty"`
	IsDefault     bool       `json:"isDefault"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

// Update input (部分更新)
type UpdateBillingAddressInput struct {
	BillingType   *string    `json:"billingType,omitempty"`
	NameOnAccount *string    `json:"nameOnAccount,omitempty"`
	CardBrand     *string    `json:"cardBrand,omitempty"`
	CardLast4     *string    `json:"cardLast4,omitempty"`
	CardExpMonth  *int       `json:"cardExpMonth,omitempty"`
	CardExpYear   *int       `json:"cardExpYear,omitempty"`
	CardToken     *string    `json:"cardToken,omitempty"`
	PostalCode    *int       `json:"postalCode,omitempty"`
	State         *string    `json:"state,omitempty"`
	City          *string    `json:"city,omitempty"`
	Street        *string    `json:"street,omitempty"`
	Country       *string    `json:"country,omitempty"`
	IsDefault     *bool      `json:"isDefault,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

// クエリ条件
type Filter struct {
	IDs          []string
	UserIDs      []string
	BillingTypes []string
	CardBrands   []string
	IsDefault    *bool

	PostalCodeMin *int
	PostalCodeMax *int

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	NameLike *string // name_on_account の部分一致など、実装依存
	CityLike *string
}

// 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt   SortColumn = "createdAt"
	SortByUpdatedAt   SortColumn = "updatedAt"
	SortByBillingType SortColumn = "billingType"
	SortByIsDefault   SortColumn = "isDefault"
	SortByPostalCode  SortColumn = "postalCode"
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
	GetDefaultByUser(ctx context.Context, userID string) (*BillingAddress, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, in CreateBillingAddressInput) (*BillingAddress, error)
	Update(ctx context.Context, id string, in UpdateBillingAddressInput) (*BillingAddress, error)
	Delete(ctx context.Context, id string) error

	// ユーザーのデフォルト住所設定（他の住所のis_defaultを落とす規約は実装側で担保）
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
