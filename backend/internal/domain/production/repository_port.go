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
	// ----------------------------------------
	// 既存: 高機能クエリ系（フィルタ＋ページングなど）
	// ----------------------------------------

	// Production を productionId で取得
	GetByID(ctx context.Context, id string) (*Production, error)

	// 指定 modelId を含む生産計画一覧
	GetByModelID(ctx context.Context, modelID string) ([]Production, error)

	// List（ページング）
	List(ctx context.Context, filter Filter, page Page) (PageResult, error)

	// Create （CreateProductionInput ベース）
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

	// ----------------------------------------
	// ★ 方針1: CRUD を持たせるための拡張メソッド
	//   - application/production.Usecase などの
	//     シンプルな CRUD 用ポートとして利用する
	// ----------------------------------------

	// Exists: productionId の存在確認
	Exists(ctx context.Context, id string) (bool, error)

	// ListAll: フィルタ無しで全件一覧を返す（コンソール一覧などシンプル用途向け）
	ListAll(ctx context.Context) ([]Production, error)

	// Save: Production エンティティを保存（新規 or 更新）
	//       実装側で upsert として扱って良い。
	Save(ctx context.Context, p Production) (*Production, error)

	// Delete: productionId で削除
	Delete(ctx context.Context, id string) error
}
