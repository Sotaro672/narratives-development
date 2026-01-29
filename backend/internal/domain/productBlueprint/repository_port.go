package productBlueprint

import (
	"context"
	"time"
)

// ========================================
// Create/Update inputs (contract only)
// ========================================

type CreateInput struct {
	ProductName      string       `json:"productName"`
	BrandID          string       `json:"brandId"`
	ItemType         ItemType     `json:"itemType"`
	Fit              string       `json:"fit"`
	Material         string       `json:"material"`
	Weight           float64      `json:"weight"`
	QualityAssurance []string     `json:"qualityAssurance"`
	ProductIdTag     ProductIDTag `json:"productIdTag"`
	AssigneeID       string       `json:"assigneeId"`
	CompanyID        string       `json:"companyId"`
	CreatedBy        *string      `json:"createdBy,omitempty"`
	CreatedAt        *time.Time   `json:"createdAt,omitempty"` // repo may set if nil
}

type Patch struct {
	ProductName *string `json:"productName,omitempty"`

	// ✅ 既存：更新に使うID
	BrandID *string `json:"brandId,omitempty"`

	// ✅ 追加：表示用（InventoryDetailなど read-model で埋める）
	// NOTE: Update入力として受け取っても、永続化に使わない想定（表示専用）。
	BrandName *string `json:"brandName,omitempty"`

	// ✅ NEW: company (read-only display fields)
	// NOTE: Update入力として受け取っても、永続化に使わない想定（表示専用）。
	CompanyID   *string `json:"companyId,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`

	ItemType         *ItemType     `json:"itemType,omitempty"`
	Fit              *string       `json:"fit,omitempty"`
	Material         *string       `json:"material,omitempty"`
	Weight           *float64      `json:"weight,omitempty"`
	QualityAssurance *[]string     `json:"qualityAssurance,omitempty"`
	ProductIdTag     *ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID       *string       `json:"assigneeId,omitempty"`
}

// ========================================
// Query contracts (filters/sort/paging)
// ========================================

type Filter struct {
	CompanyID   string // ★ 必須: マルチテナント境界
	SearchTerm  string
	BrandIDs    []string
	AssigneeIDs []string
	ItemTypes   []ItemType
	TagTypes    []ProductIDTagType
	OnlyDeleted bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
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
// History contracts
// ========================================

// HistoryRecord は ProductBlueprint のバージョン履歴 1 件分を表す。
type HistoryRecord struct {
	Blueprint ProductBlueprint
	Version   int64
	UpdatedAt time.Time
	UpdatedBy *string
}

// ProductBlueprintHistoryRepo は Firestore の
// product_blueprints_history/{blueprintId}/versions/{version}
// にアクセスするための専用ポート。
type ProductBlueprintHistoryRepo interface {
	// ライブの ProductBlueprint スナップショットを指定 version で保存
	SaveSnapshot(ctx context.Context, pb ProductBlueprint) error

	// 特定の productBlueprintID の履歴一覧（LogCard 用）
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]ProductBlueprint, error)
}

// ========================================
// Repository Port (interface contracts only)
// ========================================

type Repository interface {
	// Read (live)
	GetByID(ctx context.Context, id string) (ProductBlueprint, error)

	// ★ 追加: productBlueprintId から brandId だけを取得するヘルパ
	GetBrandIDByID(ctx context.Context, id string) (string, error)

	// ★ 追加: productBlueprintId から productName だけを取得するヘルパ
	GetProductNameByID(ctx context.Context, id string) (string, error)

	// ★ 追加: modelId(=variationId想定) から productBlueprintId を取得するヘルパ
	GetIDByModelID(ctx context.Context, modelID string) (string, error)

	// ★ 追加: productBlueprintId から Patch 相当の情報を取得するヘルパ
	GetPatchByID(ctx context.Context, id string) (Patch, error)

	// companyId 単位で productBlueprint の ID 一覧を取得
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// 存在確認（adapter の Exists を port に昇格）
	Exists(ctx context.Context, id string) (bool, error)

	// Write (live)
	Create(ctx context.Context, in CreateInput) (ProductBlueprint, error)
	Update(ctx context.Context, id string, patch Patch) (ProductBlueprint, error)
	Delete(ctx context.Context, id string) error

	// ★ 追加: ProductBlueprint 起票後に modelRefs（modelId + displayOrder）を追記する
	//
	// 要件:
	// - updatedAt / updatedBy を更新しない（touch しない部分更新）
	// - modelRefs だけを書き換える（Firestore の Update / Set(merge) など）
	//
	// Contract:
	// - refs は表示順（DisplayOrder）が埋まっていること（1..N）
	// - 実装側で既存とマージして重複排除し、必要なら displayOrder を採番し直してよい
	AppendModelRefsWithoutTouch(ctx context.Context, id string, refs []ModelRef) (ProductBlueprint, error)

	// ★ printed: false → true への状態遷移
	MarkPrinted(ctx context.Context, id string) (ProductBlueprint, error)

	// History (snapshot, versioned)
	SaveHistorySnapshot(ctx context.Context, blueprintID string, h HistoryRecord) error
	ListHistory(ctx context.Context, blueprintID string) ([]HistoryRecord, error)
	GetHistoryByVersion(ctx context.Context, blueprintID string, version int64) (HistoryRecord, error)
}
