package tokenBlueprint

import (
	"context"
	"io"
	"time"
)

// 契約（インターフェース）のみを定義します。
// エンティティ TokenBlueprint は同パッケージの entity.go を参照してください。

// Create 用入力（IDはリポジトリ側で採番可。CreatedAt/UpdatedAtはnilなら実装側で付与可）
type CreateTokenBlueprintInput struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	Description string `json:"description"`
	// IconID points to tokenIcon.TokenIcon primary key (token_icons.id). Optional.
	IconID       *string  `json:"iconId,omitempty"`
	ContentFiles []string `json:"contentFiles"`
	AssigneeID   string   `json:"assigneeId"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`
}

// 部分更新入力（nilは未更新）
// DeletedAt/DeletedBy はソフトデリート用途（Deleteで物理削除する実装でも可）
type UpdateTokenBlueprintInput struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`
	// IconID（空文字の扱いはユースケースで決定：null化等）
	IconID       *string   `json:"iconId,omitempty"`
	ContentFiles *[]string `json:"contentFiles,omitempty"`
	AssigneeID   *string   `json:"assigneeId,omitempty"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// フィルタ/検索条件
type Filter struct {
	IDs         []string
	BrandIDs    []string
	AssigneeIDs []string
	Symbols     []string

	NameLike   string // 部分一致
	SymbolLike string // 部分一致
	HasIcon    *bool  // nil=全件, true=アイコンあり, false=なし

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
	SortByName      SortColumn = "name"
	SortBySymbol    SortColumn = "symbol"
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
	Items      []TokenBlueprint
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// RepositoryPort はトークン設計ドメインのリポジトリ境界です。
type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*TokenBlueprint, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, in CreateTokenBlueprintInput) (*TokenBlueprint, error)
	Update(ctx context.Context, id string, in UpdateTokenBlueprintInput) (*TokenBlueprint, error)
	Delete(ctx context.Context, id string) error

	// 一意性チェック
	IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error)
	IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error)

	// ストレージ（アイコン/コンテンツのアップロード）
	UploadIcon(ctx context.Context, fileName, contentType string, r io.Reader) (url string, err error)
	UploadContentFile(ctx context.Context, fileName, contentType string, r io.Reader) (url string, err error)

	// トランザクション境界（任意）
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// 開発/テスト補助（任意）
	Reset(ctx context.Context) error
}
