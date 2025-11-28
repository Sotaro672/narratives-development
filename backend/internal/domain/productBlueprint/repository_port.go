// backend/internal/domain/productBlueprint/repository_port.go
package productBlueprint

import (
	"context"
	"time"
)

// ========================================
// Create/Update inputs (contract only)
// ========================================

type CreateInput struct {
	ProductName      string       `json:"productName"`
	BrandID          string       `json:"brandId"`
	ItemType         ItemType     `json:"itemType"`
	Fit              string       `json:"fit"`
	Material         string       `json:"material"`
	Weight           float64      `json:"weight"`
	QualityAssurance []string     `json:"qualityAssurance"`
	ProductIdTag     ProductIDTag `json:"productIdTag"`
	AssigneeID       string       `json:"assigneeId"`
	CompanyID        string       `json:"companyId"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"` // repo may set if nil
}

type Patch struct {
	ProductName      *string       `json:"productName,omitempty"`
	BrandID          *string       `json:"brandId,omitempty"`
	ItemType         *ItemType     `json:"itemType,omitempty"`
	Fit              *string       `json:"fit,omitempty"`
	Material         *string       `json:"material,omitempty"`
	Weight           *float64      `json:"weight,omitempty"`
	QualityAssurance *[]string     `json:"qualityAssurance,omitempty"`
	ProductIdTag     *ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID       *string       `json:"assigneeId,omitempty"`
}

// ========================================
// Query contracts (filters/sort/paging)
// ========================================

type Filter struct {
	SearchTerm string

	BrandIDs    []string
	AssigneeIDs []string
	ItemTypes   []ItemType
	TagTypes    []ProductIDTagType

	// 削除状態フィルタ:
	// - false: 通常の（未削除）レコードを対象（実装側で DeletedAt == nil を想定）
	// - true : 論理削除済み（DeletedAt != nil）のみを対象
	OnlyDeleted bool

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt   SortColumn = "createdAt"
	SortByUpdatedAt   SortColumn = "updatedAt"
	SortByProductName SortColumn = "productName"
	SortByBrandID     SortColumn = "brandId"
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

type PageResult struct {
	Items      []ProductBlueprint
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ========================================
// Repository Port (interface contracts only)
// ========================================

type Repository interface {
	// Read
	GetByID(ctx context.Context, id string) (ProductBlueprint, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 存在確認（adapter の Exists を port に昇格）
	Exists(ctx context.Context, id string) (bool, error)

	// Write
	Create(ctx context.Context, in CreateInput) (ProductBlueprint, error)
	Update(ctx context.Context, id string, patch Patch) (ProductBlueprint, error)
	Delete(ctx context.Context, id string) error

	// Dev/Test helper
	Reset(ctx context.Context) error
}
