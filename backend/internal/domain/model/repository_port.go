// backend/internal/domain/model/repository_port.go
package model

import (
	"context"
	"errors"
	"time"
)

// Domain helper types (inputs/patches)

// Measurements は「各計測位置(string) → 計測値(int)」のマップ。
// entity.go の ModelVariation.Measurements (map[string]int) に対応。
type Measurements map[string]int

// NewModelVariation corresponds to TS: Omit<ModelVariation, 'id'>
// entity.go の ModelVariation に対応し、ID/CreatedAt/UpdatedAt などは除外。
// Color は値オブジェクト Color{Name, RGB} をそのまま利用します。
type NewModelVariation struct {
	Size         string       `json:"size"`
	Color        Color        `json:"color"` // { name, rgb }
	ModelNumber  string       `json:"modelNumber"`
	Measurements Measurements `json:"measurements,omitempty"` // 各計測位置: 計測値(int)
}

// ModelVariationUpdate corresponds to TS: Partial<Omit<ModelVariation, 'id'>>
type ModelVariationUpdate struct {
	Size         *string      `json:"size,omitempty"`
	Color        *Color       `json:"color,omitempty"` // 部分更新したい場合は構造に注意
	ModelNumber  *string      `json:"modelNumber,omitempty"`
	Measurements Measurements `json:"measurements,omitempty"` // nil なら更新スキップ
}

// ModelDataUpdate is free-form for product-level metadata updates
type ModelDataUpdate map[string]any

type ModelVariationWithQuantity struct {
	ModelVariation
	Quantity int `json:"quantity"`
}

// Listing contracts (filters/sort/page)

type VariationFilter struct {
	ProductID          string
	ProductBlueprintID string

	Sizes        []string
	Colors       []string // Color.Name を前提としたフィルタとして扱う想定
	ModelNumbers []string

	SearchQuery string // free text over modelNumber/size/color (implementation-defined)

	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	Deleted *bool // nil: all, true: deleted only, false: non-deleted only
}

type VariationSort struct {
	Column VariationSortColumn
	Order  SortOrder
}

type VariationSortColumn string

const (
	SortByModelNumber VariationSortColumn = "modelNumber"
	SortBySize        VariationSortColumn = "size"
	SortByColor       VariationSortColumn = "color"
	SortByCreatedAt   VariationSortColumn = "createdAt"
	SortByUpdatedAt   VariationSortColumn = "updatedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type Page struct {
	Number  int
	PerPage int
}

type VariationPageResult struct {
	Items      []ModelVariation
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// RepositoryPort abstracts model data access (contracts only)
type RepositoryPort interface {
	// Product-scoped model data
	GetModelData(ctx context.Context, productID string) (*ModelData, error)
	GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*ModelData, error)
	UpdateModelData(ctx context.Context, productID string, updates ModelDataUpdate) (*ModelData, error)

	// Variations (CRUD)
	ListVariations(ctx context.Context, filter VariationFilter, sort VariationSort, page Page) (VariationPageResult, error)
	GetModelVariations(ctx context.Context, productID string) ([]ModelVariation, error)
	GetModelVariationByID(ctx context.Context, variationID string) (*ModelVariation, error)
	CreateModelVariation(ctx context.Context, productID string, variation NewModelVariation) (*ModelVariation, error)
	UpdateModelVariation(ctx context.Context, variationID string, updates ModelVariationUpdate) (*ModelVariation, error)
	DeleteModelVariation(ctx context.Context, variationID string) (*ModelVariation, error)

	// Batch replace all variations for a product (idempotent by modelNumber/size/color as defined by implementation)
	ReplaceModelVariations(ctx context.Context, productID string, variations []NewModelVariation) ([]ModelVariation, error)

	// Convenience aggregations (resolver-style)
	GetSizeVariations(ctx context.Context, productID string) ([]SizeVariation, error)
	GetModelNumbers(ctx context.Context, productID string) ([]ModelNumber, error)
}

// Common repository errors
var (
	ErrNotFound = errors.New("model: not found")
	ErrConflict = errors.New("model: conflict")
	ErrInvalid  = errors.New("model: invalid")
)

// Compat alias if some code refers to Repository
type Repository = RepositoryPort
