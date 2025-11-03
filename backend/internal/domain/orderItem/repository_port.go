package orderitem

import (
	"context"
	"errors"
	"time"
)

// ドメインエラー
var (
	ErrOrderIDRequired   = errors.New("orderId is required")
	ErrItemIDRequired    = errors.New("itemId is required")
	ErrOrderItemNotFound = errors.New("order item not found")
)

// 作成用入力
type NewOrderItem struct {
	OrderID            string             `json:"orderId"`
	ModelNumber        string             `json:"modelNumber,omitempty"`
	ProductBlueprintID string             `json:"productBlueprintId,omitempty"`
	InventoryID        string             `json:"inventoryId,omitempty"`
	Size               string             `json:"size,omitempty"`
	Color              string             `json:"color,omitempty"`
	Quantity           int                `json:"quantity"`
	UnitPrice          float64            `json:"unitPrice"`
	Measurements       map[string]float64 `json:"measurements,omitempty"`
}

// 部分更新入力
type OrderItemUpdate struct {
	ModelNumber        *string            `json:"modelNumber,omitempty"`
	ProductBlueprintID *string            `json:"productBlueprintId,omitempty"`
	InventoryID        *string            `json:"inventoryId,omitempty"`
	Size               *string            `json:"size,omitempty"`
	Color              *string            `json:"color,omitempty"`
	Quantity           *int               `json:"quantity,omitempty"`
	UnitPrice          *float64           `json:"unitPrice,omitempty"`
	Measurements       map[string]float64 `json:"measurements,omitempty"` // nilで未更新
	UpdatedAt          *time.Time         `json:"updatedAt,omitempty"`
}

// 補助型（フロントのItemNameInfo/ItemSpec相当。必要に応じて利用）
type ItemNameInfo struct {
	ModelNumber        string `json:"modelNumber"`
	ProductName        string `json:"productName"`
	ProductBlueprintID string `json:"productBlueprintId"`
	InventoryID        string `json:"inventoryId"`
	Size               string `json:"size"`
	Color              string `json:"color"`
}

type ItemSpec struct {
	ModelNumber  string             `json:"modelNumber"`
	Size         string             `json:"size"`
	Color        string             `json:"color"`
	Measurements map[string]float64 `json:"measurements,omitempty"`
}

// Create input (from TS OrderItem without id)
type CreateOrderItemInput struct {
	ModelID     string `json:"modelId"`
	SaleID      string `json:"saleId"`
	InventoryID string `json:"inventoryId"`
	Quantity    int    `json:"quantity"`
}

// Partial update (nil means no change)
type UpdateOrderItemInput struct {
	ModelID     *string `json:"modelId,omitempty"`
	SaleID      *string `json:"saleId,omitempty"`
	InventoryID *string `json:"inventoryId,omitempty"`
	Quantity    *int    `json:"quantity,omitempty"`
}

// List filter (implementation decides how to combine)
type Filter struct {
	ID          string
	ModelID     string
	SaleID      string
	InventoryID string
	MinQuantity *int
	MaxQuantity *int
}

// Sort specification
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByID       SortColumn = "id"
	SortByQuantity SortColumn = "quantity"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Pagination
type Page struct {
	Number  int
	PerPage int
}

// Paged result
type PageResult struct {
	Items      []OrderItem
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// RepositoryPort defines domain-level access contracts for OrderItem.
type RepositoryPort interface {
	// Basic CRUD
	GetByID(ctx context.Context, id string) (*OrderItem, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Create(ctx context.Context, in CreateOrderItemInput) (*OrderItem, error)
	Update(ctx context.Context, id string, patch UpdateOrderItemInput) (*OrderItem, error)
	Delete(ctx context.Context, id string) error

	// Dev/Test helper
	Reset(ctx context.Context) error
}

// Common repository errors
var (
	ErrNotFound = errors.New("orderItem: not found")
	ErrConflict = errors.New("orderItem: conflict")
)
