// backend/internal/domain/model/repository_port.go
package model

import (
	"context"
	"errors"
)

// ModelVariationKindは、category-specificなModel variationの種別を表す。
type ModelVariationKind string

const (
	ModelVariationKindApparel ModelVariationKind = "apparel"
	ModelVariationKindAlcohol ModelVariationKind = "alcohol"
)

// NewModelVariationは、category-specificなModel variationの新規作成入力。
//
// Product-level metadataはProductBlueprint側を正とする。
// Alcohol Model variationは容量だけを保持する。
type NewModelVariation struct {
	Kind ModelVariationKind

	Apparel *NewApparelModelVariation
	Alcohol *NewAlcoholModelVariation
}

func NewModelVariationFromApparel(
	variation NewApparelModelVariation,
) NewModelVariation {
	return NewModelVariation{
		Kind:    ModelVariationKindApparel,
		Apparel: &variation,
	}
}

func NewModelVariationFromAlcohol(
	variation NewAlcoholModelVariation,
) NewModelVariation {
	return NewModelVariation{
		Kind:    ModelVariationKindAlcohol,
		Alcohol: &variation,
	}
}

// ProductBlueprintIDは、category-specificな入力から
// ProductBlueprint IDを取得する。
func (variation NewModelVariation) ProductBlueprintID() string {
	switch variation.Kind {
	case ModelVariationKindApparel:
		if variation.Apparel == nil {
			return ""
		}

		return variation.Apparel.ProductBlueprintID

	case ModelVariationKindAlcohol:
		if variation.Alcohol == nil {
			return ""
		}

		return variation.Alcohol.ProductBlueprintID

	default:
		return ""
	}
}

// Validateは、新規作成入力のkindとcategory-specificな内容を検証する。
func (variation NewModelVariation) Validate() error {
	switch variation.Kind {
	case ModelVariationKindApparel:
		if variation.Apparel == nil {
			return ErrInvalid
		}

		// 異なるkindの入力が同時に設定されることを許可しない。
		if variation.Alcohol != nil {
			return ErrInvalid
		}

		if variation.Apparel.ProductBlueprintID == "" {
			return ErrInvalidBlueprintID
		}

		if variation.Apparel.ModelNumber == "" {
			return ErrInvalidModelNumber
		}

		if variation.Apparel.Size == "" {
			return ErrInvalidSize
		}

		if err := variation.Apparel.Color.Validate(); err != nil {
			return err
		}

		if err := variation.Apparel.Measurements.Validate(); err != nil {
			return err
		}

		return nil

	case ModelVariationKindAlcohol:
		if variation.Alcohol == nil {
			return ErrInvalid
		}

		// 異なるkindの入力が同時に設定されることを許可しない。
		if variation.Apparel != nil {
			return ErrInvalid
		}

		if variation.Alcohol.ProductBlueprintID == "" {
			return ErrInvalidBlueprintID
		}

		if variation.Alcohol.ModelNumber == "" {
			return ErrInvalidModelNumber
		}

		return variation.Alcohol.Volume.Validate()

	default:
		return ErrInvalidKind
	}
}

// ModelVariationUpdateは、Model variationの部分更新入力。
//
// MeasurementsまたはVolumeがnilの場合は、その項目の更新を行わない。
// category schemaに基づくMeasurementsの必須判定はUsecase側で行う。
type ModelVariationUpdate struct {
	Size         *string
	Color        *Color
	ModelNumber  *string
	Measurements Measurements
	Volume       *Volume
}

// Validateは、既存Model variationのkindに対して
// 更新内容が有効であることを検証する。
func (update ModelVariationUpdate) Validate(
	kind ModelVariationKind,
) error {
	if update.ModelNumber != nil &&
		*update.ModelNumber == "" {
		return ErrInvalidModelNumber
	}

	switch kind {
	case ModelVariationKindApparel:
		// ApparelにAlcohol専用項目を設定することを許可しない。
		if update.Volume != nil {
			return ErrInvalid
		}

		if update.Size != nil &&
			*update.Size == "" {
			return ErrInvalidSize
		}

		if update.Color != nil {
			if err := update.Color.Validate(); err != nil {
				return err
			}
		}

		if err := update.Measurements.Validate(); err != nil {
			return err
		}

		return nil

	case ModelVariationKindAlcohol:
		// AlcoholにApparel専用項目を設定することを許可しない。
		if update.Size != nil ||
			update.Color != nil ||
			update.Measurements != nil {
			return ErrInvalid
		}

		if update.Volume != nil {
			return update.Volume.Validate()
		}

		return nil

	default:
		return ErrInvalidKind
	}
}

// RepositoryPortは、Model variationの永続化境界を表す。
//
// Model variationはProductBlueprintに従属するため、
// ProductBlueprintを横断する汎用Listは公開しない。
type RepositoryPort interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
	) ([]ModelVariation, error)

	GetByID(
		ctx context.Context,
		variationID string,
	) (ModelVariation, error)

	Create(
		ctx context.Context,
		variation NewModelVariation,
	) (ModelVariation, error)

	Update(
		ctx context.Context,
		variationID string,
		updates ModelVariationUpdate,
	) (ModelVariation, error)

	Delete(
		ctx context.Context,
		variationID string,
	) error

	// ReplaceByProductBlueprintIDは、指定したProductBlueprintに属する
	// 既存variationの削除と新規variationの作成を、単一の原子的処理として行う。
	//
	// 永続化基盤のtransaction上限を超える場合は、
	// 書込み開始前にErrAtomicReplaceLimitExceededを返す。
	ReplaceByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		variations []NewModelVariation,
	) ([]ModelVariation, error)
}

// Repository共通エラー。
var (
	ErrNotFound = errors.New(
		"model: not found",
	)

	ErrConflict = errors.New(
		"model: conflict",
	)

	ErrInvalid = errors.New(
		"model: invalid",
	)

	ErrInvalidKind = errors.New(
		"model: invalid kind",
	)

	ErrAtomicReplaceLimitExceeded = errors.New(
		"model: atomic replace write limit exceeded",
	)
)
