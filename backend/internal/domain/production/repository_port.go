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
	PrintedAt          *time.Time        `json:"printedAt,omitempty"`
	CreatedBy          *string           `json:"createdBy,omitempty"`
	CreatedAt          *time.Time        `json:"createdAt,omitempty"`
}

// ========================================
// Query contracts (filters/paging)
// ========================================
//
// ★重要方針:
// Production は companyId を直接持たないため、
// 一覧取得（list）は必ず「companyId → productBlueprintIds → productions」のルートで行う。
// したがって、この port からは「フィルタ無し全件」「任意条件での List（company 無）」の口を用意しない。
// （もし必要なら application/usecase 側で productBlueprintIds を解決した上で、
//  その ID 群に限定したクエリメソッドを追加する）

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
	// Read
	// ----------------------------------------

	// Production を productionId で取得
	GetByID(ctx context.Context, id string) (*Production, error)

	// ★禁止（削除）:
	// - companyId が空でも呼べてしまう「全件/任意条件の list」はマルチテナント境界を破壊するため廃止
	//   例: List(ctx, filter, page), ListAll(ctx), GetByModelID(ctx, modelID) など

	// 複数の productBlueprintId に紐づく Production 一覧
	// ★一覧取得は必ずこのメソッド経由（productBlueprintIds を上位層で companyId から解決する）
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]Production, error)

	// ★ 追加: productionId → productBlueprintId を返す関数
	//
	// MintRequest / Token 発行時などで、InspectionBatch.productionId から
	// 対応する productBlueprintId を join する用途で利用。
	GetProductBlueprintIDByProductionID(ctx context.Context, productionID string) (string, error)

	// ----------------------------------------
	// Write
	// ----------------------------------------

	// Create （CreateProductionInput ベース）
	Create(ctx context.Context, in CreateProductionInput) (*Production, error)

	// Save: Production エンティティを保存（新規 or 更新）
	//       実装側で upsert として扱って良い。
	Save(ctx context.Context, p Production) (*Production, error)

	// Delete: productionId で削除
	Delete(ctx context.Context, id string) error

	// Exists: productionId の存在確認
	Exists(ctx context.Context, id string) (bool, error)

	// Tx
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
