// backend/internal/domain/model/alcohol.go
//
// NOTE:
//   - common.go側にModelVariation / ModelDataの共通定義があるため、
//     このファイルでは再定義しない。
//   - alcohol専用のvariationはAlcoholModelVariationとして定義する。
//   - alcoholでは容量ごとにmodel variationを作成する。
//   - vintage / region / material / alcoholContentなどは
//     ProductBlueprint.CategoryFields側を正とし、
//     Modelでは容量だけを扱う。
package model

import (
	"errors"
	"time"
)

// Volumeはalcoholの容量バリエーションを表す値オブジェクトです。
type Volume struct {
	Value int
	Unit  string
}

// AlcoholModelVariationはalcohol用のModel variationです。
type AlcoholModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Volume             Volume

	CreatedAt time.Time
	CreatedBy *string
	UpdatedAt time.Time
	UpdatedBy *string
}

// NewAlcoholModelVariationは
// alcohol Model variationの新規作成入力です。
type NewAlcoholModelVariation struct {
	ProductBlueprintID string
	ModelNumber        string
	Volume             Volume
}

// AlcoholItemSpecは商品個体・表示用途向けのread modelです。
type AlcoholItemSpec struct {
	ModelNumber string
	Volume      Volume
}

type VolumeVariation struct {
	ID     string
	Volume Volume
}

type AlcoholModelNumber struct {
	Volume      Volume
	ModelNumber string
}

var (
	ErrInvalidVolume = errors.New(
		"model: invalid volume",
	)

	ErrInvalidVolumeUnit = errors.New(
		"model: invalid volume unit",
	)
)

func (
	variation AlcoholModelVariation,
) Validate() error {
	if variation.ID == "" {
		return ErrInvalidID
	}

	if variation.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}

	if variation.ModelNumber == "" {
		return ErrInvalidModelNumber
	}

	if err :=
		variation.Volume.Validate(); err != nil {
		return err
	}

	return nil
}

func (
	volume Volume,
) Validate() error {
	if volume.Value <= 0 {
		return ErrInvalidVolume
	}

	switch volume.Unit {
	case "ml", "L":
		return nil

	default:
		return ErrInvalidVolumeUnit
	}
}

func (
	variation AlcoholModelVariation,
) GetID() string {
	return variation.ID
}

func (
	variation AlcoholModelVariation,
) GetProductBlueprintID() string {
	return variation.ProductBlueprintID
}

func (
	variation AlcoholModelVariation,
) GetKind() ModelVariationKind {
	return ModelVariationKindAlcohol
}

func (
	variation AlcoholModelVariation,
) GetModelNumber() string {
	return variation.ModelNumber
}

func (
	variation AlcoholModelVariation,
) ToItemSpec() AlcoholItemSpec {
	return AlcoholItemSpec{
		ModelNumber: variation.ModelNumber,

		Volume: variation.Volume,
	}
}
