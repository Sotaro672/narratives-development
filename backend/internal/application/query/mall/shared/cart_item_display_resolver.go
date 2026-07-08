// backend/internal/application/query/mall/shared/cart_item_display_resolver.go
package shared

import (
	malldto "narratives/internal/application/query/mall/dto"
	cartdom "narratives/internal/domain/cart"
)

// ResaleCartItemMeta is repository-derived display metadata for resale cart items.
type ResaleCartItemMeta struct {
	ID                 string
	Price              int
	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// CartModelDisplay is a small model display read model for CartItemDTO.
type CartModelDisplay struct {
	Kind        string
	ModelNumber string
	ModelLabel  string

	// apparel
	Size  string
	Color string

	// alcohol
	VolumeValue *int
	VolumeUnit  string
}

// ResaleCartItemDisplayInput is the shared mapper input for resale cart items.
//
// It intentionally accepts already-resolved metadata.
// Repository access stays in each query service.
type ResaleCartItemDisplayInput struct {
	Item cartdom.CartItem

	Meta *ResaleCartItemMeta

	ImageURL string

	BrandName string
	ModelID   string
	Model     CartModelDisplay

	ProductBlueprintID string
	ProductName        string
}

// InferCartItemType infers the cart item type from explicit Type first,
// then from legacy/partial fields.
func InferCartItemType(it cartdom.CartItem) cartdom.CartItemType {
	switch it.Type {
	case cartdom.CartItemTypeList, cartdom.CartItemTypeResale:
		return it.Type
	}

	if it.ResaleID != "" || it.ProductID != "" {
		return cartdom.CartItemTypeResale
	}

	if it.InventoryID != "" || it.ListID != "" || it.ModelID != "" {
		return cartdom.CartItemTypeList
	}

	return ""
}

// ResaleCartItemToDTO maps a resale cart item to CartItemDTO.
//
// Policy:
// - invalid resale item returns false
// - qty is always normalized to 1
// - ImageURL is also copied to ListImage for existing frontend compatibility
// - ProductName is also copied to Title for existing frontend compatibility
func ResaleCartItemToDTO(
	in ResaleCartItemDisplayInput,
) (malldto.CartItemDTO, bool) {
	it := in.Item

	if it.ResaleID == "" || it.ProductID == "" {
		return malldto.CartItemDTO{}, false
	}

	item := malldto.CartItemDTO{
		Type:      string(cartdom.CartItemTypeResale),
		ResaleID:  it.ResaleID,
		ProductID: it.ProductID,
		Qty:       1,
	}

	if in.Meta != nil {
		if in.Meta.ProductID != "" {
			item.ProductID = in.Meta.ProductID
		}

		if in.Meta.ProductBlueprintID != "" {
			item.ProductBlueprintID = in.Meta.ProductBlueprintID
		}

		if in.Meta.TokenBlueprintID != "" {
			item.TokenBlueprintID = in.Meta.TokenBlueprintID
		}

		if in.Meta.BrandID != "" {
			item.BrandID = in.Meta.BrandID
		}

		price := in.Meta.Price
		item.Price = &price
	}

	if in.ProductBlueprintID != "" && item.ProductBlueprintID == "" {
		item.ProductBlueprintID = in.ProductBlueprintID
	}

	if in.ImageURL != "" {
		item.ImageURL = in.ImageURL
		item.ListImage = in.ImageURL
	}

	if in.BrandName != "" {
		item.BrandName = in.BrandName
	}

	if in.ModelID != "" {
		item.ModelID = in.ModelID
	}

	ApplyCartModelDisplay(&item, in.Model)

	if in.ProductName != "" {
		item.ProductName = in.ProductName
		item.Title = in.ProductName
	}

	return item, true
}

// ApplyCartModelDisplay applies model display fields to CartItemDTO.
func ApplyCartModelDisplay(
	item *malldto.CartItemDTO,
	model CartModelDisplay,
) {
	if item == nil {
		return
	}

	if model.Kind != "" {
		item.ModelKind = model.Kind
	}
	if model.ModelNumber != "" {
		item.ModelNumber = model.ModelNumber
	}
	if model.ModelLabel != "" {
		item.ModelLabel = model.ModelLabel
	}

	if model.Size != "" {
		item.Size = model.Size
	}
	if model.Color != "" {
		item.Color = model.Color
	}

	if model.VolumeValue != nil {
		item.VolumeValue = model.VolumeValue
	}
	if model.VolumeUnit != "" {
		item.VolumeUnit = model.VolumeUnit
	}
}
