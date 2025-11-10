package productBlueprint

import (
	"errors"
	"fmt"
	model "narratives/internal/domain/model"
	"net/url"
	"strings"
	"time"
)

// 汎用エラー（ドメイン共通）
var (
	ErrNotFound     = errors.New("productBlueprint: not found")
	ErrConflict     = errors.New("productBlueprint: conflict")
	ErrInvalid      = errors.New("productBlueprint: invalid")
	ErrUnauthorized = errors.New("productBlueprint: unauthorized")
	ErrForbidden    = errors.New("productBlueprint: forbidden")
	ErrInternal     = errors.New("productBlueprint: internal")
)

// 補助ヘルパー
func IsNotFound(err error) bool     { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool     { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool      { return errors.Is(err, ErrInvalid) }
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }
func IsForbidden(err error) bool    { return errors.Is(err, ErrForbidden) }
func IsInternal(err error) bool     { return errors.Is(err, ErrInternal) }

// ラップ用ヘルパー（原因を保持）
func WrapInvalid(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalid, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrInvalid, msg, err)
}

func WrapConflict(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrConflict, msg, err)
}

func WrapNotFound(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrNotFound, msg, err)
}

// Enums (mirrors TS)
type ItemType string

const (
	ItemTops    ItemType = "tops"
	ItemBottoms ItemType = "bottoms"
	ItemOther   ItemType = "other"
)

func IsValidItemType(v ItemType) bool {
	switch v {
	case ItemTops, ItemBottoms, ItemOther:
		return true
	default:
		return false
	}
}

// strings.ToLower にそのまま渡せるよう、型エイリアスに変更
type ProductIDTagType = string

const (
	TagQR  ProductIDTagType = "qr"
	TagNFC ProductIDTagType = "nfc"
)

func IsValidTagType(v ProductIDTagType) bool {
	switch v {
	case TagQR, TagNFC:
		return true
	default:
		return false
	}
}

// Value objects
type LogoDesignFile struct {
	Name string
	URL  string
}

func (f LogoDesignFile) validate() error {
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("logoDesignFile: name required")
	}
	if _, err := url.ParseRequestURI(f.URL); err != nil {
		return fmt.Errorf("logoDesignFile: invalid url: %w", err)
	}
	return nil
}

type ProductIDTag struct {
	Type           ProductIDTagType
	LogoDesignFile *LogoDesignFile // optional
}

func (t ProductIDTag) validate() error {
	if !IsValidTagType(t.Type) {
		return ErrInvalidTagType
	}
	if t.LogoDesignFile != nil {
		if err := t.LogoDesignFile.validate(); err != nil {
			return err
		}
	}
	return nil
}

// Entity
type ProductBlueprint struct {
	ID               string
	ProductName      string
	BrandID          string
	ItemType         ItemType
	Variations       []model.ModelVariation
	Fit              string
	Material         string
	Weight           float64
	QualityAssurance []string
	ProductIdTag     ProductIDTag `json:"-" db:"-"`
	AssigneeID       string
	CreatedBy        *string   // TS: string | null
	CreatedAt        time.Time // TS: Date | string
	UpdatedBy        *string   // TS: string | null
	UpdatedAt        time.Time // TS: Date | string
}

// Errors
var (
	ErrInvalidID        = errors.New("productBlueprint: invalid id")
	ErrInvalidProduct   = errors.New("productBlueprint: invalid productName")
	ErrInvalidBrand     = errors.New("productBlueprint: invalid brandId")
	ErrInvalidItemType  = errors.New("productBlueprint: invalid itemType")
	ErrInvalidWeight    = errors.New("productBlueprint: invalid weight")
	ErrInvalidTagType   = errors.New("productBlueprint: invalid productIdTag.type")
	ErrInvalidCreatedAt = errors.New("productBlueprint: invalid createdAt")
	ErrInvalidAssignee  = errors.New("productBlueprint: invalid assigneeId")
)

// Constructors
func New(
	id, productName, brandID string,
	itemType ItemType,
	variations []model.ModelVariation,
	fit, material string,
	weight float64,
	qualityAssurance []string,
	productIDTag ProductIDTag,
	assigneeID string,
	createdBy *string,
	createdAt time.Time,
) (ProductBlueprint, error) {
	pb := ProductBlueprint{
		ID:               strings.TrimSpace(id),
		ProductName:      strings.TrimSpace(productName),
		BrandID:          strings.TrimSpace(brandID),
		ItemType:         itemType,
		Variations:       dedupVariationsByID(variations),
		Fit:              strings.TrimSpace(fit),
		Material:         strings.TrimSpace(material),
		Weight:           weight,
		QualityAssurance: dedupTrim(qualityAssurance),
		ProductIdTag:     productIDTag,
		AssigneeID:       strings.TrimSpace(assigneeID),
		CreatedBy:        createdBy,
		CreatedAt:        createdAt,
		UpdatedBy:        createdBy,
		UpdatedAt:        createdAt,
	}
	if err := pb.validate(); err != nil {
		return ProductBlueprint{}, err
	}
	return pb, nil
}

