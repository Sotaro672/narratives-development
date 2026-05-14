// backend/internal/domain/model/alcohol.go
// Responsibility: alcohol category model variation definitions.
//
// NOTE:
//   - common.go 側に ModelVariation / ModelData / Model の共通定義があるため、
//     このファイルでは再定義しない。
//   - alcohol 専用の variation は AlcoholModelVariation として定義する。
//   - alcohol では容量ごとに model variation を作成する。
//   - 在庫・出品売価入力に必要な modelId を確保するため、
//     alcohol.sake / alcohol.wine / alcohol.beer などでは Volume を使う。
//   - 現時点では容量のみを variation field として扱い、
//     vintage / region / material / alcoholContent などは
//     ProductBlueprint.CategoryFields 側を正とする。
package model

import (
	"errors"
	"time"
)

// Volume は alcohol の容量バリエーションを表す値オブジェクト。
// - Value: 容量の数値。例: 720, 1000, 1800
// - Unit : 単位。例: "ml", "L"
type Volume struct {
	Value int
	Unit  string
}

// AlcoholModelVariation は alcohol 用の model variation。
//
// category ごとの扱い:
//   - alcohol.sake:
//     Volume を使う
//   - alcohol.wine:
//     Volume を使う
//   - alcohol.beer:
//     Volume を使う
//   - alcohol.shochu:
//     Volume を使う
//   - alcohol.spirits:
//     Volume を使う
//   - alcohol.whisky:
//     Volume を使う
//
// NOTE:
// ModelNumber は現行実装で必須として扱う。
// 在庫・出品売価連携が modelNumber / modelId を前提にしているため、
// alcohol でも必須を維持する。
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

// NewAlcoholModelVariation は alcohol model variation の新規作成時に使う入力モデル。
// 既存の AlcoholModelVariation から ID や監査情報だけを省いた形。
type NewAlcoholModelVariation struct {
	ProductBlueprintID string
	ModelNumber        string
	Volume             Volume
}

// AlcoholItemSpec は alcohol model variation を商品個体・表示用途向けに
// 簡易化した read model 的な値。
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
	ErrInvalidVolume     = errors.New("model: invalid volume")
	ErrInvalidVolumeUnit = errors.New("model: invalid volume unit")
)

var AllowedVolumeUnits = map[string]struct{}{
	"ml": {},
	"L":  {},
}

func (mv AlcoholModelVariation) Validate() error {
	return mv.validate()
}

func (mv AlcoholModelVariation) validate() error {
	if mv.ID == "" {
		return ErrInvalidID
	}
	if mv.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}
	if mv.ModelNumber == "" {
		return ErrInvalidModelNumber
	}
	if err := mv.Volume.Validate(); err != nil {
		return err
	}

	// CreatedAt / UpdatedAt はゼロ値許容（リポジトリや Usecase 側で設定）
	return nil
}

func (v Volume) Validate() error {
	if v.Value <= 0 {
		return ErrInvalidVolume
	}
	if v.Unit == "" {
		return ErrInvalidVolumeUnit
	}
	if !volumeUnitAllowed(v.Unit) {
		return ErrInvalidVolumeUnit
	}

	return nil
}

func (mv AlcoholModelVariation) GetID() string {
	return mv.ID
}

func (mv AlcoholModelVariation) GetProductBlueprintID() string {
	return mv.ProductBlueprintID
}

func (mv AlcoholModelVariation) GetModelNumber() string {
	return mv.ModelNumber
}

func (mv AlcoholModelVariation) ToItemSpec() AlcoholItemSpec {
	return AlcoholItemSpec{
		ModelNumber: mv.ModelNumber,
		Volume:      mv.Volume,
	}
}

func volumeUnitAllowed(unit string) bool {
	if len(AllowedVolumeUnits) == 0 {
		return true
	}
	_, ok := AllowedVolumeUnits[unit]

	return ok
}
