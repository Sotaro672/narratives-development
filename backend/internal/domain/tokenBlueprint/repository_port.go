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
	Name         string     `json:"name"`
	Symbol       string     `json:"symbol"`
	BrandID      string     `json:"brandId"`
	CompanyID    string     `json:"companyId"`
	Description  string     `json:"description"`
	IconID       *string    `json:"iconId,omitempty"`
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
	Minted       bool       `json:"minted"` // ★ boolean に変更（create時は常に false を想定）
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	CreatedBy    string     `json:"createdBy"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string     `json:"updatedBy"`
}

// ===============================
// Update 用入力
// ===============================
type UpdateTokenBlueprintInput struct {
	Name         *string    `json:"name,omitempty"`
	Symbol       *string    `json:"symbol,omitempty"`
	BrandID      *string    `json:"brandId,omitempty"`
	Description  *string    `json:"description,omitempty"`
	IconID       *string    `json:"iconId,omitempty"`
	ContentFiles *[]string  `json:"contentFiles,omitempty"`
	AssigneeID   *string    `json:"assigneeId,omitempty"`
	Minted       *bool      `json:"minted,omitempty"` // ★ boolean に変更
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    *string    `json:"updatedBy,omitempty"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	DeletedBy    *string    `json:"deletedBy,omitempty"`

	// ★ 追加: metadataUri 用のポインタフィールド
	MetadataURI *string `json:"metadataUri,omitempty"`
}

// ===============================
// Patch（表示用）
// ===============================
//
// TokenBlueprintCard に表示するための最小情報。
// - inventory/detail 側の ViewModel に埋め込む用途を想定
// - 取得できない項目は nil のままでも OK（フロントで "" へフォールバック可能）
type Patch struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	BrandName   *string `json:"brandName,omitempty"`
	CompanyID   *string `json:"companyId,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`
	Description *string `json:"description,omitempty"`
	IconURL     *string `json:"iconUrl,omitempty"`
	Minted      *bool   `json:"minted,omitempty"`
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

// ===============================
// Helper Functions
// ===============================

// ★ brandId ごとに一覧取得
func ListByBrandID(
	ctx context.Context,
	repo RepositoryPort,
	brandID string,
	page Page,
) (PageResult, error) {

	f := Filter{
		BrandIDs: []string{brandID},
	}
	return repo.List(ctx, f, page)
}

// ==========================================================
// ★ minted = false（notYet） のみを一覧取得
// ==========================================================
func ListMintedNotYet(
	ctx context.Context,
	repo RepositoryPort,
	page Page,
) (PageResult, error) {

	// mint 状態で DB フィルタできないため後段でフィルタ
	result, err := repo.List(ctx, Filter{}, page)
	if err != nil {
		return PageResult{}, err
	}

	items := []TokenBlueprint{}
	for _, tb := range result.Items {
		if !tb.Minted { // false = notYet
			items = append(items, tb)
		}
	}

	result.Items = items
	result.TotalCount = len(items)
	result.TotalPages = 1 // メモリ内フィルタなので 1 とする

	return result, nil
}

// ==========================================================
// ★ minted = true（minted） のみ一覧取得
// ==========================================================
func ListMintedCompleted(
	ctx context.Context,
	repo RepositoryPort,
	page Page,
) (PageResult, error) {

	result, err := repo.List(ctx, Filter{}, page)
	if err != nil {
		return PageResult{}, err
	}

	items := []TokenBlueprint{}
	for _, tb := range result.Items {
		if tb.Minted { // true = minted
			items = append(items, tb)
		}
	}

	result.Items = items
	result.TotalCount = len(items)
	result.TotalPages = 1

	return result, nil
}
