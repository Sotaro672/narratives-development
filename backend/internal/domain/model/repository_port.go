// backend/internal/domain/model/repository_port.go
package model

import (
	"context"
	"errors"
	"time"
)

// Domain helper types (inputs/patches)

// Measurements は「各計測位置(string) → 計測値(int)」のマップ。
// apparel.go の ApparelModelVariation.Measurements に対応。

type ModelVariationKind string

const (
	ModelVariationKindApparel ModelVariationKind = "apparel"
	ModelVariationKindAlcohol ModelVariationKind = "alcohol"
)

// NewModelVariation は category-specific model variation の新規作成入力。
//
// NOTE:
//   - apparel では NewApparelModelVariation を使う。
//   - alcohol では NewAlcoholModelVariation を使う。
//   - どの category で model variation を作成するかは、
//     productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
//   - Product-level metadata は productBlueprint.CategoryFields に集約する。
//   - alcohol の vintage / region / material / alcoholContent などは ProductBlueprint.CategoryFields 側を正とし、
//     model variation では容量のみを扱う。
type NewModelVariation struct {
	Kind ModelVariationKind

	Apparel *NewApparelModelVariation
	Alcohol *NewAlcoholModelVariation
}

func NewModelVariationFromApparel(v NewApparelModelVariation) NewModelVariation {
	return NewModelVariation{
		Kind:    ModelVariationKindApparel,
		Apparel: &v,
	}
}

func NewModelVariationFromAlcohol(v NewAlcoholModelVariation) NewModelVariation {
	return NewModelVariation{
		Kind:    ModelVariationKindAlcohol,
		Alcohol: &v,
	}
}

func (v NewModelVariation) ProductBlueprintID() string {
	switch v.Kind {
	case ModelVariationKindApparel:
		if v.Apparel == nil {
			return ""
		}

		return v.Apparel.ProductBlueprintID

	case ModelVariationKindAlcohol:
		if v.Alcohol == nil {
			return ""
		}

		return v.Alcohol.ProductBlueprintID

	default:
		return ""
	}
}

func (v NewModelVariation) Validate() error {
	switch v.Kind {
	case ModelVariationKindApparel:
		if v.Apparel == nil {
			return ErrInvalid
		}
		if v.Apparel.ProductBlueprintID == "" {
			return ErrInvalidBlueprintID
		}
		if v.Apparel.ModelNumber == "" {
			return ErrInvalidModelNumber
		}
		if v.Apparel.Size == "" {
			return ErrInvalidSize
		}
		if v.Apparel.Color.Name == "" {
			return ErrInvalidColor
		}
		if v.Apparel.Color.RGB < 0 {
			return ErrInvalidColor
		}
		for k, value := range v.Apparel.Measurements {
			if k == "" || value < 0 {
				return ErrInvalidMeasurements
			}
		}

		return nil

	case ModelVariationKindAlcohol:
		if v.Alcohol == nil {
			return ErrInvalid
		}
		if v.Alcohol.ProductBlueprintID == "" {
			return ErrInvalidBlueprintID
		}
		if v.Alcohol.ModelNumber == "" {
			return ErrInvalidModelNumber
		}

		return v.Alcohol.Volume.Validate()

	default:
		return ErrInvalid
	}
}

// ModelVariationUpdate corresponds to TS: Partial<Omit<ModelVariation, 'id'>>
//
// NOTE:
//   - apparel では size / color / measurements 更新に対応する。
//   - alcohol では volume 更新に対応する。
//   - Measurements / Volume は nil なら更新スキップ。
//   - apparel.outerwear / apparel.shoes では Measurements は空でもよい。
//   - measurements 必須判定は productBlueprintCategory schema を
//     application/usecase 側で参照して行う。
type ModelVariationUpdate struct {
	Size         *string      `json:"size,omitempty"`
	Color        *Color       `json:"color,omitempty"` // 部分更新したい場合は構造に注意
	ModelNumber  *string      `json:"modelNumber,omitempty"`
	Measurements Measurements `json:"measurements,omitempty"` // nil なら更新スキップ

	// alcohol 用。
	// nil なら更新スキップ。
	Volume *Volume `json:"volume,omitempty"`
}

// Listing contracts (filters/sort/page)

type VariationFilter struct {
	ProductBlueprintID string

	Sizes        []string
	Colors       []string // Color.Name を前提としたフィルタとして扱う想定
	Volumes      []Volume // alcohol 用の容量フィルタ
	ModelNumbers []string

	SearchQuery string // free text over modelNumber/size/color/volume (implementation-defined)

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

// RepositoryPort abstracts model variation data access.
//
// NOTE:
//   - Product-level metadata は productBlueprint.CategoryFields に集約する。
//   - この port は category-specific model variation の永続化境界として扱う。
//   - apparel では size / color / measurements を使う。
//   - alcohol では volume のみを使う。
//   - どの category で model variation を作成するかは、
//     productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
type RepositoryPort interface {
	// Variations (CRUD)
	ListVariations(ctx context.Context, filter VariationFilter, page Page) (VariationPageResult, error)

	GetModelVariations(ctx context.Context, productBlueprintID string) ([]ModelVariation, error)
	GetModelVariationByID(ctx context.Context, variationID string) (*ModelVariation, error)

	// CreateModelVariation creates a category-specific model variation.
	//
	// NOTE:
	//   - 新規作成では productID は使わず、NewModelVariation.ProductBlueprintID() で紐付ける。
	//   - apparel.outerwear / apparel.shoes では Measurements は nil / 空でもよい。
	//   - alcohol では Volume のみを variation field として扱う。
	//   - measurements 必須カテゴリかどうかは usecase 側で category schema を参照して判定する。
	CreateModelVariation(ctx context.Context, variation NewModelVariation) (*ModelVariation, error)

	UpdateModelVariation(ctx context.Context, variationID string, updates ModelVariationUpdate) (*ModelVariation, error)
	DeleteModelVariation(ctx context.Context, variationID string) (*ModelVariation, error)

	// ReplaceModelVariations replaces category-specific model variations.
	//
	// NOTE:
	//   - 全要素が同じ ProductBlueprintID を持つ前提。
	//   - ProductBlueprintID は NewModelVariation.ProductBlueprintID() から解決する。
	//   - apparel では size / color / measurements を使う。
	//   - alcohol では volume のみを使う。
	ReplaceModelVariations(ctx context.Context, variations []NewModelVariation) ([]ModelVariation, error)

	// Convenience aggregations (resolver-style)
	GetSizeVariations(ctx context.Context, productBlueprintID string) ([]SizeVariation, error)
	GetModelNumbers(ctx context.Context, productBlueprintID string) ([]ModelNumber, error)
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
