// backend/internal/domain/productBlueprint/entity.go
package productBlueprint

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// 汎用エラー（ドメイン共通）
var (
	ErrNotFound       = errors.New("productBlueprint: not found")
	ErrConflict       = errors.New("productBlueprint: conflict")
	ErrInvalid        = errors.New("productBlueprint: invalid")
	ErrUnauthorized   = errors.New("productBlueprint: unauthorized")
	ErrForbidden      = errors.New("productBlueprint: forbidden")
	ErrInternal       = errors.New("productBlueprint: internal")
	ErrRestoreExpired = errors.New("productBlueprint: restore expired") // ★ TTL 期限切れ復旧不可用
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

type ProductIDTag struct {
	Type ProductIDTagType
}

func (t ProductIDTag) validate() error {
	if !IsValidTagType(t.Type) {
		return ErrInvalidTagType
	}
	return nil
}

// ======================================
// Entity（★ Version を完全削除）
// ======================================

type ProductBlueprint struct {
	ID string

	ProductName string
	CompanyID   string
	BrandID     string
	ItemType    ItemType
	Fit         string
	Material    string
	Weight      float64

	QualityAssurance []string
	ProductIdTag     ProductIDTag
	AssigneeID       string

	// ★ 印刷状態: false=未印刷, true=印刷済み
	Printed bool

	CreatedBy *string
	CreatedAt time.Time
	UpdatedBy *string
	UpdatedAt time.Time

	DeletedBy *string
	DeletedAt *time.Time

	// ★ 物理削除予定日時（Firestore TTL 対象フィールド）
	ExpireAt *time.Time
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
		Fit:              strings.TrimSpace(fit),
		Material:         strings.TrimSpace(material),
		Weight:           weight,
		QualityAssurance: dedupTrim(qualityAssurance),
		ProductIdTag:     productIDTag,
		AssigneeID:       strings.TrimSpace(assigneeID),
		CompanyID:        strings.TrimSpace(companyID),
		// ★ create 時は常に false（未印刷）をセット
		Printed:   false,
		CreatedBy: createdBy,
		CreatedAt: createdAt.UTC(),
		UpdatedBy: createdBy,
		UpdatedAt: createdAt.UTC(),
		DeletedBy: nil,
		DeletedAt: nil,
		ExpireAt:  nil,
	}

	if err := pb.validate(); err != nil {
		return ProductBlueprint{}, err
	}

	return pb, nil
}

func NewFromStringTime(
	id, productName, brandID string,
	itemType ItemType,
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
		itemType,
		fit, material, weight,
		qualityAssurance, productIDTag,
		assigneeID, createdBy, t,
		companyID,
	)
}

// ======================================
// Update Helpers
// ======================================

// Printed == true（印刷済み）の場合に更新・削除を禁止
func (p ProductBlueprint) canModify() bool {
	return !p.Printed
}

// Printed を true（印刷済み）にするための専用メソッド
func (p *ProductBlueprint) MarkPrinted(now time.Time, updatedBy *string) error {
	if p.Printed {
		// すでに printed の場合は何もしない（idempotent）
		return nil
	}
	p.Printed = true
	p.touch(now, updatedBy)
	return nil
}

// ======================================
// Update Methods
// ======================================

func (p *ProductBlueprint) UpdateAssignee(assigneeID string, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden // printed のため更新不可
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ErrInvalidAssignee
	}
	p.AssigneeID = assigneeID
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateQualityAssurance(items []string, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden // printed のため更新不可
	}
	p.QualityAssurance = dedupTrim(items)
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateTag(tag ProductIDTag, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden // printed のため更新不可
	}
	if err := tag.validate(); err != nil {
		return err
	}
	p.ProductIdTag = tag
	p.touch(now, updatedBy)
	return nil
}

// ★ Version 更新機能（BumpVersion）は削除済み

// Soft Delete（論理削除 + TTL セット）
func (p *ProductBlueprint) SoftDelete(now time.Time, deletedBy *string, ttl time.Duration) error {
	if !p.canModify() {
		return ErrForbidden // printed のため削除不可
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}
	p.DeletedAt = &now
	p.DeletedBy = deletedBy

	if ttl > 0 {
		exp := now.Add(ttl)
		p.ExpireAt = &exp
	} else {
		p.ExpireAt = nil
	}

	p.touch(now, deletedBy)
	return nil
}

// 復旧（DeletedAt / DeletedBy / ExpireAt をクリアして Updated 系を進める）
func (p *ProductBlueprint) Restore(now time.Time, restoredBy *string) error {
	if !p.canModify() {
		return ErrForbidden // printed のため更新不可
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	p.DeletedAt = nil
	p.DeletedBy = nil
	p.ExpireAt = nil
	p.touch(now, restoredBy)
	return nil
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
	// Printed は bool のため常に有効
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
