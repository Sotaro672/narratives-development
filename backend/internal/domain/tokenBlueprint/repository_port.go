// backend/internal/domain/tokenBlueprint/repository_port.go
package tokenBlueprint

import (
	"context"
	"time"
)

// ===============================
// Create 用入力
// ===============================
//
// entity.go 正:
// - iconId は存在しない（保持しない）
// - contentFiles は []ContentFile（embedded）
// - minted は create 時は常に false（入力として持たせても repo 側で無視/固定化して良い）
// - metadataUri は任意
type CreateTokenBlueprintInput struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	ContentFiles []ContentFile `json:"contentFiles"`
	AssigneeID   string        `json:"assigneeId"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`

	MetadataURI string `json:"metadataUri,omitempty"`
}

// ===============================
// Update 用入力
// ===============================
//
// entity.go 正:
// - iconId は存在しない
// - contentFiles は []ContentFile の全置換
// - minted は bool
// - metadataUri は任意
type UpdateTokenBlueprintInput struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`

	ContentFiles *[]ContentFile `json:"contentFiles,omitempty"` // 全置換
	AssigneeID   *string        `json:"assigneeId,omitempty"`
	Minted       *bool          `json:"minted,omitempty"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`

	MetadataURI *string `json:"metadataUri,omitempty"`
}

// ===============================
// Patch（表示用）
// ===============================
//
// entity.go 正に寄せる:
// - Patch は read-model 用の最小情報（string/bool を基本）
// - icon は objectPath 規約＋metadata解決で返すため、ここでは IconURL だけを保持（任意）
// - metadataUri を含める
type Patch struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description"`
	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`
	IconURL     string `json:"iconUrl,omitempty"`
}

// ===============================
// Filter（検索条件）
// ===============================
//
// entity.go 正:
// - iconId が無いので HasIcon は廃止
type Filter struct {
	IDs         []string
	BrandIDs    []string
	CompanyIDs  []string
	AssigneeIDs []string
	Symbols     []string

	NameLike   string
	SymbolLike string

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

	// ★ Patch 取得（read-model 用）
	GetPatchByID(ctx context.Context, id string) (Patch, error)

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
		if !tb.Minted {
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
		if tb.Minted {
			items = append(items, tb)
		}
	}

	result.Items = items
	result.TotalCount = len(items)
	result.TotalPages = 1

	return result, nil
}
