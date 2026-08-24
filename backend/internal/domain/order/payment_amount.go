// backend/internal/domain/order/payment_amount.go
package order

import (
	"errors"
	"strings"
)

// -------------------------------------------------------
// Errors
// -------------------------------------------------------

var (
	ErrInvalidPaymentAmount = errors.New(
		"order: invalid payment amount",
	)
)

// -------------------------------------------------------
// Payment Amount
// -------------------------------------------------------

// CalculatePaymentAmount は Order に保存された snapshot を基に、
// Stripe PaymentIntent へ渡す正規の支払総額を計算します。
//
// 支払総額:
//
//	商品小計
//	+ 配送料
//	+ 消費税
//
// 消費税:
//
//	軽減税率対象商品: 8%
//	標準税率対象商品: 10%
//	配送料: 10%
//
// 金額の source of truth は Order であり、
// frontend から渡された金額は使用しません。
func CalculatePaymentAmount(
	order Order,
) (int, error) {
	if len(order.Items) == 0 {
		return 0, ErrInvalidPaymentAmount
	}

	if len(
		order.ShippingQuoteSnapshot.Items,
	) == 0 {
		return 0, ErrInvalidPaymentAmount
	}

	if strings.TrimSpace(
		order.ShippingQuoteSnapshot.Currency,
	) != ShippingQuoteCurrencyJPY {
		return 0, ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)

	shippingAmount :=
		order.ShippingQuoteSnapshot.Amount

	if shippingAmount < 0 {
		return 0, ErrInvalidPaymentAmount
	}

	taxableAmount8 := 0

	// 配送料は標準税率10%の課税対象。
	taxableAmount10 :=
		shippingAmount

	for _, item := range order.Items {
		if item.Price < 0 ||
			item.Qty <= 0 {
			return 0, ErrInvalidPaymentAmount
		}

		if item.Price >
			maxInt/item.Qty {
			return 0, ErrInvalidPaymentAmount
		}

		lineAmount :=
			item.Price *
				item.Qty

		switch item.ConsumptionTaxRate {
		case ConsumptionTaxRateReduced:
			if taxableAmount8 >
				maxInt-lineAmount {
				return 0, ErrInvalidPaymentAmount
			}

			taxableAmount8 +=
				lineAmount

		case ConsumptionTaxRateStandard:
			if taxableAmount10 >
				maxInt-lineAmount {
				return 0, ErrInvalidPaymentAmount
			}

			taxableAmount10 +=
				lineAmount

		default:
			return 0, ErrInvalidPaymentAmount
		}
	}

	if taxableAmount8 >
		maxInt/ConsumptionTaxRateReduced {
		return 0, ErrInvalidPaymentAmount
	}

	if taxableAmount10 >
		maxInt/ConsumptionTaxRateStandard {
		return 0, ErrInvalidPaymentAmount
	}

	taxAmount8 :=
		taxableAmount8 *
			ConsumptionTaxRateReduced /
			100

	taxAmount10 :=
		taxableAmount10 *
			ConsumptionTaxRateStandard /
			100

	if taxAmount8 >
		maxInt-taxAmount10 {
		return 0, ErrInvalidPaymentAmount
	}

	taxAmount :=
		taxAmount8 +
			taxAmount10

	if taxableAmount8 >
		maxInt-taxableAmount10 {
		return 0, ErrInvalidPaymentAmount
	}

	subtotalWithShipping :=
		taxableAmount8 +
			taxableAmount10

	if subtotalWithShipping >
		maxInt-taxAmount {
		return 0, ErrInvalidPaymentAmount
	}

	total :=
		subtotalWithShipping +
			taxAmount

	if total <= 0 {
		return 0, ErrInvalidPaymentAmount
	}

	return total, nil
}
