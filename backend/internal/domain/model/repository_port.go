// backend/internal/domain/model/repository_port.go
package model

import (
	"context"
	"errors"
)

// ModelVariationKind は category-specific model variation の種別を表す。
//
// NOTE:
//   - repository port 自体は productBlueprintID 配下の model variation 永続化のみを扱う。
//   - category ごとの入力仕様・variation を作る/作らない判定は、
//     productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
//   - Product-level metadata は productBlueprint.CategoryFields に集約する。
//   - alcohol の vintage / region / material / alcoholContent などは ProductBlueprint.CategoryFields 側を正とし、
//     model variation では容量のみを扱う。
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
//   - 今後 category が増える場合も、repository port の method は増やさず、
//     NewModelVariation の組み立て・validation・mapping を application/usecase 側で吸収する。
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

// RepositoryPort abstracts model variation data access.
//
// NOTE:
//   - model variation は productBlueprint に従属するため、
//     productBlueprintID を超えた横断 List は持たない。
//   - Product-level metadata は productBlueprint.CategoryFields に集約する。
//   - この port は category-specific model variation の永続化境界として扱う。
//   - apparel では size / color / measurements を使う。
//   - alcohol では volume のみを使う。
//   - どの category で model variation を作成するかは、
//     productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
//   - size / color / volume / modelNumber などの表示用 aggregation は repository port ではなく、
//     application/query/read model 側で ListByProductBlueprintID の結果から組み立てる。
type RepositoryPort interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]ModelVariation, error)
	GetByID(ctx context.Context, variationID string) (ModelVariation, error)

	// Create creates a category-specific model variation.
	//
	// NOTE:
	//   - 新規作成では productID は使わず、NewModelVariation.ProductBlueprintID() で紐付ける。
	//   - apparel.outerwear / apparel.shoes では Measurements は nil / 空でもよい。
	//   - alcohol では Volume のみを variation field として扱う。
	//   - measurements 必須カテゴリかどうかは usecase 側で category schema を参照して判定する。
	Create(ctx context.Context, variation NewModelVariation) (ModelVariation, error)

	Update(ctx context.Context, variationID string, updates ModelVariationUpdate) (ModelVariation, error)
	Delete(ctx context.Context, variationID string) error
}

// Common repository errors
var (
	ErrNotFound = errors.New("model: not found")
	ErrConflict = errors.New("model: conflict")
	ErrInvalid  = errors.New("model: invalid")
)

// Compat alias if some code refers to Repository
type Repository = RepositoryPort