func NewFromStringTime(
	id, productName, brandID string,
	itemType ItemType,
	variations []model.ModelVariation,
	fit, material string,
	weight float64,
	qualityAssurance []string,
	productIDTag ProductIDTag,
	assigneeID string,
	createdBy *string,
	createdAt string,
) (ProductBlueprint, error) {
	t, err := parseTime(createdAt)
	if err != nil {
		return ProductBlueprint{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	return New(
		id, productName, brandID,
		itemType, variations,
		fit, material, weight,
		qualityAssurance, productIDTag,
		assigneeID, createdBy, t,
	)
}

// 更新系メソッド
func (p *ProductBlueprint) UpdateAssignee(assigneeID string, now time.Time, updatedBy *string) error {
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ErrInvalidAssignee
	}
	p.AssigneeID = assigneeID
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateQualityAssurance(items []string, now time.Time, updatedBy *string) {
	p.QualityAssurance = dedupTrim(items)
	p.touch(now, updatedBy)
}

func (p *ProductBlueprint) UpdateTag(tag ProductIDTag, now time.Time, updatedBy *string) error {
	if err := tag.validate(); err != nil {
		return err
	}
	p.ProductIdTag = tag
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateVariations(vars []model.ModelVariation, now time.Time, updatedBy *string) {
	p.Variations = dedupVariationsByID(vars)
	p.touch(now, updatedBy)
}

// Validation
func (p ProductBlueprint) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.ProductName == "" {
		return ErrInvalidProduct
	}
	if p.BrandID == "" {
		return ErrInvalidBrand
	}
	if !IsValidItemType(p.ItemType) {
		return ErrInvalidItemType
	}
	if p.Weight < 0 {
		return ErrInvalidWeight
	}
	if err := p.ProductIdTag.validate(); err != nil {
		return err
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}

// Helpers
func (p *ProductBlueprint) touch(now time.Time, updatedBy *string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	p.UpdatedAt = now
	p.UpdatedBy = updatedBy
}

func parseTime(s string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, ErrInvalidCreatedAt
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
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func dedupVariationsByID(vars []model.ModelVariation) []model.ModelVariation {
	seen := make(map[string]struct{}, len(vars))
	out := make([]model.ModelVariation, 0, len(vars))
	for _, v := range vars {
		v.ID = strings.TrimSpace(v.ID)
		if v.ID == "" {
			continue
		}
		if _, ok := seen[v.ID]; ok {
			continue
		}
		seen[v.ID] = struct{}{}
		out = append(out, v)
	}
	return out
}

// ProductBlueprintsTableDDL defines the SQL for migration.
const ProductBlueprintsTableDDL = `
BEGIN;

CREATE TABLE IF NOT EXISTS product_blueprints (
  id                     TEXT        PRIMARY KEY,
  product_name           TEXT        NOT NULL,
  brand_id               TEXT        NOT NULL,
  item_type              TEXT        NOT NULL CHECK (item_type IN ('tops','bottoms','other')),
  fit                    TEXT        NOT NULL DEFAULT '',
  material               TEXT        NOT NULL DEFAULT '',
  weight                 DOUBLE PRECISION NOT NULL CHECK (weight >= 0),
  quality_assurance      TEXT[]      NOT NULL DEFAULT '{}',
  product_id_tag_type    TEXT        NOT NULL CHECK (product_id_tag_type IN ('qr','nfc')),
  model_variations       JSONB       NOT NULL DEFAULT '[]'::jsonb,
  assignee_id            TEXT        NOT NULL,
  created_by             TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by             TEXT,
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_pb_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(product_name)) > 0
    AND char_length(trim(brand_id)) > 0
    AND char_length(trim(assignee_id)) > 0
  ),
  CONSTRAINT chk_pb_qa_no_empty CHECK (
    NOT EXISTS (SELECT 1 FROM unnest(quality_assurance) t(x) WHERE x = '')
  ),
  CONSTRAINT chk_pb_model_variations_array CHECK (jsonb_typeof(model_variations) = 'array'),
  CHECK (updated_at >= created_at)
);

ALTER TABLE product_blueprints
  ADD CONSTRAINT fk_pb_brand
  FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE RESTRICT;

ALTER TABLE product_blueprints
  ADD CONSTRAINT fk_pb_assignee
  FOREIGN KEY (assignee_id) REFERENCES members(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_pb_brand_id   ON product_blueprints(brand_id);
CREATE INDEX IF NOT EXISTS idx_pb_created_at ON product_blueprints(created_at);

COMMIT;
`
