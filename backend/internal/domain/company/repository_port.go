// backend/internal/domain/company/repository_port.go
package company

import (
	"context"
	"errors"
	"time"
)

// Patch（部分更新）: nil のフィールドは更新しない
type CompanyPatch struct {
	Name     *string
	Admin    *string
	IsActive *bool

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("company: not found")
	ErrConflict = errors.New("company: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// ID
	NewID(ctx context.Context) (string, error)

	// 取得
	GetByID(ctx context.Context, id string) (Company, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 変更
	Create(ctx context.Context, c Company) (Company, error)
	Update(ctx context.Context, id string, patch CompanyPatch) (Company, error)
	Delete(ctx context.Context, id string) error
}
