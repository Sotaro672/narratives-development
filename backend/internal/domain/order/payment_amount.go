// backend/internal/domain/order/payment_amount.go
package order

import "errors"

// -------------------------------------------------------
// Errors
// -------------------------------------------------------

var (
	ErrInvalidPaymentAmount = errors.New("order: invalid payment amount")
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
// buyer が支払う金額の内訳を計算します.
//
// List:
//
//	商品小計
//	+ 配送料
//	+ 消費税
//
// Resale:
//
//	商品価格のみ
//
// Resale商品の配送料は buyer の支払総額には加算しません.
// 発送時に確定したResale配送料は seller側の売上分配額から控除するために
// ShippingQuoteSnapshotへ保持します.
//
// 消費税:
//
//	軽減税率対象List商品: 8%
//	標準税率対象List商品: 10%
//	Resale商品: 非課税
//	List商品の配送料: 10%
//	Resale商品の配送料: 非課税かつbuyerへの追加請求対象外
//
// キャンセル済み商品:
//
//	IsCancelled == true の商品は支払対象から除外します.
//
// 全商品がキャンセル済みの場合は決済できません.
//
// 金額の source of truth は Order であり、frontend から渡された金額は使用しません.
func CalculatePaymentAmountSummary(order Order) (PaymentAmountSummary, error) {
	if len(order.Items) == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}
	if len(order.ShippingQuoteSnapshot.Items) == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}
	if order.ShippingQuoteSnapshot.Currency != ShippingQuoteCurrencyJPY {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)
	snapshotShippingAmount := order.ShippingQuoteSnapshot.Amount
	if snapshotShippingAmount < 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	chargedShippingAmount := 0
	taxableShippingAmount10 := 0
	calculatedSnapshotShippingAmount := 0

	for _, shippingItem := range order.ShippingQuoteSnapshot.Items {
		if shippingItem.Amount < 0 {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
		if calculatedSnapshotShippingAmount > maxInt-shippingItem.Amount {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
		calculatedSnapshotShippingAmount += shippingItem.Amount

		switch shippingItem.Type {
		case "", OrderItemTypeList:
			if chargedShippingAmount > maxInt-shippingItem.Amount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}
			chargedShippingAmount += shippingItem.Amount

			if taxableShippingAmount10 > maxInt-shippingItem.Amount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}
			taxableShippingAmount10 += shippingItem.Amount

		case OrderItemTypeResale:
			// Resale配送料はbuyerへ追加請求しない.
			// seller側の売上分配額から控除するためShippingQuoteSnapshotにのみ保持する.

		default:
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
	}

	if calculatedSnapshotShippingAmount != snapshotShippingAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	subtotalAmount := 0
	taxableAmount8 := 0
	taxableAmount10 := taxableShippingAmount10
	activeItemCount := 0

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		activeItemCount++

		if item.Price < 0 || item.Qty <= 0 {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
		if item.Price > maxInt/item.Qty {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}

		lineAmount := item.Price * item.Qty
		if subtotalAmount > maxInt-lineAmount {
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
		subtotalAmount += lineAmount

		switch item.Type {
		case OrderItemTypeResale:
			// Resale商品は非課税.
			// buyerは商品価格のみを支払い、Resale配送料はこの価格からseller側で控除する.
			continue

		case OrderItemTypeList:
		default:
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}

		switch item.ConsumptionTaxRate {
		case ConsumptionTaxRateReduced:
			if taxableAmount8 > maxInt-lineAmount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}
			taxableAmount8 += lineAmount

		case ConsumptionTaxRateStandard:
			if taxableAmount10 > maxInt-lineAmount {
				return PaymentAmountSummary{}, ErrInvalidPaymentAmount
			}
			taxableAmount10 += lineAmount

		default:
			return PaymentAmountSummary{}, ErrInvalidPaymentAmount
		}
	}

	if activeItemCount == 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}
	if taxableAmount8 > maxInt/ConsumptionTaxRateReduced {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}
	if taxableAmount10 > maxInt/ConsumptionTaxRateStandard {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	taxAmount8 := taxableAmount8 * ConsumptionTaxRateReduced / 100
	taxAmount10 := taxableAmount10 * ConsumptionTaxRateStandard / 100
	if taxAmount8 > maxInt-taxAmount10 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	taxAmount := taxAmount8 + taxAmount10

	if subtotalAmount > maxInt-chargedShippingAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}
	subtotalWithShipping := subtotalAmount + chargedShippingAmount

	if subtotalWithShipping > maxInt-taxAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	totalAmount := subtotalWithShipping + taxAmount
	if totalAmount <= 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	return PaymentAmountSummary{
		SubtotalAmount: subtotalAmount,
		ShippingAmount: chargedShippingAmount,
		ConsumptionTax: taxAmount,
		TotalAmount:    totalAmount,
	}, nil
}

// CalculatePaymentAmount は Order に保存された snapshot を基に、
// Stripe PaymentIntent へ渡す正規の支払総額を返します.
func CalculatePaymentAmount(order Order) (int, error) {
	summary, err := CalculatePaymentAmountSummary(order)
	if err != nil {
		return 0, err
	}

	return summary.TotalAmount, nil
}
