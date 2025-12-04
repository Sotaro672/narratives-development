// backend/internal/domain/production/repository_port.go
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
	Status             *ProductionStatus `json:"status,omitempty"`

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// ========================================
// Query contracts (filters/paging)
// ========================================

type Filter struct {
	ID                 string
	ProductBlueprintID string
	AssigneeID         string
	ModelID            string

	Statuses []ProductionStatus

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
	// Production を productionId で取得
	GetByID(ctx context.Context, id string) (*Production, error)

	// 指定 modelId を含む生産計画一覧
	GetByModelID(ctx context.Context, modelID string) ([]Production, error)

	// List（ページング）
	List(ctx context.Context, filter Filter, page Page) (PageResult, error)

	// Create
	Create(ctx context.Context, in CreateProductionInput) (*Production, error)

	// 複数の productBlueprintId に紐づく Production 一覧
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]Production, error)

	// ★ 追加: productionId → productBlueprintId を返す関数
	//
	// MintRequest / Token 発行時などで、InspectionBatch.productionId から
	// 対応する productBlueprintId を join する用途で利用。
	GetProductBlueprintIDByProductionID(ctx context.Context, productionID string) (string, error)

	// Tx
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
