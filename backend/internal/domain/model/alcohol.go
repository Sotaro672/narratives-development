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

	commondom "narratives/internal/domain/common"
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

	// DeletionLifecycleはModel Documentの
	// 論理削除・復旧・物理削除予定時刻を保持します。
	//
	// statusが未設定の既存Documentは、
	// common.NormalizeDeletionStatusによりactiveとして扱います。
	commondom.DeletionLifecycle

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

	if err := validateModelDeletionLifecycle(
		variation.DeletionLifecycle,
	); err != nil {
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

// GetDeletionLifecycleは、ポインタを共有しないLifecycleを返します。
func (
	variation AlcoholModelVariation,
) GetDeletionLifecycle() commondom.DeletionLifecycle {
	return variation.
		DeletionLifecycle.
		Clone()
}

// CanModifyはModel自身が変更可能な状態か返します。
//
// ProductBlueprintのprinted状態はModel自身には保持しないため、
// UsecaseまたはRepository側でProductBlueprint.CanModifyと
// 組み合わせて判定します。
func (
	variation AlcoholModelVariation,
) CanModify() bool {
	return canModifyModelDeletionLifecycle(
		variation.DeletionLifecycle,
	)
}

// CanRestoreは指定時刻時点で復旧可能か返します。
func (
	variation AlcoholModelVariation,
) CanRestore(
	now time.Time,
) bool {
	return variation.
		DeletionLifecycle.
		CanRestore(now)
}

// IsPurgeEligibleは指定時刻時点で
// 物理削除対象か返します。
func (
	variation AlcoholModelVariation,
) IsPurgeEligible(
	now time.Time,
) bool {
	return variation.
		DeletionLifecycle.
		IsPurgeEligible(now)
}

// SoftDeleteはModelを論理削除状態へ遷移させます。
//
// 同じModelへ複数回実行された場合は、
// 最初のDeletedAtとPurgeAtを維持します。
func (
	variation *AlcoholModelVariation,
) SoftDelete(
	now time.Time,
	deletedBy *string,
) error {
	if variation == nil {
		return ErrInvalid
	}

	return softDeleteModelDeletionLifecycle(
		&variation.DeletionLifecycle,
		&variation.UpdatedAt,
		&variation.UpdatedBy,
		now,
		deletedBy,
	)
}

// Restoreは論理削除済みModelを復旧します。
//
// now < PurgeAtの場合だけ復旧可能です。
func (
	variation *AlcoholModelVariation,
) Restore(
	now time.Time,
	restoredBy *string,
) error {
	if variation == nil {
		return ErrInvalid
	}

	return restoreModelDeletionLifecycle(
		&variation.DeletionLifecycle,
		&variation.UpdatedAt,
		&variation.UpdatedBy,
		now,
		restoredBy,
	)
}

func (
	variation AlcoholModelVariation,
) ToItemSpec() AlcoholItemSpec {
	return AlcoholItemSpec{
		ModelNumber: variation.ModelNumber,

		Volume: variation.Volume,
	}
}
