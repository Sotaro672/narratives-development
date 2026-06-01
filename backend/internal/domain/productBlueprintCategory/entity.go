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

	ErrRepositoryInvalidInput = errors.New("productBlueprintCategory: repository invalid input")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

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

// ======================================
// Entity
// ======================================

type ProductBlueprintCategory struct {
	ID CategoryID

	Code CategoryCode

	NameJa string
	NameEn string

	ParentID *CategoryID

	Path []string

	Kind CategoryKind

	DisplayOrder int

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

type Snapshot struct {
	ID     CategoryID
	Code   CategoryCode
	NameJa string
	NameEn string
	Kind   CategoryKind
	Path   []string
}

// ======================================
// Constructors
// ======================================

// Reconstruct は永続化層から復元するときに使う。
// テーブル定義中心の entity として扱うため、ここでは validate は行わない。
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
		Path:         append([]string(nil), path...),
		Kind:         kind,
		DisplayOrder: displayOrder,
		Attributes:   attributes,
		CreatedAt:    createdAt.UTC(),
		UpdatedAt:    updatedAt.UTC(),
	}

	return category, nil
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

func IsValidCategoryKind(v CategoryKind) bool {
	return common.IsValidProductCategoryKind(v)
}
