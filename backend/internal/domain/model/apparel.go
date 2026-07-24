// Responsibility: apparel category model variation definitions.
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
)

var (
	ErrProductIDRequired = errors.New("productId is required")

	ErrVariationIDRequired = errors.New("variationId is required")

	ErrTargetVariationNotFound = errors.New("target variation not found")

	ErrNoVariationsFoundForSize = errors.New("no variations found for size")

	ErrNoVariationsFoundForColor = errors.New("no variations found for color")

	ErrProductBlueprintIDNotFound = errors.New("product blueprint id not found")

	ErrProductBlueprintNotFound = errors.New("product blueprint not found")

	ErrVariationNotFound = errors.New("variation not found")
)

// Colorはカラーバリエーションを表す値オブジェクトです。
// RGBは0x000000から0xFFFFFFまでの24bit整数を使用します。
type Color struct {
	Name string
	RGB  int
}

func (c Color) Validate() error {
	if c.Name == "" {
		return ErrInvalidColor
	}

	if c.RGB < 0 || c.RGB > 0xFFFFFF {
		return ErrInvalidColor
	}

	return nil
}

// Measurementsはapparelの採寸値を表します。
// nilと空mapのどちらも有効です。
type Measurements map[string]int

func (m Measurements) Validate() error {
	for key, value := range m {
		if key == "" || value < 0 {
			return ErrInvalidMeasurements
		}
	}

	return nil
}

func (m Measurements) Clone() Measurements {
	if m == nil {
		return nil
	}

	out := make(
		Measurements,
		len(m),
	)

	for key, value := range m {
		out[key] = value
	}

	return out
}

// ApparelModelVariationはapparel用のmodel variationです。
type ApparelModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Measurements       Measurements
	Color              Color
	CreatedAt          time.Time
	CreatedBy          *string
	UpdatedAt          time.Time
	UpdatedBy          *string
}

// NewApparelModelVariationは
// apparel model variationの新規作成入力です。
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
	ErrInvalidID = errors.New("model: invalid id")

	ErrInvalidProductID = errors.New("model: invalid productId")

	ErrInvalidBlueprintID = errors.New("model: invalid productBlueprintId")

	ErrInvalidModelNumber = errors.New("model: invalid modelNumber")

	ErrInvalidSize = errors.New("model: invalid size")

	ErrInvalidColor = errors.New("model: invalid color")

	ErrInvalidMeasurements = errors.New("model: invalid measurements")

	ErrInvalidUpdatedAt = errors.New("model: invalid updatedAt")

	ErrDuplicateVariationID = errors.New("model: duplicate variation id")

	ErrProductMismatch = errors.New(
		"model: variation.productBlueprintId mismatch",
	)
)

func (mv ApparelModelVariation) Validate() error {
	if mv.ID == "" {
		return ErrInvalidID
	}

	if mv.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}

	if mv.ModelNumber == "" {
		return ErrInvalidModelNumber
	}

	if mv.Size == "" {
		return ErrInvalidSize
	}

	if err := mv.Color.Validate(); err != nil {
		return err
	}

	if err := mv.Measurements.Validate(); err != nil {
		return err
	}

	return nil
}

func (mv ApparelModelVariation) GetID() string {
	return mv.ID
}

func (
	mv ApparelModelVariation,
) GetProductBlueprintID() string {
	return mv.ProductBlueprintID
}

func (
	mv ApparelModelVariation,
) GetKind() ModelVariationKind {
	return ModelVariationKindApparel
}

func (
	mv ApparelModelVariation,
) GetModelNumber() string {
	return mv.ModelNumber
}

func (
	mv *ApparelModelVariation,
) SetMeasurement(
	key string,
	value int,
) error {
	if key == "" || value < 0 {
		return ErrInvalidMeasurements
	}

	if mv.Measurements == nil {
		mv.Measurements = make(
			Measurements,
			1,
		)
	}

	mv.Measurements[key] = value

	return nil
}

func (
	mv *ApparelModelVariation,
) RemoveMeasurement(
	key string,
) {
	if mv.Measurements == nil {
		return
	}

	delete(
		mv.Measurements,
		key,
	)
}

func (
	mv ApparelModelVariation,
) ToItemSpec() ApparelItemSpec {
	return ApparelItemSpec{
		ModelNumber:  mv.ModelNumber,
		Size:         mv.Size,
		Color:        mv.Color.Name,
		Measurements: mv.Measurements.Clone(),
	}
}
