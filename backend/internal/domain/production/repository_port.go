package production

import (
	"context"
	"time"
)

// ========================================
// Create inputs (contract only)
// ========================================

// CreateProductionInput - 作成用入力（IDは実装側で採番可）
type CreateProductionInput struct {
	ProductBlueprintID string            `json:"productBlueprintId"`
	AssigneeID         string            `json:"assigneeId"`
	Models             []ModelQuantity   `json:"models"`
	Status             *ProductionStatus `json:"status,omitempty"` // nilなら既定（manufacturing）

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"` // nilなら実装側で設定
}

// ========================================
// Query contracts (filters/paging)
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
}

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

	// 指定modelIdを含む生産計画を返す（Filter.ModelID のショートカット）
	GetByModelID(ctx context.Context, modelID string) ([]Production, error)

	// List（sort削除）
	List(ctx context.Context, filter Filter, page Page) (PageResult, error)

	// Create のみ残す
	Create(ctx context.Context, in CreateProductionInput) (*Production, error)

	// Tx（任意）
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
