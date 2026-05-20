// backend/internal/domain/productBlueprintCategory/entity.go
package productBlueprintCategory

import (
	"errors"
	"fmt"
	"time"

	"narratives/internal/domain/common"
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
type CategoryKind = common.ProductCategoryKind

const (
	CategoryKindApparel    = common.ProductCategoryKindApparel
	CategoryKindFood       = common.ProductCategoryKindFood
	CategoryKindAlcohol    = common.ProductCategoryKindAlcohol
	CategoryKindCosmetics  = common.ProductCategoryKindCosmetics
	CategoryKindGoods      = common.ProductCategoryKindGoods
	CategoryKindHealthcare = common.ProductCategoryKindHealthcare
	CategoryKindOther      = common.ProductCategoryKindOther
)

func IsValidCategoryKind(v CategoryKind) bool {
	return common.IsValidProductCategoryKind(v)
}

// ======================================
// Entity
// ======================================

// ProductBlueprintCategory は productBlueprint が参照する商品カテゴリマスタ。
// productBlueprint 側には categoryId と denormalized 表示用フィールドを保存し、
// カテゴリ定義の正はこのドメインで管理する。
//
// NOTE:
// カテゴリごとの入力項目定義は、この entity.go には持たせない。
// 入力項目定義は input_schema.go 側で CategoryInputSchema として管理する。
type ProductBlueprintCategory struct {
	ID CategoryID

	// 例:
	// - apparel
	// - apparel.tops
	// - apparel.bottoms
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
	// これは入力欄一覧ではなく、法務・運用・表示上の要件フラグとして扱う。
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
	category := ProductBlueprintCategory{
		ID:           id,
		Code:         code,
		NameJa:       nameJa,
		NameEn:       nameEn,
		ParentID:     parentID,
		Path:         path,
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
		Path:         path,
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
	c.NameJa = nameJa
	c.NameEn = nameEn
	c.UpdatedAt = now.UTC()

	return c.validate()
}

func (c *ProductBlueprintCategory) ChangeParent(parentID *CategoryID, path []string, now time.Time) error {
	c.ParentID = parentID
	c.Path = path
	c.UpdatedAt = now.UTC()

	return c.validate()
}

func (c *ProductBlueprintCategory) ChangeDisplayOrder(displayOrder int, now time.Time) error {
	c.DisplayOrder = displayOrder
	c.UpdatedAt = now.UTC()

	return c.validate()
}

func (c *ProductBlueprintCategory) ChangeAttributes(attributes CategoryAttributes, now time.Time) error {
	c.Attributes = attributes
	c.UpdatedAt = now.UTC()

	return c.validate()
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
	if string(c.ID) == "" {
		return ErrInvalidID
	}

	if string(c.Code) == "" {
		return ErrInvalidCode
	}

	if c.NameJa == "" {
		return ErrInvalidNameJa
	}

	if !IsValidCategoryKind(c.Kind) {
		return ErrInvalidKind
	}

	if len(c.Path) == 0 {
		return ErrInvalidPath
	}

	for _, p := range c.Path {
		if p == "" {
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
