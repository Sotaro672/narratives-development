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
// 表示名
// ========================================

// FormatLastFirst は lastName -> firstName の順で表示名を組み立てます。
func FormatLastFirst(lastName string, firstName string) string {
	if lastName == "" {
		return firstName
	}
	if firstName == "" {
		return lastName
	}
	return lastName + " " + firstName
}

// FormatName は User の LastName -> FirstName の順で表示名を組み立てます。
func FormatName(u *User) string {
	if u == nil {
		return ""
	}
	return FormatLastFirst(u.LastName, u.FirstName)
}

// ========================================
// Repository Port（契約のみ）
// ========================================

type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*User, error)

	// 変更系
	//
	// Create は docId = uid を必ず指定する契約
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
