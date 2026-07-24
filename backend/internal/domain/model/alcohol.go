// Responsibility: alcohol category model variation definitions.
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

// AlcoholModelVariationはalcohol用のmodel variationです。
type AlcoholModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Volume             Volume
	CreatedAt          time.Time
	CreatedBy          *string
	UpdatedAt          time.Time
	UpdatedBy          *string
}

// NewAlcoholModelVariationは
// alcohol model variationの新規作成入力です。
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
	ErrInvalidVolume = errors.New("model: invalid volume")

	ErrInvalidVolumeUnit = errors.New("model: invalid volume unit")
)

func (mv AlcoholModelVariation) Validate() error {
	if mv.ID == "" {
		return ErrInvalidID
	}

	if mv.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}

	if mv.ModelNumber == "" {
		return ErrInvalidModelNumber
	}

	return mv.Volume.Validate()
}

func (v Volume) Validate() error {
	if v.Value <= 0 {
		return ErrInvalidVolume
	}

	switch v.Unit {
	case "ml", "L":
		return nil

	default:
		return ErrInvalidVolumeUnit
	}
}

func (mv AlcoholModelVariation) GetID() string {
	return mv.ID
}

func (
	mv AlcoholModelVariation,
) GetProductBlueprintID() string {
	return mv.ProductBlueprintID
}

func (
	mv AlcoholModelVariation,
) GetKind() ModelVariationKind {
	return ModelVariationKindAlcohol
}

func (
	mv AlcoholModelVariation,
) GetModelNumber() string {
	return mv.ModelNumber
}

func (
	mv AlcoholModelVariation,
) ToItemSpec() AlcoholItemSpec {
	return AlcoholItemSpec{
		ModelNumber: mv.ModelNumber,
		Volume:      mv.Volume,
	}
}
