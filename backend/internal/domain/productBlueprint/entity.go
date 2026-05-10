// backend/internal/domain/productBlueprint/entity.go
package productBlueprint

import (
	"errors"
	"fmt"
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

// ProductBlueprintCategorySnapshot は productBlueprint 側へ denormalize 保存するカテゴリ表示用情報。
// 正のカテゴリ定義は productBlueprintCategory ドメイン / productBlueprintCategories collection 側で管理する。
type ProductBlueprintCategorySnapshot struct {
	ID     string
	Code   string
	NameJa string
	NameEn string
	Kind   string
	Path   []string
}

// Validate は package 外から ProductBlueprintCategorySnapshot を検証するための公開メソッド。
// Firestore repository / usecase / handler など productBlueprint package 外から利用する。
func (s ProductBlueprintCategorySnapshot) Validate() error {
	return s.validate()
}

func (s ProductBlueprintCategorySnapshot) validate() error {
	if s.ID == "" {
		return ErrInvalidCategoryID
	}
	if s.Code == "" {
		return ErrInvalidCategoryCode
	}
	if s.NameJa == "" {
		return ErrInvalidCategoryNameJa
	}
	if s.Kind == "" {
		return ErrInvalidCategoryKind
	}
	return nil
}

// ======================================
// Model references (modelIds + displayOrder)
// ======================================

// ModelRef は productBlueprint 配下に紐づく model の参照を表す。
// - ModelID: model テーブルの docId
// - DisplayOrder: 表示順（1..N の採番）
type ModelRef struct {
	ModelID      string
	DisplayOrder int
}

// normalizeModelRefs は、modelIds を正規化し displayOrder を 1..N で採番して返す。
// - 入力の順序を保持する（caller 側で「色登録順→サイズ登録順」に並べてから渡す前提）
// - 空、重複は除外する（順序は保持）
// - displayOrder は採番し直す（連番）
func normalizeModelRefs(modelIDs []string) []ModelRef {
	seen := make(map[string]struct{}, len(modelIDs))
	out := make([]ModelRef, 0, len(modelIDs))

	order := 1
	for _, id := range modelIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		out = append(out, ModelRef{
			ModelID:      id,
			DisplayOrder: order,
		})
		order++
	}
	return out
}

// mergeAndRenumberModelRefs は既存 + 追加入力をマージし、重複排除しつつ displayOrder を 1..N で採番し直す。
// - 既存の順序を維持した上で、追加分を末尾に足す
// - 空、重複は除外
func mergeAndRenumberModelRefs(existing []ModelRef, appendIDs []string) []ModelRef {
	seen := make(map[string]struct{}, len(existing)+len(appendIDs))
	outIDs := make([]string, 0, len(existing)+len(appendIDs))

	for _, r := range existing {
		id := r.ModelID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		outIDs = append(outIDs, id)
	}

	for _, id := range appendIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		outIDs = append(outIDs, id)
	}

	return normalizeModelRefs(outIDs)
}

// ======================================
// Entity
// ======================================

type ProductBlueprint struct {
	ID string

	ProductName string
	CompanyID   string
	BrandID     string

	ProductBlueprintCategory ProductBlueprintCategorySnapshot

	Fit      string
	Material string
	Weight   float64

	QualityAssurance []string
	ProductIdTag     ProductIDTag
	AssigneeID       string

	// modelIds（model テーブルの docId）と displayOrder を保持
	ModelRefs []ModelRef

	// 印刷状態: false=未印刷, true=印刷済み
	Printed bool

	CreatedBy *string
	CreatedAt time.Time
	UpdatedBy *string
	UpdatedAt time.Time
}

// ======================================
// Errors
// ======================================

var (
	ErrInvalidID        = errors.New("productBlueprint: invalid id")
	ErrInvalidProduct   = errors.New("productBlueprint: invalid productName")
	ErrInvalidBrand     = errors.New("productBlueprint: invalid brandId")
	ErrInvalidWeight    = errors.New("productBlueprint: invalid weight")
	ErrInvalidTagType   = errors.New("productBlueprint: invalid productIdTag.type")
	ErrInvalidCreatedAt = errors.New("productBlueprint: invalid createdAt")
	ErrInvalidAssignee  = errors.New("productBlueprint: invalid assigneeId")
	ErrInvalidCompanyID = errors.New("productBlueprint: invalid companyId")

	ErrInvalidCategoryID     = errors.New("productBlueprint: invalid productBlueprintCategory.id")
	ErrInvalidCategoryCode   = errors.New("productBlueprint: invalid productBlueprintCategory.code")
	ErrInvalidCategoryNameJa = errors.New("productBlueprint: invalid productBlueprintCategory.nameJa")
	ErrInvalidCategoryKind   = errors.New("productBlueprint: invalid productBlueprintCategory.kind")
)

