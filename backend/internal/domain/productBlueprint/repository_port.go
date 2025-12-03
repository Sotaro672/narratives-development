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

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"` // repo may set if nil
}

type Patch struct {
	ProductName      *string       `json:"productName,omitempty"`
	BrandID          *string       `json:"brandId,omitempty"`
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
	SearchTerm string

	BrandIDs    []string
	AssigneeIDs []string
	ItemTypes   []ItemType
	TagTypes    []ProductIDTagType

	// 削除状態フィルタ:
	// - false: 通常の（未削除）レコードを対象（実装側で DeletedAt == nil を想定）
	// - true : 論理削除済み（DeletedAt != nil）のみを対象
	OnlyDeleted bool

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt   SortColumn = "createdAt"
	SortByUpdatedAt   SortColumn = "updatedAt"
	SortByProductName SortColumn = "productName"
	SortByBrandID     SortColumn = "brandId"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

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
// LogCard で使う version / updatedAt / updatedBy に加えて、
// 当時の ProductBlueprint 本体も保持しておく。
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
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)

	// companyId 単位で productBlueprint の ID 一覧を取得
	// （MintRequest 用のチェーン: companyId → productBlueprintId → production → mintRequest）
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// 存在確認（adapter の Exists を port に昇格）
	Exists(ctx context.Context, id string) (bool, error)

	// Write (live)
	Create(ctx context.Context, in CreateInput) (ProductBlueprint, error)
	Update(ctx context.Context, id string, patch Patch) (ProductBlueprint, error)
	Delete(ctx context.Context, id string) error

	// History (snapshot, versioned)
	// ★ version は ProductBlueprint.Version に従う前提。
	SaveHistorySnapshot(ctx context.Context, blueprintID string, h HistoryRecord) error
	ListHistory(ctx context.Context, blueprintID string) ([]HistoryRecord, error)
	GetHistoryByVersion(ctx context.Context, blueprintID string, version int64) (HistoryRecord, error)

	// Dev/Test helper
	Reset(ctx context.Context) error
}
