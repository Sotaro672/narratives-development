// backend/internal/domain/model/apparel.go
//
// NOTE:
//   - common.go側にModelVariation / ModelDataの共通定義があるため、
//     このファイルでは再定義しない。
//   - apparel専用のvariationはApparelModelVariationとして定義する。
//   - apparel.tops / apparel.bottoms / apparel.dressは
//     Color / Size / Measurementsを使う。
//   - apparel.outerwear / apparel.shoesはColor / Sizeを使い、
//     Measurementsは空でもよい。
//   - apparel.accessory / apparel.bagは原則model variationを作成しない。
//   - Measurementsの必須判定は
//     productBlueprintCategory/input_schema.goのschemaを
//     application/usecase側で参照して行う。
package model

import (
	"errors"
	"time"

	commondom "narratives/internal/domain/common"
)

var (
	ErrProductIDRequired = errors.New(
		"productId is required",
	)

	ErrVariationIDRequired = errors.New(
		"variationId is required",
	)

	ErrTargetVariationNotFound = errors.New(
		"target variation not found",
	)

	ErrNoVariationsFoundForSize = errors.New(
		"no variations found for size",
	)

	ErrNoVariationsFoundForColor = errors.New(
		"no variations found for color",
	)

	ErrProductBlueprintIDNotFound = errors.New(
		"product blueprint id not found",
	)

	ErrProductBlueprintNotFound = errors.New(
		"product blueprint not found",
	)

	ErrVariationNotFound = errors.New(
		"variation not found",
	)
)

// Colorはカラーバリエーションを表す値オブジェクトです。
// RGBは0x000000から0xFFFFFFまでの24bit整数を使用します。
type Color struct {
	Name string
	RGB  int
}

func (
	color Color,
) Validate() error {
	if color.Name == "" {
		return ErrInvalidColor
	}

	if color.RGB < 0 ||
		color.RGB > 0xFFFFFF {
		return ErrInvalidColor
	}

	return nil
}

// Measurementsはapparelの採寸値を表します。
// nilと空mapのどちらも有効です。
type Measurements map[string]int

func (
	measurements Measurements,
) Validate() error {
	for key, value := range measurements {
		if key == "" ||
			value < 0 {
			return ErrInvalidMeasurements
		}
	}

	return nil
}

func (
	measurements Measurements,
) Clone() Measurements {
	if measurements == nil {
		return nil
	}

	output := make(
		Measurements,
		len(measurements),
	)

	for key, value := range measurements {
		output[key] = value
	}

	return output
}

// ApparelModelVariationはapparel用のModel variationです。
type ApparelModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Measurements       Measurements
	Color              Color

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

// NewApparelModelVariationは
// apparel Model variationの新規作成入力です。
type NewApparelModelVariation struct {
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Color              Color
	Measurements       Measurements
}

// ApparelItemSpecは商品個体・表示用途向けのread modelです。
type ApparelItemSpec struct {
	ModelNumber  string
	Size         string
	Color        string
	Measurements Measurements
}

type SizeVariation struct {
	ID           string
	Size         string
	Measurements Measurements
}

type ModelNumber struct {
	Size        string
	Color       string
	ModelNumber string
}

var (
	ErrInvalidID = errors.New(
		"model: invalid id",
	)

	ErrInvalidProductID = errors.New(
		"model: invalid productId",
	)

	ErrInvalidBlueprintID = errors.New(
		"model: invalid productBlueprintId",
	)

	ErrInvalidModelNumber = errors.New(
		"model: invalid modelNumber",
	)

	ErrInvalidSize = errors.New(
		"model: invalid size",
	)

	ErrInvalidColor = errors.New(
		"model: invalid color",
	)

	ErrInvalidMeasurements = errors.New(
		"model: invalid measurements",
	)

	ErrInvalidUpdatedAt = errors.New(
		"model: invalid updatedAt",
	)

	ErrDuplicateVariationID = errors.New(
		"model: duplicate variation id",
	)

	ErrProductMismatch = errors.New(
		"model: variation.productBlueprintId mismatch",
	)
)

func (
	variation ApparelModelVariation,
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

	if variation.Size == "" {
		return ErrInvalidSize
	}

	if err :=
		variation.Color.Validate(); err != nil {
		return err
	}

	if err :=
		variation.Measurements.Validate(); err != nil {
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
	variation ApparelModelVariation,
) GetID() string {
	return variation.ID
}

func (
	variation ApparelModelVariation,
) GetProductBlueprintID() string {
	return variation.ProductBlueprintID
}

func (
	variation ApparelModelVariation,
) GetKind() ModelVariationKind {
	return ModelVariationKindApparel
}

func (
	variation ApparelModelVariation,
) GetModelNumber() string {
	return variation.ModelNumber
}

// GetDeletionLifecycleは、ポインタを共有しないLifecycleを返します。
func (
	variation ApparelModelVariation,
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
	variation ApparelModelVariation,
) CanModify() bool {
	return canModifyModelDeletionLifecycle(
		variation.DeletionLifecycle,
	)
}

// CanRestoreは指定時刻時点で復旧可能か返します。
func (
	variation ApparelModelVariation,
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
	variation ApparelModelVariation,
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
	variation *ApparelModelVariation,
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
	variation *ApparelModelVariation,
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
	variation *ApparelModelVariation,
) SetMeasurement(
	key string,
	value int,
) error {
	if variation == nil {
		return ErrInvalid
	}

	if !variation.CanModify() {
		return ErrModelModificationForbidden
	}

	if key == "" ||
		value < 0 {
		return ErrInvalidMeasurements
	}

	if variation.Measurements == nil {
		variation.Measurements = make(
			Measurements,
			1,
		)
	}

	variation.Measurements[key] =
		value

	return nil
}

func (
	variation *ApparelModelVariation,
) RemoveMeasurement(
	key string,
) error {
	if variation == nil {
		return ErrInvalid
	}

	if !variation.CanModify() {
		return ErrModelModificationForbidden
	}

	if key == "" {
		return ErrInvalidMeasurements
	}

	if variation.Measurements == nil {
		return nil
	}

	delete(
		variation.Measurements,
		key,
	)

	return nil
}

func (
	variation ApparelModelVariation,
) ToItemSpec() ApparelItemSpec {
	return ApparelItemSpec{
		ModelNumber: variation.ModelNumber,

		Size: variation.Size,

		Color: variation.Color.Name,

		Measurements: variation.
			Measurements.
			Clone(),
	}
}
