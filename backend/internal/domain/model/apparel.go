// backend/internal/domain/model/apparel.go
// Responsibility: apparel category model variation definitions.
//
// NOTE:
//   - common.go 側に ModelVariation / ModelData / Model の共通定義があるため、
//     このファイルでは再定義しない。
//   - apparel 専用の variation は ApparelModelVariation として定義する。
//   - apparel.tops / apparel.bottoms / apparel.dress は Color / Size / Measurements を使う。
//   - apparel.outerwear / apparel.shoes は Color / Size を使い、Measurements は空でもよい。
//   - apparel.accessory / apparel.bag は原則 model variation を作成しない。
//   - どのカテゴリで Measurements を必須にするかは、
//     productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
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
)

// ==========================
// Types
// ==========================

// Color はカラーバリエーションを表す値オブジェクト。
// - Name: 表示名（例: "Green", "ネイビー"）
// - RGB : 0xRRGGBB などの int 表現を想定
type Color struct {
	Name string
	RGB  int
}

// Measurements は apparel の採寸値を表すマップ型エイリアス。
// 例:
// - shoulderWidth
// - bodyWidth
// - bodyLength
// - sleeveLength
// - waist
// - hip
// - inseam
//
// NOTE:
// apparel.tops / apparel.bottoms / apparel.dress では measurements を使う。
// apparel.outerwear / apparel.shoes では measurements は空でもよい。
// apparel.accessory / apparel.bag は原則 model variation 自体を作成しない。
//
// Firestore アダプタなどから model.Measurements 型として利用する。
type Measurements = map[string]int

// ApparelModelVariation は apparel 用の model variation。
//
// category ごとの扱い:
//   - apparel.tops:
//     Color / Size / Measurements を使う
//   - apparel.bottoms:
//     Color / Size / Measurements を使う
//   - apparel.dress:
//     Color / Size / Measurements を使う
//   - apparel.outerwear:
//     Color / Size を使う。Measurements は空でもよい
//   - apparel.shoes:
//     Color / Size を使う。Measurements は空でもよい
//   - apparel.accessory:
//     原則 model variation を作成しない
//   - apparel.bag:
//     原則 model variation を作成しない
//
// NOTE:
// ModelNumber は現行実装で必須として扱う。
// 画面上の入力仕様から外す場合でも、既存の model 参照・表示・在庫連携が
// ModelNumber を前提にしている可能性があるため、現時点では必須を維持する。
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

// NewApparelModelVariation は apparel model variation の新規作成時に使う入力モデル。
// 既存の ApparelModelVariation から ID や監査情報だけを省いた形。
//
// NOTE:
// Measurements は nil / 空 map を許容する。
// measurements 必須カテゴリかどうかは、この domain ではなく
// application/usecase 側で productBlueprintCategory schema を参照して判定する。
type NewApparelModelVariation struct {
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Color              Color
	Measurements       Measurements
}

// ApparelItemSpec は apparel model variation を商品個体・表示用途向けに
// 簡易化した read model 的な値。
// Measurements はカテゴリによって nil / 空 map の場合がある。
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

// ==========================
// Errors
// ==========================

var (
	ErrInvalidID            = errors.New("model: invalid id")
	ErrInvalidProductID     = errors.New("model: invalid productId")
	ErrInvalidBlueprintID   = errors.New("model: invalid productBlueprintId")
	ErrInvalidModelNumber   = errors.New("model: invalid modelNumber")
	ErrInvalidSize          = errors.New("model: invalid size")
	ErrInvalidColor         = errors.New("model: invalid color")
	ErrInvalidMeasurements  = errors.New("model: invalid measurements")
	ErrInvalidUpdatedAt     = errors.New("model: invalid updatedAt")
	ErrDuplicateVariationID = errors.New("model: duplicate variation id")
	ErrProductMismatch      = errors.New("model: variation.productBlueprintId mismatch")
)

// ==========================
// Policy
// ==========================

var AllowedSizes = map[string]struct{}{}
var AllowedColors = map[string]struct{}{}

// ==========================
// Validation
// ==========================

func (mv ApparelModelVariation) Validate() error {
	return mv.validate()
}

func (mv ApparelModelVariation) validate() error {
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
	if mv.Color.Name == "" {
		return ErrInvalidColor
	}
	if mv.Color.RGB < 0 {
		return ErrInvalidColor
	}
	if !sizeAllowed(mv.Size) {
		return ErrInvalidSize
	}
	if !colorAllowed(mv.Color.Name) {
		return ErrInvalidColor
	}

	// Measurements は nil / 空 map を許容する。
	// measurements 必須カテゴリかどうかは productBlueprintCategory schema を
	// application/usecase 側で参照して判定する。
	for k, v := range mv.Measurements {
		if k == "" {
			return ErrInvalidMeasurements
		}
		if v < 0 {
			return ErrInvalidMeasurements
		}
	}

	// CreatedAt / UpdatedAt はゼロ値許容（リポジトリや Usecase 側で設定）
	return nil
}

// ==========================
// Common interface helpers
// ==========================

func (mv ApparelModelVariation) GetID() string {
	return mv.ID
}

func (mv ApparelModelVariation) GetProductBlueprintID() string {
	return mv.ProductBlueprintID
}

func (mv ApparelModelVariation) GetModelNumber() string {
	return mv.ModelNumber
}

// ==========================
// Behavior（ApparelModelVariation）
// ==========================

func (mv *ApparelModelVariation) SetMeasurement(key string, value int) error {
	if key == "" || value < 0 {
		return ErrInvalidMeasurements
	}
	if mv.Measurements == nil {
		mv.Measurements = make(Measurements, 1)
	}
	mv.Measurements[key] = value

	return nil
}

func (mv *ApparelModelVariation) RemoveMeasurement(key string) {
	if mv.Measurements == nil {
		return
	}
	delete(mv.Measurements, key)
}

func (mv ApparelModelVariation) ToItemSpec() ApparelItemSpec {
	return ApparelItemSpec{
		ModelNumber:  mv.ModelNumber,
		Size:         mv.Size,
		Color:        mv.Color.Name,
		Measurements: cloneMeasurements(mv.Measurements),
	}
}

// ==========================
// Helpers
// ==========================

func sizeAllowed(size string) bool {
	if len(AllowedSizes) == 0 {
		return true
	}
	_, ok := AllowedSizes[size]

	return ok
}

func colorAllowed(colorName string) bool {
	if len(AllowedColors) == 0 {
		return true
	}
	_, ok := AllowedColors[colorName]

	return ok
}

func cloneMeasurements(m Measurements) Measurements {
	if m == nil {
		return nil
	}
	out := make(Measurements, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}
