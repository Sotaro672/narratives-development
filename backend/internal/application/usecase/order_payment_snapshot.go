// backend/internal/application/usecase/order_payment_snapshot.go
package usecase

import (
	"context"

	orderdom "narratives/internal/domain/order"
)

// =======================
// Payment method snapshot
// =======================

func (u *OrderUsecase) resolvePaymentMethodSnapshot(
	ctx context.Context,
	userID string,
	paymentMethodID string,
) (orderdom.PaymentMethodSnapshot, error) {
	paymentMethod, err := u.paymentMethodRepo.GetByID(
		ctx,
		paymentMethodID,
	)
	if err != nil {
		return orderdom.PaymentMethodSnapshot{}, err
	}

	if paymentMethod == nil ||
		paymentMethod.UserID != userID {
		return orderdom.PaymentMethodSnapshot{},
			orderdom.ErrInvalidPaymentMethod
	}

	return orderdom.PaymentMethodSnapshot{
		PaymentMethodID:       paymentMethod.ID,
		CustomerID:            paymentMethod.StripeCustomerID,
		StripePaymentMethodID: paymentMethod.StripePaymentMethodID,
		Brand:                 paymentMethod.Brand,
		Last4:                 paymentMethod.Last4,
		ExpMonth:              paymentMethod.ExpMonth,
		ExpYear:               paymentMethod.ExpYear,
		CardholderName:        paymentMethod.CardholderName,
		IsDefault:             paymentMethod.IsDefault,
	}, nil
}
