// backend/internal/application/usecase/order_shipping_snapshot.go
package usecase

import (
	"context"
	"strings"

	orderdom "narratives/internal/domain/order"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

// =======================
// Shipping snapshot
// =======================

func (u *OrderUsecase) resolveShippingSnapshot(
	ctx context.Context,
	userID string,
	shippingAddressID string,
) (orderdom.ShippingSnapshot, error) {
	if u == nil ||
		u.shippingAddressRepo == nil {
		return orderdom.ShippingSnapshot{},
			orderdom.ErrInvalidShippingSnapshot
	}

	userID = strings.TrimSpace(userID)
	shippingAddressID = strings.TrimSpace(shippingAddressID)

	if userID == "" ||
		shippingAddressID == "" {
		return orderdom.ShippingSnapshot{},
			orderdom.ErrInvalidShippingSnapshot
	}

	address, err := u.shippingAddressRepo.GetByUser(
		ctx,
		shippingAddressID,
		userID,
	)
	if err != nil {
		return orderdom.ShippingSnapshot{}, err
	}

	if address == nil {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	if address.ID != shippingAddressID {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	if address.UserID != userID {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	return orderdom.ShippingSnapshot{
		ZipCode: address.ZipCode,
		State:   address.State,
		City:    address.City,
		Street:  address.Street,
		Street2: address.Street2,
		Country: address.Country,
	}, nil
}

// =======================
// Shipping quote snapshot
// =======================

func (u *OrderUsecase) resolveShippingQuoteSnapshot(
	ctx context.Context,
	userID string,
	shippingAddressID string,
	input []CreateOrderItemInput,
) (orderdom.ShippingQuoteSnapshot, error) {
	if u == nil ||
		u.shippingQuoteUC == nil {
		return orderdom.ShippingQuoteSnapshot{},
			orderdom.ErrInvalidShippingQuote
	}

	userID = strings.TrimSpace(userID)
	shippingAddressID = strings.TrimSpace(shippingAddressID)

	if userID == "" ||
		shippingAddressID == "" ||
		len(input) == 0 {
		return orderdom.ShippingQuoteSnapshot{},
			orderdom.ErrInvalidShippingQuote
	}

	quoteItems := make(
		[]orderdom.ShippingQuoteItemSnapshot,
		0,
		len(input),
	)

	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, item := range input {
		if item.Type != orderdom.OrderItemTypeList {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		if item.ListID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		quote, err := u.shippingQuoteUC.Quote(
			ctx,
			ShippingQuoteInput{
				UserID:                       userID,
				ListID:                       item.ListID,
				ModelID:                      item.ModelID,
				DestinationShippingAddressID: shippingAddressID,
			},
		)
		if err != nil {
			return orderdom.ShippingQuoteSnapshot{}, err
		}

		if quote.Amount < 0 ||
			quote.Amount > int64(maxInt) {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		unitAmount := int(quote.Amount)

		if unitAmount > 0 &&
			item.Qty > maxInt/unitAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		lineAmount := unitAmount * item.Qty

		if total > maxInt-lineAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		total += lineAmount

		quoteItems = append(
			quoteItems,
			orderdom.ShippingQuoteItemSnapshot{
				ListID:                       quote.ListID,
				InventoryID:                  quote.InventoryID,
				ModelID:                      quote.ModelID,
				OriginShippingAddressID:      quote.OriginShippingAddressID,
				DestinationShippingAddressID: quote.DestinationShippingAddressID,
				Carrier:                      string(quote.TransportationOption),
				TransportationID:             quote.TransportationID,
				Size:                         quote.Size,
				Qty:                          item.Qty,
				UnitAmount:                   unitAmount,
				Amount:                       lineAmount,
				Currency:                     quote.Currency,
			},
		)
	}

	return orderdom.ShippingQuoteSnapshot{
		Items:    quoteItems,
		Amount:   total,
		Currency: orderdom.ShippingQuoteCurrencyJPY,
	}, nil
}

func resolveOrderDestinationShippingAddressID(
	snapshot orderdom.ShippingQuoteSnapshot,
) (string, error) {
	if len(snapshot.Items) == 0 {
		return "",
			orderdom.ErrInvalidShippingQuote
	}

	destinationShippingAddressID := strings.TrimSpace(
		snapshot.Items[0].DestinationShippingAddressID,
	)

	if destinationShippingAddressID == "" {
		return "",
			orderdom.ErrInvalidShippingQuote
	}

	for _, item := range snapshot.Items {
		if strings.TrimSpace(
			item.DestinationShippingAddressID,
		) != destinationShippingAddressID {
			return "",
				orderdom.ErrInvalidShippingQuote
		}
	}

	return destinationShippingAddressID, nil
}

func createOrderItemInputsFromSnapshots(
	items []orderdom.OrderItemSnapshot,
) ([]CreateOrderItemInput, error) {
	if len(items) == 0 {
		return nil,
			orderdom.ErrInvalidItems
	}

	result := make(
		[]CreateOrderItemInput,
		0,
		len(items),
	)

	for _, item := range items {
		if item.IsCancelled {
			continue
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			result = append(
				result,
				CreateOrderItemInput{
					Type:         orderdom.OrderItemTypeList,
					ListID:       item.ListID,
					ModelID:      item.ModelID,
					Qty:          item.Qty,
					IsCancelled:  item.IsCancelled,
					IsDispatched: item.IsDispatched,
				},
			)

		case orderdom.OrderItemTypeResale:
			return nil,
				orderdom.ErrInvalidShippingQuote

		default:
			return nil,
				orderdom.ErrInvalidItemSnapshot
		}
	}

	return result, nil
}
