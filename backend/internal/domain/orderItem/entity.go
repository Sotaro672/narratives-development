// backend\internal\domain\orderItem\entity.go
package orderitem

import (
	"errors"
	"strings"
)

type OrderItem struct {
	ID          string
	ModelID     string
	SaleID      string
	InventoryID string
	Quantity    int
}

// Errors
var (
	ErrInvalidID          = errors.New("orderItem: invalid id")
	ErrInvalidModelID     = errors.New("orderItem: invalid modelId")
	ErrInvalidSaleID      = errors.New("orderItem: invalid saleId")
	ErrInvalidInventoryID = errors.New("orderItem: invalid inventoryId")
	ErrInvalidQuantity    = errors.New("orderItem: invalid quantity")
)

// Policy
var (
	MinQuantity = 1 // inclusive
	MaxQuantity = 0 // 0 disables upper bound
)

// Constructors

func New(id, modelID, saleID, inventoryID string, quantity int) (OrderItem, error) {
	oi := OrderItem{
		ID:          strings.TrimSpace(id),
		ModelID:     strings.TrimSpace(modelID),
		SaleID:      strings.TrimSpace(saleID),
		InventoryID: strings.TrimSpace(inventoryID),
		Quantity:    quantity,
	}
	if err := oi.validate(); err != nil {
		return OrderItem{}, err
	}
	return oi, nil
}

// NewFromCreate matches a typical create payload with server-assigned id.
func NewFromCreate(id string, create struct {
	ModelID     string
	SaleID      string
	InventoryID string
	Quantity    int
}) (OrderItem, error) {
	return New(id, create.ModelID, create.SaleID, create.InventoryID, create.Quantity)
}

// Mutators

func (o *OrderItem) SetQuantity(q int) error {
	if q < MinQuantity || (MaxQuantity > 0 && q > MaxQuantity) {
		return ErrInvalidQuantity
	}
	o.Quantity = q
	return nil
}

func (o *OrderItem) IncrementQuantity(delta int) error {
	n := o.Quantity + delta
	if n < MinQuantity || (MaxQuantity > 0 && n > MaxQuantity) {
		return ErrInvalidQuantity
	}
	o.Quantity = n
	return nil
}

func (o *OrderItem) ReassignInventory(inventoryID string) error {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ErrInvalidInventoryID
	}
	o.InventoryID = inventoryID
	return nil
}

func (o *OrderItem) ReassignSale(saleID string) error {
	saleID = strings.TrimSpace(saleID)
	if saleID == "" {
		return ErrInvalidSaleID
	}
	o.SaleID = saleID
	return nil
}

func (o *OrderItem) UpdateModelID(modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ErrInvalidModelID
	}
	o.ModelID = modelID
	return nil
}

// Validation

func (o OrderItem) validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.ModelID == "" {
		return ErrInvalidModelID
	}
	if o.SaleID == "" {
		return ErrInvalidSaleID
	}
	if o.InventoryID == "" {
		return ErrInvalidInventoryID
	}
	if o.Quantity < MinQuantity || (MaxQuantity > 0 && o.Quantity > MaxQuantity) {
		return ErrInvalidQuantity
	}
	return nil
}
