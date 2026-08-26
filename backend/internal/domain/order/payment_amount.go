// backend/internal/domain/order/payment_amount.go
package order

import "errors"

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

type PaymentAmountSummary struct {
	SubtotalAmount int
	ShippingAmount int
	ConsumptionTax int
	TotalAmount    int
}

// CalculatePaymentAmountSummary は Order に保存された snapshot を基に、
// 支払金額の内訳を計算します。
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
// キャンセル済み商品:
//
//	IsCancelled == true の商品は支払対象から除外します。
//
// 全商品がキャンセル済みの場合は決済できません。
//
// 金額の source of truth は Order であり、
// frontend から渡された金額は使用しません。
func CalculatePaymentAmountSummary(
	order Order,
) (PaymentAmountSummary, error) {
	if len(order.Items) == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	if len(order.ShippingQuoteSnapshot.Items) == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	if order.ShippingQuoteSnapshot.Currency !=
		ShippingQuoteCurrencyJPY {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)

	shippingAmount :=
		order.ShippingQuoteSnapshot.Amount

	if shippingAmount < 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	subtotalAmount := 0
	taxableAmount8 := 0

	// 配送料は標準税率10%の課税対象。
	taxableAmount10 :=
		shippingAmount

	activeItemCount := 0

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		activeItemCount++

		if item.Price < 0 ||
			item.Qty <= 0 {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}

		if item.Price >
			maxInt/item.Qty {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}

		lineAmount :=
			item.Price *
				item.Qty

		if subtotalAmount >
			maxInt-lineAmount {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}

		subtotalAmount +=
			lineAmount

		switch item.ConsumptionTaxRate {
		case ConsumptionTaxRateReduced:
			if taxableAmount8 >
				maxInt-lineAmount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}

			taxableAmount8 +=
				lineAmount

		case ConsumptionTaxRateStandard:
			if taxableAmount10 >
				maxInt-lineAmount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}

			taxableAmount10 +=
				lineAmount

		default:
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
	}

	if activeItemCount == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	if taxableAmount8 >
		maxInt/ConsumptionTaxRateReduced {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	if taxableAmount10 >
		maxInt/ConsumptionTaxRateStandard {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
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
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	taxAmount :=
		taxAmount8 +
			taxAmount10

	if subtotalAmount >
		maxInt-shippingAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	subtotalWithShipping :=
		subtotalAmount +
			shippingAmount

	if subtotalWithShipping >
		maxInt-taxAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	totalAmount :=
		subtotalWithShipping +
			taxAmount

	if totalAmount <= 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	return PaymentAmountSummary{
		SubtotalAmount: subtotalAmount,
		ShippingAmount: shippingAmount,
		ConsumptionTax: taxAmount,
		TotalAmount:    totalAmount,
	}, nil
}

// CalculatePaymentAmount は Order に保存された snapshot を基に、
// Stripe PaymentIntent へ渡す正規の支払総額を返します。
func CalculatePaymentAmount(
	order Order,
) (int, error) {
	summary, err :=
		CalculatePaymentAmountSummary(order)
	if err != nil {
		return 0, err
	}

	return summary.TotalAmount, nil
}
