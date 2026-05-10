// backend/internal/domain/productBlueprintCategory/entity.go
package productBlueprintCategory

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ======================================
// Domain errors
// ======================================

var (
	ErrNotFound     = errors.New("productBlueprintCategory: not found")
	ErrConflict     = errors.New("productBlueprintCategory: conflict")
	ErrInvalid      = errors.New("productBlueprintCategory: invalid")
	ErrUnauthorized = errors.New("productBlueprintCategory: unauthorized")
	ErrForbidden    = errors.New("productBlueprintCategory: forbidden")
	ErrInternal     = errors.New("productBlueprintCategory: internal")
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
// Value types
// ======================================

type CategoryID string
type CategoryCode string
type CategoryKind string

const (
	CategoryKindApparel    CategoryKind = "apparel"
	CategoryKindFood       CategoryKind = "food"
	CategoryKindAlcohol    CategoryKind = "alcohol"
	CategoryKindCosmetics  CategoryKind = "cosmetics"
	CategoryKindGoods      CategoryKind = "goods"
	CategoryKindHealthcare CategoryKind = "healthcare"
	CategoryKindOther      CategoryKind = "other"
)

func IsValidCategoryKind(v CategoryKind) bool {
	switch v {
	case CategoryKindApparel,
		CategoryKindFood,
		CategoryKindAlcohol,
		CategoryKindCosmetics,
		CategoryKindGoods,
		CategoryKindHealthcare,
		CategoryKindOther:
		return true
	default:
		return false
	}
}

// ======================================
// Entity
// ======================================

// ProductBlueprintCategory は productBlueprint が参照する商品カテゴリマスタ。
// productBlueprint 側には categoryId と denormalized 表示用フィールドを保存し、
// カテゴリ定義の正はこのドメインで管理する。
type ProductBlueprintCategory struct {
	ID CategoryID

	// 例:
	// - apparel
	// - apparel.tops
	// - apparel.bottoms
	// - food
	// - alcohol
	// - alcohol.sake
	// - alcohol.wine
	// - cosmetics
	Code CategoryCode

	// 表示名
	NameJa string
	NameEn string

	// 親カテゴリ。トップ階層の場合は nil。
	ParentID *CategoryID

	// 階層パス。
	// 例: ["alcohol", "sake"]
	Path []string

	// 大分類
	Kind CategoryKind

	// 表示順
	DisplayOrder int

	// カテゴリごとの追加要件。
	// 飲食品、酒類、化粧品などのカテゴリ差分を productBlueprint 本体へ直書きしないために持つ。
	Attributes CategoryAttributes

	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryAttributes struct {
	RequiresExpirationDate bool
	RequiresLotNumber      bool
	RequiresIngredients    bool
	RequiresAlcoholNotice  bool
	RequiresCosmeticNotice bool
	RequiresStorageMethod  bool
}

// Snapshot は productBlueprint 側へ denormalize して保存する表示用カテゴリ情報。
// 正は ProductBlueprintCategory だが、一覧表示・検索・表示高速化のため productBlueprint にコピーしてよい。
type Snapshot struct {
	ID     CategoryID
	Code   CategoryCode
	NameJa string
	NameEn string
	Kind   CategoryKind
	Path   []string
}

// ======================================
// Validation errors
// ======================================

var (
	ErrInvalidID           = errors.New("productBlueprintCategory: invalid id")
	ErrInvalidCode         = errors.New("productBlueprintCategory: invalid code")
	ErrInvalidNameJa       = errors.New("productBlueprintCategory: invalid nameJa")
	ErrInvalidKind         = errors.New("productBlueprintCategory: invalid kind")
	ErrInvalidPath         = errors.New("productBlueprintCategory: invalid path")
	ErrInvalidDisplayOrder = errors.New("productBlueprintCategory: invalid displayOrder")
	ErrInvalidCreatedAt    = errors.New("productBlueprintCategory: invalid createdAt")
	ErrInvalidUpdatedAt    = errors.New("productBlueprintCategory: invalid updatedAt")
)

// ======================================
// Constructors
// ======================================

func New(
	id CategoryID,
	code CategoryCode,
	nameJa string,
	nameEn string,
	parentID *CategoryID,
	path []string,
	kind CategoryKind,
	displayOrder int,
	attributes CategoryAttributes,
	now time.Time,
) (ProductBlueprintCategory, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	category := ProductBlueprintCategory{
		ID:           id,
		Code:         code,
		NameJa:       nameJa,
		NameEn:       nameEn,
		ParentID:     parentID,
		Path:         normalizePath(path),
		Kind:         kind,
		DisplayOrder: displayOrder,
		Attributes:   attributes,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}

	if err := category.validate(); err != nil {
		return ProductBlueprintCategory{}, err
	}

	return category, nil
}

// Reconstruct は永続化層から復元するときに使う。
// CreatedAt / UpdatedAt を維持したまま validate する。
func Reconstruct(
	id CategoryID,
	code CategoryCode,
	nameJa string,
	nameEn string,
	parentID *CategoryID,
	path []string,
	kind CategoryKind,
	displayOrder int,
	attributes CategoryAttributes,
	createdAt time.Time,
	updatedAt time.Time,
) (ProductBlueprintCategory, error) {
	category := ProductBlueprintCategory{
		ID:           id,
		Code:         code,
		NameJa:       nameJa,
		NameEn:       nameEn,
		ParentID:     parentID,
		Path:         normalizePath(path),
		Kind:         kind,
		DisplayOrder: displayOrder,
		Attributes:   attributes,
		CreatedAt:    createdAt.UTC(),
		UpdatedAt:    updatedAt.UTC(),
	}

	if err := category.validate(); err != nil {
		return ProductBlueprintCategory{}, err
	}

	return category, nil
}

// ======================================
// Update methods
// ======================================

func (c *ProductBlueprintCategory) Rename(nameJa string, nameEn string, now time.Time) error {
	if strings.TrimSpace(nameJa) == "" {
		return ErrInvalidNameJa
	}

	c.NameJa = nameJa
	c.NameEn = nameEn
	c.touch(now)

	return c.validate()
}

func (c *ProductBlueprintCategory) ChangeParent(parentID *CategoryID, path []string, now time.Time) error {
	c.ParentID = parentID
	c.Path = normalizePath(path)
	c.touch(now)

	return c.validate()
}

func (c *ProductBlueprintCategory) ChangeDisplayOrder(displayOrder int, now time.Time) error {
	if displayOrder <= 0 {
		return ErrInvalidDisplayOrder
	}

	c.DisplayOrder = displayOrder
	c.touch(now)

	return nil
}

func (c *ProductBlueprintCategory) ChangeAttributes(attributes CategoryAttributes, now time.Time) error {
	c.Attributes = attributes
	c.touch(now)

	return nil
}

// ToSnapshot は productBlueprint へ denormalize 保存するための表示用スナップショットを返す。
func (c ProductBlueprintCategory) ToSnapshot() Snapshot {
	return Snapshot{
		ID:     c.ID,
		Code:   c.Code,
		NameJa: c.NameJa,
		NameEn: c.NameEn,
		Kind:   c.Kind,
		Path:   append([]string(nil), c.Path...),
	}
}

// ======================================
// Validation
// ======================================

func (c ProductBlueprintCategory) validate() error {
	if strings.TrimSpace(string(c.ID)) == "" {
		return ErrInvalidID
	}

	if strings.TrimSpace(string(c.Code)) == "" {
		return ErrInvalidCode
	}

	if strings.TrimSpace(c.NameJa) == "" {
		return ErrInvalidNameJa
	}

	if !IsValidCategoryKind(c.Kind) {
		return ErrInvalidKind
	}

	if len(c.Path) == 0 {
		return ErrInvalidPath
	}

	for _, p := range c.Path {
		if strings.TrimSpace(p) == "" {
			return ErrInvalidPath
		}
	}

	if c.DisplayOrder <= 0 {
		return ErrInvalidDisplayOrder
	}

	if c.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if c.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	return nil
}

// ======================================
// Helpers
// ======================================

func (c *ProductBlueprintCategory) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	c.UpdatedAt = now.UTC()
}

func normalizePath(path []string) []string {
	out := make([]string, 0, len(path))
	for _, p := range path {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
