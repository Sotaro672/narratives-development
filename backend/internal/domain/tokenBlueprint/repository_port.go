package tokenBlueprint

import (
	"context"
	"io"
	"time"
)

// ===============================
// Create 用入力
// ===============================
type CreateTokenBlueprintInput struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"` // ★ 追加
	Description string `json:"description"`

	IconID       *string  `json:"iconId,omitempty"`
	ContentFiles []string `json:"contentFiles"`
	AssigneeID   string   `json:"assigneeId"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`
}

// ===============================
// Update 用入力
// ===============================
type UpdateTokenBlueprintInput struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`

	IconID       *string   `json:"iconId,omitempty"`
	ContentFiles *[]string `json:"contentFiles,omitempty"`
	AssigneeID   *string   `json:"assigneeId,omitempty"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// ===============================
// Filter（検索条件）
// ===============================
type Filter struct {
	IDs         []string
	BrandIDs    []string
	CompanyIDs  []string
	AssigneeIDs []string
	Symbols     []string

	NameLike   string
	SymbolLike string
	HasIcon    *bool

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
}

// ===============================
// Page / PageResult
// ===============================
type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []TokenBlueprint
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ===============================
// RepositoryPort（リポジトリ境界）
// ===============================
type RepositoryPort interface {
	// 単体取得
	GetByID(ctx context.Context, id string) (*TokenBlueprint, error)

	// ★ ID → Name の高速解決
	GetNameByID(ctx context.Context, id string) (string, error)

	// 一覧取得
	List(ctx context.Context, filter Filter, page Page) (PageResult, error)

	// 一覧件数
	Count(ctx context.Context, filter Filter) (int, error)

	// ★ companyId で限定した一覧
	ListByCompanyID(ctx context.Context, companyID string, page Page) (PageResult, error)

	// 作成・更新・削除
	Create(ctx context.Context, in CreateTokenBlueprintInput) (*TokenBlueprint, error)
	Update(ctx context.Context, id string, in UpdateTokenBlueprintInput) (*TokenBlueprint, error)
	Delete(ctx context.Context, id string) error

	// 一意性チェック
	IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error)
	IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error)

	// ストレージ
	UploadIcon(ctx context.Context, fileName, contentType string, r io.Reader) (url string, err error)
	UploadContentFile(ctx context.Context, fileName, contentType string, r io.Reader) (url string, err error)

	// トランザクション
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// テスト/開発用リセット
	Reset(ctx context.Context) error
}
