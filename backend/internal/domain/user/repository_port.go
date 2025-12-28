package user

import (
	"context"
	"errors"
	"time"
)

// 契約（インターフェース）のみを定義します。
// エンティティ User は同パッケージの entity.go を参照してください。

// ========================================
// 入出力（契約のみ）
// ========================================

type CreateUserInput struct {
	FirstName     *string `json:"first_name,omitempty"`
	FirstNameKana *string `json:"first_name_kana,omitempty"`
	LastNameKana  *string `json:"last_name_kana,omitempty"`
	LastName      *string `json:"last_name,omitempty"`

	// nil の場合は実装側で現在時刻を付与可
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type UpdateUserInput struct {
	FirstName     *string `json:"first_name,omitempty"`
	FirstNameKana *string `json:"first_name_kana,omitempty"`
	LastNameKana  *string `json:"last_name_kana,omitempty"`
	LastName      *string `json:"last_name,omitempty"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ========================================
// 検索条件/ソート/ページング（契約のみ）
// ========================================

type Filter struct {
	IDs []string

	FirstNameLike string
	LastNameLike  string
	NameLike      string // 氏名のあいまい検索に利用（実装に委ねる）

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByUpdatedAt SortColumn = "updatedAt"
	SortByDeletedAt SortColumn = "deletedAt"
	SortByFirstName SortColumn = "first_name"
	SortByLastName  SortColumn = "last_name"
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
	Items      []User
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
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, in CreateUserInput) (*User, error)
	Update(ctx context.Context, id string, in UpdateUserInput) (*User, error)
	Delete(ctx context.Context, id string) error

	// 管理（開発/テスト用）
	Reset(ctx context.Context) error

	// 任意: トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("user: not found")
	ErrConflict = errors.New("user: conflict")
)
