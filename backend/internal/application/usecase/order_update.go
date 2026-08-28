// backend/internal/application/usecase/order_update.go
package usecase

import (
	"context"
	"strings"

	orderdom "narratives/internal/domain/order"
)

type UpdateOrderInput struct {
	ID string

	UserID   *string
	AvatarID *string
	CartID   *string

	ShippingAddressID *string
	PaymentMethodID   *string

	ReplaceItems *[]CreateOrderItemInput
}

func (u *OrderUsecase) Update(
	ctx context.Context,
	in UpdateOrderInput,
) (orderdom.Order, error) {
	order, err := u.repo.GetByID(ctx, in.ID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.UserID != nil {
		order.UserID = *in.UserID
	}

	if in.AvatarID != nil {
		order.AvatarID = *in.AvatarID
	}

	if in.CartID != nil {
		order.CartID = *in.CartID
	}

	shippingAddressID, err := resolveOrderDestinationShippingAddressID(
		order.ShippingQuoteSnapshot,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.ShippingAddressID != nil {
		shippingAddressID = strings.TrimSpace(*in.ShippingAddressID)

		if shippingAddressID == "" {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingSnapshot
		}
	}

	shouldRefreshShipping :=
		in.ShippingAddressID != nil ||
			in.UserID != nil

	if shouldRefreshShipping {
		shipping, err := u.resolveShippingSnapshot(
			ctx,
			order.UserID,
			shippingAddressID,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdateShippingSnapshot(shipping); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.PaymentMethodID != nil {
		paymentMethod, err := u.resolvePaymentMethodSnapshot(
			ctx,
			order.UserID,
			*in.PaymentMethodID,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdatePaymentMethodSnapshot(
			paymentMethod,
		); err != nil {
			return orderdom.Order{}, err
		}
	}

	var shippingQuoteItems []CreateOrderItemInput

	if in.ReplaceItems != nil {
		items, err := u.resolveOrderItems(
			ctx,
			*in.ReplaceItems,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.ReplaceItems(items); err != nil {
			return orderdom.Order{}, err
		}

		shippingQuoteItems = append(
			[]CreateOrderItemInput(nil),
			(*in.ReplaceItems)...,
		)
	}

	shouldRefreshShippingQuote :=
		in.ReplaceItems != nil ||
			in.ShippingAddressID != nil ||
			in.UserID != nil

	if shouldRefreshShippingQuote {
		if shippingQuoteItems == nil {
			shippingQuoteItems, err =
				createOrderItemInputsFromSnapshots(
					order.Items,
				)
			if err != nil {
				return orderdom.Order{}, err
			}
		}

		shippingQuote, err := u.resolveShippingQuoteSnapshot(
			ctx,
			order.UserID,
			shippingAddressID,
			shippingQuoteItems,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdateShippingQuoteSnapshot(
			shippingQuote,
		); err != nil {
			return orderdom.Order{}, err
		}
	}

	checked, err := orderdom.New(
		order.ID,
		order.UserID,
		order.AvatarID,
		order.CartID,
		order.ShippingSnapshot,
		order.ShippingQuoteSnapshot,
		order.PaymentMethodSnapshot,
		order.Items,
		order.CreatedAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	checked.Paid = order.Paid

	// Repository.Update must persist the Order and replace its canonical
	// orderTransferItems projection in the same Firestore transaction.
	return u.repo.Update(ctx, checked, nil)
}
