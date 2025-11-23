package productBlueprint

import (
	"errors"
	"fmt"
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

func IsNotFound(err error) bool     { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool     { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool      { return errors.Is(err, ErrInvalid) }
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }
func IsForbidden(err error) bool    { return errors.Is(err, ErrForbidden) }
func IsInternal(err error) bool     { return errors.Is(err, ErrInternal) }

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

// ======================================
// Enums (ItemType)
// ======================================

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

// ======================================
// ProductIDTagType
// ======================================

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

// ======================================
// Value objects
// ======================================

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
	LogoDesignFile *LogoDesignFile
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

// ======================================
// Entity (Variations → VariationIDs に変更)
// ======================================

type ProductBlueprint struct {
	ID               string
	ProductName      string
	BrandID          string
	ItemType         ItemType
	VariationIDs     []string
	Fit              string
	Material         string
	Weight           float64
	QualityAssurance []string
	ProductIdTag     ProductIDTag
	CompanyID        string
	AssigneeID       string
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedBy        *string
	UpdatedAt        time.Time
	DeletedBy        *string
	DeletedAt        *time.Time
}

// ======================================
// Errors
// ======================================

var (
	ErrInvalidID        = errors.New("productBlueprint: invalid id")
	ErrInvalidProduct   = errors.New("productBlueprint: invalid productName")
	ErrInvalidBrand     = errors.New("productBlueprint: invalid brandId")
	ErrInvalidItemType  = errors.New("productBlueprint: invalid itemType")
	ErrInvalidWeight    = errors.New("productBlueprint: invalid weight")
	ErrInvalidTagType   = errors.New("productBlueprint: invalid productIdTag.type")
	ErrInvalidCreatedAt = errors.New("productBlueprint: invalid createdAt")
	ErrInvalidAssignee  = errors.New("productBlueprint: invalid assigneeId")
	ErrInvalidCompanyID = errors.New("productBlueprint: invalid companyId")
)

// ======================================
// Constructors
// ======================================

func New(
	id, productName, brandID string,
	itemType ItemType,
	variationIDs []string,
	fit, material string,
	weight float64,
	qualityAssurance []string,
	productIDTag ProductIDTag,
	assigneeID string,
	createdBy *string,
	createdAt time.Time,
	companyID string,
) (ProductBlueprint, error) {

	pb := ProductBlueprint{
		ID:               strings.TrimSpace(id),
		ProductName:      strings.TrimSpace(productName),
		BrandID:          strings.TrimSpace(brandID),
		ItemType:         itemType,
		VariationIDs:     dedupTrim(variationIDs),
		Fit:              strings.TrimSpace(fit),
		Material:         strings.TrimSpace(material),
		Weight:           weight,
		QualityAssurance: dedupTrim(qualityAssurance),
		ProductIdTag:     productIDTag,
		AssigneeID:       strings.TrimSpace(assigneeID),
		CompanyID:        strings.TrimSpace(companyID),
		CreatedBy:        createdBy,
		CreatedAt:        createdAt,
		UpdatedBy:        createdBy,
		UpdatedAt:        createdAt,
		DeletedBy:        nil,
		DeletedAt:        nil,
	}

	if err := pb.validate(); err != nil {
		return ProductBlueprint{}, err
	}

	return pb, nil
}

func NewFromStringTime(
	id, productName, brandID string,
	itemType ItemType,
	variationIDs []string,
	fit, material string,
	weight float64,
	qualityAssurance []string,
	productIDTag ProductIDTag,
	assigneeID string,
	createdBy *string,
	createdAt string,
	companyID string,
) (ProductBlueprint, error) {

	t, err := parseTime(createdAt)
	if err != nil {
		return ProductBlueprint{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}

	return New(
		id, productName, brandID,
		itemType, variationIDs,
		fit, material, weight,
		qualityAssurance, productIDTag,
		assigneeID, createdBy, t,
		companyID,
	)
}

// ======================================
// Update Methods
// ======================================

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

func (p *ProductBlueprint) UpdateVariationIDs(ids []string, now time.Time, updatedBy *string) {
	p.VariationIDs = dedupTrim(ids)
	p.touch(now, updatedBy)
}

// ======================================
// Validation
// ======================================

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
	if strings.TrimSpace(p.CompanyID) == "" {
		return ErrInvalidCompanyID
	}
	if err := p.ProductIdTag.validate(); err != nil {
		return err
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}

// ======================================
// Helpers
// ======================================

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
