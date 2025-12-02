// backend/internal/domain/model/repository_port.go
package model

import (
	"context"
	"errors"
	"time"
)

// Domain helper types (inputs/patches)

// Measurements は「各計測位置(string) → 計測値(int)」のマップ。
// entity.go の ModelVariation.Measurements (map[string]int) に対応。

// ModelVariationUpdate corresponds to TS: Partial<Omit<ModelVariation, 'id'>>
type ModelVariationUpdate struct {
	Size         *string      `json:"size,omitempty"`
	Color        *Color       `json:"color,omitempty"` // 部分更新したい場合は構造に注意
	ModelNumber  *string      `json:"modelNumber,omitempty"`
	Measurements Measurements `json:"measurements,omitempty"` // nil なら更新スキップ
}

// ModelDataUpdate is free-form for product-level metadata updates
type ModelDataUpdate map[string]any

// Listing contracts (filters/sort/page)

type VariationFilter struct {
	ProductID          string
	ProductBlueprintID string

	Sizes        []string
	Colors       []string // Color.Name を前提としたフィルタとして扱う想定
	ModelNumbers []string

	SearchQuery string // free text over modelNumber/size/color (implementation-defined)

	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	Deleted *bool // nil: all, true: deleted only, false: non-deleted only
}

type Page struct {
	Number  int
	PerPage int
}

type VariationPageResult struct {
	Items      []ModelVariation
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// RepositoryPort abstracts model data access (contracts only)
type RepositoryPort interface {
	// Product-scoped model data
	GetModelData(ctx context.Context, productID string) (*ModelData, error)
	GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*ModelData, error)
	UpdateModelData(ctx context.Context, productID string, updates ModelDataUpdate) (*ModelData, error)

	// Variations (CRUD)
	ListVariations(ctx context.Context, filter VariationFilter, page Page) (VariationPageResult, error)
	GetModelVariations(ctx context.Context, productID string) ([]ModelVariation, error)
	GetModelVariationByID(ctx context.Context, variationID string) (*ModelVariation, error)

	// ★ 新規作成では productID は使わず、NewModelVariation.ProductBlueprintID で紐付ける
	CreateModelVariation(ctx context.Context, variation NewModelVariation) (*ModelVariation, error)

	UpdateModelVariation(ctx context.Context, variationID string, updates ModelVariationUpdate) (*ModelVariation, error)
	DeleteModelVariation(ctx context.Context, variationID string) (*ModelVariation, error)

	// ★ 一括置き換えも NewModelVariation 側の ProductBlueprintID から解決する
	//   （全要素が同じ ProductBlueprintID を持つ前提）
	ReplaceModelVariations(ctx context.Context, variations []NewModelVariation) ([]ModelVariation, error)

	// Convenience aggregations (resolver-style)
	GetSizeVariations(ctx context.Context, productID string) ([]SizeVariation, error)
	GetModelNumbers(ctx context.Context, productID string) ([]ModelNumber, error)
}

// Common repository errors
var (
	ErrNotFound = errors.New("model: not found")
	ErrConflict = errors.New("model: conflict")
	ErrInvalid  = errors.New("model: invalid")
)

// Compat alias if some code refers to Repository
type Repository = RepositoryPort

// ============================================================
// History repository port for Model (versioned snapshot)
// ============================================================
//
// Firestore 実装（ModelHistoryRepositoryFS）は、以下のようなパスで保存する想定：
//
//	product_blueprints_history/{blueprintId}/models/{version}/variations/{variationId}
//
// version は ProductBlueprint 側の Version と同期して管理する。
type ModelHistoryRepository interface {
	// SaveSnapshot:
	//   指定された blueprintID + blueprintVersion に対して、
	//   variations（ライブの ModelVariation 一式）のスナップショットを保存する。
	SaveSnapshot(
		ctx context.Context,
		productBlueprintID string,
		productBlueprintVersion int64,
		variations []ModelVariation,
	) error

	// ListByProductBlueprintIDAndVersion:
	//   指定された blueprintID + version に紐づく ModelVariation の履歴をすべて返す。
	//   LogCard から、特定バージョン時点の Model 行を表示する用途を想定。
	ListByProductBlueprintIDAndVersion(
		ctx context.Context,
		productBlueprintID string,
		productBlueprintVersion int64,
	) ([]ModelVariation, error)
}
