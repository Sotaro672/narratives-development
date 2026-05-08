// backend/internal/domain/user/repository_port.go
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

	// nil の場合は実装側で現在時刻を付与可（ただしサーバが正）
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"` // nil=未指定, zero=not deleted, non-zero=deleted
}

type UpdateUserInput struct {
	FirstName     *string `json:"first_name,omitempty"`
	FirstNameKana *string `json:"first_name_kana,omitempty"`
	LastNameKana  *string `json:"last_name_kana,omitempty"`
	LastName      *string `json:"last_name,omitempty"`

	// UpdatedAt は未指定なら実装側で NOW を付与可（ただしサーバが正）
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"` // nil=未指定, zero=not deleted, non-zero=deleted
}

// ========================================
// 検索条件/ページング（契約のみ）
// ※ sort 機能は削除
// ========================================

type Filter struct {
	IDs         []string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time
}

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

	// List: filter + paging のみ（sort は削除）
	List(ctx context.Context, filter Filter, page Page) (PageResult, error)

	// ✅ 画面表示用（best-effort）: userId -> "lastName firstName"
	// - 実装側で lastName / firstName を取得し、"姓 名" の順で返す
	// - 見つからない場合は ErrNotFound
	GetNameByID(ctx context.Context, id string) (string, error)

	// 変更系
	//
	// ✅ Create は docId = uid を必ず指定する契約
	// - /mall/me/users の設計と整合（uid は認証から得る）
	// - body の id を信用しない（spoof 防止）
	Create(ctx context.Context, id string, in CreateUserInput) (*User, error)

	Update(ctx context.Context, id string, in UpdateUserInput) (*User, error)
	Delete(ctx context.Context, id string) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("user: not found")
	ErrConflict = errors.New("user: conflict")
)