// ======================================
// Constructors
// ======================================

func New(
	id, productName, brandID string,
	category ProductBlueprintCategorySnapshot,
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
		ID:                       id,
		ProductName:              productName,
		BrandID:                  brandID,
		ProductBlueprintCategory: category,
		Fit:                      fit,
		Material:                 material,
		Weight:                   weight,
		QualityAssurance:         dedupKeepOrder(qualityAssurance),
		ProductIdTag:             productIDTag,
		AssigneeID:               assigneeID,
		CompanyID:                companyID,

		// create 時点では modelRefs は空
		ModelRefs: nil,

		// create 時は常に false（未印刷）
		Printed:   false,
		CreatedBy: createdBy,
		CreatedAt: createdAt.UTC(),
		UpdatedBy: createdBy,
		UpdatedAt: createdAt.UTC(),
	}

	if err := pb.validate(); err != nil {
		return ProductBlueprint{}, err
	}

	return pb, nil
}

// ======================================
// Update Helpers
// ======================================

// Printed == true（印刷済み）の場合に更新を禁止
func (p ProductBlueprint) canModify() bool {
	return !p.Printed
}

// Printed を true（印刷済み）にするための専用メソッド
func (p *ProductBlueprint) MarkPrinted(now time.Time, updatedBy *string) error {
	if p.Printed {
		return nil
	}
	p.Printed = true
	p.touch(now, updatedBy)
	return nil
}

// ======================================
// Update Methods
// ======================================

func (p *ProductBlueprint) UpdateCategory(category ProductBlueprintCategorySnapshot, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden
	}
	if err := category.validate(); err != nil {
		return err
	}

	p.ProductBlueprintCategory = category
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateAssignee(assigneeID string, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden
	}

	if assigneeID == "" {
		return ErrInvalidAssignee
	}
	p.AssigneeID = assigneeID
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateQualityAssurance(items []string, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden
	}
	p.QualityAssurance = dedupKeepOrder(items)
	p.touch(now, updatedBy)
	return nil
}

func (p *ProductBlueprint) UpdateTag(tag ProductIDTag, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden
	}
	if err := tag.validate(); err != nil {
		return err
	}
	p.ProductIdTag = tag
	p.touch(now, updatedBy)
	return nil
}

// UpdateModelIDs は「通常の更新」として modelIDs を受け取り、displayOrder を採番して置き換える。
// - 入力順を保持して displayOrder を 1..N で採番
// - printed の場合は更新不可
// - updatedAt / updatedBy は更新される
func (p *ProductBlueprint) UpdateModelIDs(modelIDs []string, now time.Time, updatedBy *string) error {
	if !p.canModify() {
		return ErrForbidden
	}
	p.ModelRefs = normalizeModelRefs(modelIDs)
	p.touch(now, updatedBy)
	return nil
}

// AppendModelIDsNoTouch は「起票後追記」専用の更新メソッド。
// 要件: updatedAt / updatedBy を更新しない。
// - printed の場合は追記不可
// - 既存順序を保持しつつ追記し、displayOrder は 1..N で採番し直す
func (p *ProductBlueprint) AppendModelIDsNoTouch(modelIDs []string) error {
	if !p.canModify() {
		return ErrForbidden
	}
	p.ModelRefs = mergeAndRenumberModelRefs(p.ModelRefs, modelIDs)
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
	if err := p.ProductBlueprintCategory.validate(); err != nil {
		return err
	}
	if p.Weight < 0 {
		return ErrInvalidWeight
	}
	if p.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if err := p.ProductIdTag.validate(); err != nil {
		return err
	}
	if p.AssigneeID == "" {
		return ErrInvalidAssignee
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	for _, r := range p.ModelRefs {
		if r.ModelID == "" {
			return WrapInvalid(nil, "modelRefs.modelId is empty")
		}
		if r.DisplayOrder <= 0 {
			return WrapInvalid(nil, "modelRefs.displayOrder must be > 0")
		}
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
	p.UpdatedAt = now.UTC()
	p.UpdatedBy = updatedBy
}

func dedupKeepOrder(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
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
