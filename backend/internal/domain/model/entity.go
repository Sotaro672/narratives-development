// backend\internal\domain\model\entity.go
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

// Validation (moved from entity.go)

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
	// Optional audit coherence
	if mv.CreatedAt != nil && mv.UpdatedAt != nil && mv.UpdatedAt.Before(*mv.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

func (md ModelData) validate() error {
	if md.ProductID == "" {
		return ErrInvalidProductID
	}
	if md.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}
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
// Types (mirror TS)
// ==========================

type ModelVariation struct {
	ID                 string
	ProductBlueprintID string
	ModelNumber        string
	Size               string
	Color              string
	Measurements       map[string]float64

	// Audit fields (optional, mirrors TS optional fields)
	CreatedAt *time.Time
	CreatedBy *string
	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

type ModelData struct {
	ProductID          string
	ProductBlueprintID string
	Variations         []ModelVariation
	UpdatedAt          time.Time
}

// Alias to satisfy usecases expecting model.Model.
type Model = ModelData

type ItemSpec struct {
	ModelNumber  string
	Size         string
	Color        string
	Measurements map[string]float64 // nil means "unset"
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
	// productBlueprintId の不一致に合わせてメッセージ更新
	ErrProductMismatch = errors.New("model: variation.productBlueprintId mismatch")
)

// ==========================
// Policy (optional)
// If empty, any non-empty size/color is accepted.
// You can populate these from modelConstants.ts if you want strict checking.
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

// ModelsTableDDL defines the SQL for the model domain migrations.
const ModelsTableDDL = `
-- Migration: Initialize/Update Model domain (model_sets, model_variations)
-- Mirrors backend/internal/domain/model/entity.go

BEGIN;

-- A set of variations for a product (tracks UpdatedAt at product scope)
CREATE TABLE IF NOT EXISTS model_sets (
  product_id           TEXT        PRIMARY KEY,
  product_blueprint_id TEXT        NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_model_sets_non_empty CHECK (
    char_length(trim(product_id)) > 0
    AND char_length(trim(product_blueprint_id)) > 0
  )
);

-- Ensure blueprint is unique to reference from variations
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_sets_product_blueprint_id
  ON model_sets(product_blueprint_id);

-- Each concrete variation (measurements embedded as JSONB)
CREATE TABLE IF NOT EXISTS model_variations (
  id                   TEXT NOT NULL PRIMARY KEY,
  product_blueprint_id TEXT NOT NULL, -- TS: ModelVariation.productBlueprintId
  model_number         TEXT NOT NULL,
  size                 TEXT NOT NULL,
  color                TEXT NOT NULL,
  measurements         JSONB NOT NULL DEFAULT '{}'::jsonb, -- TS: Record<string, number>

  -- Audit (optional in TS)
  created_at TIMESTAMPTZ NULL,
  created_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,
  updated_at TIMESTAMPTZ NULL,
  updated_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by UUID        NULL REFERENCES members(id) ON DELETE RESTRICT,

  CONSTRAINT chk_model_variations_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(product_blueprint_id)) > 0
    AND char_length(trim(model_number)) > 0
    AND char_length(trim(size)) > 0
    AND char_length(trim(color)) > 0
  ),
  -- measurements must be a JSON object
  CONSTRAINT chk_model_variations_measurements_object CHECK (jsonb_typeof(measurements) = 'object'),

  -- Audit coherence (when provided)
  CONSTRAINT chk_model_variations_time_order CHECK (
    updated_at IS NULL OR created_at IS NULL OR updated_at >= created_at
  )
);

-- Relationships
ALTER TABLE model_sets
  ADD CONSTRAINT fk_model_sets_product
  FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

-- variations -> sets by product_blueprint_id
ALTER TABLE model_variations
  ADD CONSTRAINT fk_model_variations_set_by_blueprint
  FOREIGN KEY (product_blueprint_id) REFERENCES model_sets(product_blueprint_id) ON DELETE CASCADE;

-- Uniqueness to avoid duplicates per blueprint
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_variations_blueprint_modelnumber_size_color
  ON model_variations(product_blueprint_id, model_number, size, color);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_model_variations_product_blueprint_id  ON model_variations(product_blueprint_id);
CREATE INDEX IF NOT EXISTS idx_model_variations_model_number          ON model_variations(model_number);
CREATE INDEX IF NOT EXISTS idx_model_variations_size                  ON model_variations(size);
CREATE INDEX IF NOT EXISTS idx_model_variations_color                 ON model_variations(color);
CREATE INDEX IF NOT EXISTS idx_model_variations_measurements_gin      ON model_variations USING GIN (measurements);
CREATE INDEX IF NOT EXISTS idx_model_variations_created_at            ON model_variations (created_at);
CREATE INDEX IF NOT EXISTS idx_model_variations_updated_at            ON model_variations (updated_at);
CREATE INDEX IF NOT EXISTS idx_model_variations_deleted_at            ON model_variations (deleted_at);

COMMIT;
`
