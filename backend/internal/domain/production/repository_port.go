package production

import (
	"context"
	"time"
)

// ========================================
// Create/Update inputs (contract only)
// ========================================

// CreateProductionInput - 作成用入力（ID/UpdatedAtは実装側で採番/付与可）
type CreateProductionInput struct {
	ProductBlueprintID string            `json:"productBlueprintId"`
	AssigneeID         string            `json:"assigneeId"`
	Models             []ModelQuantity   `json:"models"`           // [{modelId, quantity}]
	Status             *ProductionStatus `json:"status,omitempty"` // nilなら既定（manufacturing)

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"` // nilなら実装側で設定
}

// UpdateProductionInput - 部分更新入力（nilは未更新）
// 状態遷移の整合性はアプリ/ドメインサービスで担保してください。
type UpdateProductionInput struct {
	ProductBlueprintID *string           `json:"productBlueprintId,omitempty"`
	AssigneeID         *string           `json:"assigneeId,omitempty"`
	Models             *[]ModelQuantity  `json:"models,omitempty"`
	Status             *ProductionStatus `json:"status,omitempty"`

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`

	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"` // ソフトデリート用
}

// ドメイン特化オペレーション用の入力（必要に応じて利用）
type MarkPrintedInput struct {
	At time.Time
}

type MarkInspectedInput struct {
	At time.Time
}

// ========================================
// Query contracts (filters/sort/paging)
// ========================================

type Filter struct {
	// 識別子
	ID                 string
	ProductBlueprintID string
	AssigneeID         string

	// モデルIDに紐づく生産計画（Modelsに含まれるもの）
	ModelID string

	// ステータス
	Statuses []ProductionStatus

	// 時刻レンジ
	PrintedFrom   *time.Time
	PrintedTo     *time.Time
	InspectedFrom *time.Time
	InspectedTo   *time.Time
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	UpdatedFrom   *time.Time
	UpdatedTo     *time.Time
	DeletedFrom   *time.Time
	DeletedTo     *time.Time

	// nil: 全件, true: 削除済のみ, false: 未削除のみ
	Deleted *bool
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByID          SortColumn = "id"
	SortByCreatedAt   SortColumn = "createdAt"
	SortByUpdatedAt   SortColumn = "updatedAt"
	SortByPrintedAt   SortColumn = "printedAt"
	SortByInspectedAt SortColumn = "inspectedAt"
	SortByStatus      SortColumn = "status"
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
	Items      []Production
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ========================================
// Repository Port (interface contracts only)
// ========================================

type RepositoryPort interface {
	// 取得
	GetByID(ctx context.Context, id string) (*Production, error)
	// 互換API: 指定modelIdを含む生産計画を返す（Filter.ModelIDのショートカット）
	GetByModelID(ctx context.Context, modelID string) ([]Production, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, in CreateProductionInput) (*Production, error)
	Update(ctx context.Context, id string, patch UpdateProductionInput) (*Production, error)
	Delete(ctx context.Context, id string) error

	// ドメイン特化（必要なら。実装側で Update と等価にしても良い）
	MarkPrinted(ctx context.Context, id string, in MarkPrintedInput) (*Production, error)
	MarkInspected(ctx context.Context, id string, in MarkInspectedInput) (*Production, error)
	ResetToManufacturing(ctx context.Context, id string) (*Production, error)

	// トランザクション境界（任意）
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
