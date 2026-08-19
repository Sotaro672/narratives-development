// backend/internal/domain/model/apparel.go
//
// NOTE:
//   - common.go側にModelVariationの共通定義があるため、このファイルでは再定義しない。
//   - apparel専用のvariationはApparelModelVariationとして定義する。
//   - apparel.tops / apparel.bottoms / apparel.dressはColor / Size / Measurementsを使う。
//   - apparel.outerwear / apparel.shoesはColor / Sizeを使い、Measurementsは空でもよい。
//   - apparel.accessory / apparel.bagは原則model variationを作成しない。
//   - Measurementsの必須判定はproductBlueprintCategory/input_schema.goのschemaをapplication/usecase側で参照して行う。
//   - 配送用の梱包後情報はShippingPackageとしてModel variationごとに保持する。
package model

import (
	"errors"
	"time"
)

var (
	ErrProductIDRequired          = errors.New("productId is required")
	ErrVariationIDRequired        = errors.New("variationId is required")
	ErrTargetVariationNotFound    = errors.New("target variation not found")
	ErrNoVariationsFoundForSize   = errors.New("no variations found for size")
	ErrNoVariationsFoundForColor  = errors.New("no variations found for color")
	ErrProductBlueprintIDNotFound = errors.New("product blueprint id not found")
	ErrProductBlueprintNotFound   = errors.New("product blueprint not found")
	ErrVariationNotFound          = errors.New("variation not found")
	ErrInvalidID                  = errors.New("model: invalid id")
	ErrInvalidProductID           = errors.New("model: invalid productId")
	ErrInvalidBlueprintID         = errors.New("model: invalid productBlueprintId")
	ErrInvalidModelNumber         = errors.New("model: invalid modelNumber")
	ErrInvalidSize                = errors.New("model: invalid size")
	ErrInvalidColor               = errors.New("model: invalid color")
	ErrInvalidMeasurements        = errors.New("model: invalid measurements")
	ErrInvalidUpdatedAt           = errors.New("model: invalid updatedAt")
	ErrDuplicateVariationID       = errors.New("model: duplicate variation id")
	ErrProductMismatch            = errors.New("model: variation.productBlueprintId mismatch")
)

// Colorはカラーバリエーションを表す値オブジェクトです。
// RGBは0x000000から0xFFFFFFまでの24bit整数を使用します。
type Color struct {
	Name string
	RGB  int
}

func (color Color) Validate() error {
	if color.Name == "" {
		return ErrInvalidColor
	}

	if color.RGB < 0 || color.RGB > 0xFFFFFF {
		return ErrInvalidColor
	}

	return nil
}

// Measurementsはapparelの採寸値を表します。
// nilと空mapのどちらも有効です。
type Measurements map[string]int

func (measurements Measurements) Validate() error {
	for key, value := range measurements {
		if key == "" || value < 0 {
			return ErrInvalidMeasurements
		}
	}

	return nil
}

func (measurements Measurements) Clone() Measurements {
	if measurements == nil {
		return nil
	}

	output := make(Measurements, len(measurements))
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
	ShippingPackage    ShippingPackage

	CreatedAt time.Time
	CreatedBy *string
	UpdatedAt time.Time
	UpdatedBy *string
}

// NewApparelModelVariationはapparel Model variationの新規作成入力です。
type NewApparelModelVariation struct {
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Color              Color
	Measurements       Measurements
	ShippingPackage    ShippingPackage
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

func (variation ApparelModelVariation) Validate() error {
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

	if err := variation.Color.Validate(); err != nil {
		return err
	}

	if err := variation.Measurements.Validate(); err != nil {
		return err
	}

	if err := variation.ShippingPackage.Validate(); err != nil {
		return err
	}

	return nil
}

func (variation ApparelModelVariation) GetID() string {
	return variation.ID
}

func (variation ApparelModelVariation) GetProductBlueprintID() string {
	return variation.ProductBlueprintID
}

func (variation ApparelModelVariation) GetKind() ModelVariationKind {
	return ModelVariationKindApparel
}

func (variation ApparelModelVariation) GetModelNumber() string {
	return variation.ModelNumber
}

func (variation ApparelModelVariation) GetShippingPackage() ShippingPackage {
	return variation.ShippingPackage
}
