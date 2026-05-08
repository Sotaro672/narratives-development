// backend/internal/domain/tokenBlueprint/repository_port.go
package tokenBlueprint

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// ===============================
// Create 用入力
// ===============================
//
// Firebase Storage 移行後の正:
// - backend は GCS signed URL / upload endpoint を持たない
// - iconId は存在しない
// - tokenIconObjectPath / tokenContentsObjectPath は create input として扱わない
// - tokenBlueprintIcon は frontend が Firebase Storage へ直接 upload する
// - iconUrl は Firebase Storage の downloadURL を保存する
// - tokenBlueprintContents も frontend が Firebase Storage へ直接 upload する
// - contentFiles は []ContentFile embedded として保存する
// - contentFiles[].url には Firebase Storage の downloadURL を保存する
// - contentFiles[].objectPath には Firebase Storage 上の objectPath を保存する
// - minted は create 時は常に false
// - metadataUri は任意
type CreateTokenBlueprintInput struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	IconURL      string        `json:"iconUrl,omitempty"`
	ContentFiles []ContentFile `json:"contentFiles"`

	AssigneeID string `json:"assigneeId"`

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
// Firebase Storage 移行後の正:
// - iconUrl は Firebase Storage downloadURL
// - contentFiles は []ContentFile の全置換
// - contentFiles[].url は Firebase Storage downloadURL
// - contentFiles[].objectPath は Firebase Storage objectPath
// - backend は signed URL を発行しない
// - minted は bool
// - metadataUri は任意
type UpdateTokenBlueprintInput struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`

	IconURL      *string        `json:"iconUrl,omitempty"`
	ContentFiles *[]ContentFile `json:"contentFiles,omitempty"`

	AssigneeID *string `json:"assigneeId,omitempty"`
	Minted     *bool   `json:"minted,omitempty"`

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
// Firebase Storage 移行後の正:
// - Patch は read-model 用の最小情報
// - iconUrl は Firebase Storage downloadURL
// - tokenIconObjectPath / tokenContentsObjectPath は GCS 互換不要のため保持しない
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
//
// 共通化方針:
// - CreatedFrom/To, UpdatedFrom/To は common.FilterCommon の TimeRange へ寄せる
// - SearchQuery は将来の汎用検索用（必要なら NameLike/SymbolLike と併用可能）
type Filter struct {
	common.FilterCommon

	IDs         []string
	BrandIDs    []string
	CompanyIDs  []string
	AssigneeIDs []string
	Symbols     []string

	NameLike   string
	SymbolLike string
}

// ===============================
// RepositoryPort（リポジトリ境界）
// ===============================
type RepositoryPort interface {
	// 単体取得
	GetByID(ctx context.Context, id string) (*TokenBlueprint, error)

	// Patch 取得（read-model 用）
	GetPatchByID(ctx context.Context, id string) (Patch, error)

	// ID → Name の高速解決
	GetNameByID(ctx context.Context, id string) (string, error)

	// 一覧取得（オフセットページング）
	List(ctx context.Context, filter Filter, page common.Page) (common.PageResult[TokenBlueprint], error)

	// companyId で限定した一覧
	ListByCompanyID(ctx context.Context, companyID string, page common.Page) (common.PageResult[TokenBlueprint], error)

	// brandId で限定した一覧
	ListByBrandID(ctx context.Context, brandID string, page common.Page) (common.PageResult[TokenBlueprint], error)

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

// brandId ごとに一覧取得
func ListByBrandID(
	ctx context.Context,
	repo RepositoryPort,
	brandID string,
	page common.Page,
) (common.PageResult[TokenBlueprint], error) {
	return repo.ListByBrandID(ctx, brandID, page)
}

// ==========================================================
// minted = true（minted） のみ一覧取得
// ==========================================================
func ListMintedCompleted(
	ctx context.Context,
	repo RepositoryPort,
	page common.Page,
) (common.PageResult[TokenBlueprint], error) {
	result, err := repo.List(ctx, Filter{}, page)
	if err != nil {
		return common.PageResult[TokenBlueprint]{}, err
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
