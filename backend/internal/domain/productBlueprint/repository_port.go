// backend/internal/domain/productBlueprint/repository_port.go
package productBlueprint

import (
	"context"
	"time"
)

// ========================================
// Create/Update inputs (contract only)
// ========================================

type CreateInput struct {
	// ★ Create時に usecase で生成して渡す
	ID string `json:"id"`

	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandID   string `json:"brandId"`
	CompanyID string `json:"companyId"`

	// productBlueprintCategories の正データから usecase で生成して渡す denormalized snapshot
	ProductBlueprintCategory ProductBlueprintCategorySnapshot `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値を保持する。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	//
	// brandId / productName / productIdTagType / description など、
	// ProductBlueprint の共通 field はここには入れない。
	CategoryFields CategoryFields `json:"categoryFields,omitempty"`

	ProductIdTag ProductIDTag `json:"productIdTag"`
	AssigneeID   string       `json:"assigneeId"`

	// ★ modelRefs（modelId + displayOrder）
	// NOTE:
	// - create 時点では空でもよい（後段で AppendModelRefsWithoutTouch で追記する運用を許容）
	// - 永続化は adapter 側で modelRefs として保存する想定
	ModelRefs []ModelRef `json:"modelRefs,omitempty"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"` // ★ usecase が必ず埋める（domain.validate が必須）
}

type Patch struct {
	ProductName *string `json:"productName,omitempty"`
	Description *string `json:"description,omitempty"`

	// ✅ 既存：更新に使うID
	BrandID *string `json:"brandId,omitempty"`

	// ✅ 追加：表示用（InventoryDetailなど read-model で埋める）
	// NOTE: Update入力として受け取っても、永続化に使わない想定（表示専用）。
	BrandName *string `json:"brandName,omitempty"`

	// ✅ company (read-only display fields)
	// NOTE: Update入力として受け取っても、永続化に使わない想定（表示専用）。
	CompanyID   *string `json:"companyId,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`

	// productBlueprintCategories の正データから usecase で生成して渡す denormalized snapshot
	ProductBlueprintCategory *ProductBlueprintCategorySnapshot `json:"productBlueprintCategory,omitempty"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値を保持する。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	//
	// nil の場合は更新しない。
	// 空 map の場合は categoryFields を空に更新する想定。
	CategoryFields *CategoryFields `json:"categoryFields,omitempty"`

	ProductIdTag *ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID   *string       `json:"assigneeId,omitempty"`

	// ★ modelRefs を受ける（displayOrder 含む）
	// NOTE:
	// - これを永続化に使う（modelRefs を正にする）
	// - displayOrder は 1..N の採番済みを期待（ただし実装側で正規化/再採番してよい）
	ModelRefs *[]ModelRef `json:"modelRefs,omitempty"`
}

// ========================================
// Query contracts (filters/sort/paging)
// ========================================

type Filter struct {
	CompanyID   string // ★ 必須: マルチテナント境界
	SearchTerm  string
	BrandIDs    []string
	AssigneeIDs []string

	// カテゴリ検索用。
	// productBlueprint 側では denormalized field を検索対象にする。
	ProductBlueprintCategoryIDs   []string
	ProductBlueprintCategoryCodes []string
	ProductBlueprintCategoryKinds []string

	TagTypes []ProductIDTagType
}

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []ProductBlueprint
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ========================================
// Repository Port (interface contracts only)
// ========================================

type Repository interface {
	// Read (live)
	GetByID(ctx context.Context, id string) (ProductBlueprint, error)

	// companyId 単位で ProductBlueprint 一覧を取得する唯一の正規 port。
	// ID 一覧が必要な場合も、この戻り値から呼び出し側で ID を抽出する。
	ListByCompanyID(ctx context.Context, companyID string) ([]ProductBlueprint, error)

	// brandId から productBlueprint の ID 一覧を取得するヘルパ。
	ListIDsByBrandID(ctx context.Context, brandID string) ([]string, error)

	// modelId(=variationId想定) から、その model を含む ProductBlueprint の ID と modelRefs を取得する。
	//
	// 戻り値:
	// - productBlueprintID: model が紐づく ProductBlueprint の ID
	// - modelRefs: 対象 ProductBlueprint の modelRefs（displayOrder 含む）
	//
	// NOTE:
	// - 旧 GetModelRefsByModelID は廃止。
	// - productBlueprintId だけが必要な caller は第1戻り値を使う。
	// - displayOrder が必要な caller は第2戻り値の modelRefs から対象 modelId を探す。
	GetIDByModelID(ctx context.Context, modelID string) (string, []ModelRef, error)

	// Write (live)
	Create(ctx context.Context, in CreateInput) (ProductBlueprint, error)
	Update(ctx context.Context, id string, patch Patch) (ProductBlueprint, error)

	// Delete physically removes a ProductBlueprint by ID.
	Delete(ctx context.Context, id string) error

	// ProductBlueprint 起票後に modelRefs（modelId + displayOrder）を追記する。
	AppendModelRefsWithoutTouch(ctx context.Context, id string, refs []ModelRef) (ProductBlueprint, error)

	// printed: false → true への状態遷移。
	MarkPrinted(ctx context.Context, id string) (ProductBlueprint, error)
}
