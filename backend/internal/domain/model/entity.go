package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
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
// Validation
// ==========================

func (mv ModelVariation) validate() error {
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
	if mv.Color == "" {
		return ErrInvalidColor
	}
	if !sizeAllowed(mv.Size) {
		return ErrInvalidSize
	}
	if !colorAllowed(mv.Color) {
		return ErrInvalidColor
	}

	for k, v := range mv.Measurements {
		if strings.TrimSpace(k) == "" || math.IsNaN(v) || math.IsInf(v, 0) {
			return ErrInvalidMeasurements
		}
	}

	// createdAt / updatedAt の検証は削除

	return nil
}

func (md ModelData) validate() error {
	if md.ProductID == "" {
		return ErrInvalidProductID
	}
	if md.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}

	// UpdatedAt は必須のまま
	if md.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	seen := make(map[string]struct{}, len(md.Variations))
	for _, v := range md.Variations {
		if err := v.validate(); err != nil {
			return err
		}
		if v.ProductBlueprintID != md.ProductBlueprintID {
			return ErrProductMismatch
		}
		if _, dup := seen[v.ID]; dup {
			return ErrDuplicateVariationID
		}
		seen[v.ID] = struct{}{}
	}
	return nil
}

// ==========================
// Types
// ==========================

type ModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Color              string
	Measurements       map[string]float64

	// 削除した: CreatedAt, CreatedBy, UpdatedAt, UpdatedBy
	// 残す: DeletedAt / DeletedBy
	DeletedAt *time.Time
	DeletedBy *string
}

type ModelData struct {
	ProductID          string
	ProductBlueprintID string
	Variations         []ModelVariation
	UpdatedAt          time.Time
}

type Model = ModelData

type ItemSpec struct {
	ModelNumber  string
	Size         string
	Color        string
	Measurements map[string]float64
}

type SizeVariation struct {
	ID           string
	Size         string
	Measurements map[string]float64
}

type ModelNumber struct {
	Size        string
	Color       string
	ModelNumber string
}

type ProductionQuantity struct {
	Size     string
	Color    string
	Quantity int
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
// Constructors
// ==========================

func NewModelData(
	productID, productBlueprintID string,
	variations []ModelVariation,
	updatedAt time.Time,
) (ModelData, error) {
	md := ModelData{
		ProductID:          strings.TrimSpace(productID),
		ProductBlueprintID: strings.TrimSpace(productBlueprintID),
		Variations:         append([]ModelVariation(nil), variations...),
		UpdatedAt:          updatedAt,
	}
	if err := md.validate(); err != nil {
		return ModelData{}, err
	}
	return md, nil
}

func NewModelDataFromStringTime(
	productID, productBlueprintID string,
	variations []ModelVariation,
	updatedAt string,
) (ModelData, error) {
	t, err := parseTime(updatedAt)
	if err != nil {
		return ModelData{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return NewModelData(productID, productBlueprintID, variations, t)
}

// ==========================
// Behavior
// ==========================

func (mv *ModelVariation) SetMeasurement(key string, value float64) error {
	key = strings.TrimSpace(key)
	if key == "" || math.IsNaN(value) || math.IsInf(value, 0) {
		return ErrInvalidMeasurements
	}
	if mv.Measurements == nil {
		mv.Measurements = make(map[string]float64, 1)
	}
	mv.Measurements[key] = value
	return nil
}

func (mv *ModelVariation) RemoveMeasurement(key string) {
	if mv.Measurements == nil {
		return
	}
	delete(mv.Measurements, key)
}

func (mv ModelVariation) ToItemSpec() ItemSpec {
	return ItemSpec{
		ModelNumber:  mv.ModelNumber,
		Size:         mv.Size,
		Color:        mv.Color,
		Measurements: cloneMeasurements(mv.Measurements),
	}
}

func (md *ModelData) AddVariation(v ModelVariation, now time.Time) error {
	if v.ProductBlueprintID != md.ProductBlueprintID {
		return ErrProductMismatch
	}
	for _, cur := range md.Variations {
		if cur.ID == v.ID {
			return ErrDuplicateVariationID
		}
	}
	md.Variations = append(md.Variations, v)
	md.touch(now)
	return nil
}

func (md *ModelData) ReplaceVariations(vars []ModelVariation, now time.Time) error {
	seen := make(map[string]struct{}, len(vars))
	for i := range vars {
		if vars[i].ProductBlueprintID != md.ProductBlueprintID {
			return ErrProductMismatch
		}
		if _, dup := seen[vars[i].ID]; dup {
			return ErrDuplicateVariationID
		}
		seen[vars[i].ID] = struct{}{}
	}
	md.Variations = append([]ModelVariation(nil), vars...)
	md.touch(now)
	return nil
}

func (md *ModelData) FindVariationByID(id string) (*ModelVariation, bool) {
	for i := range md.Variations {
		if md.Variations[i].ID == id {
			return &md.Variations[i], true
		}
	}
	return nil, false
}

func (md *ModelData) FindVariationByModelNumber(mn string) (*ModelVariation, bool) {
	for i := range md.Variations {
		if md.Variations[i].ModelNumber == mn {
			return &md.Variations[i], true
		}
	}
	return nil, false
}

// ==========================
// Helpers
// ==========================

func (md *ModelData) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	md.UpdatedAt = now
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrInvalidUpdatedAt
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func sizeAllowed(size string) bool {
	if len(AllowedSizes) == 0 {
		return true
	}
	_, ok := AllowedSizes[size]
	return ok
}

func colorAllowed(color string) bool {
	if len(AllowedColors) == 0 {
		return true
	}
	_, ok := AllowedColors[color]
	return ok
}

func cloneMeasurements(m map[string]float64) map[string]float64 {
	if m == nil {
		return nil
	}
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
