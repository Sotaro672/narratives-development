// backend/internal/domain/refund/amount_calculator.go
package refund

import (
	"errors"

	orderdom "narratives/internal/domain/order"
)

// -------------------------------------------------------
// Errors
// -------------------------------------------------------

var (
	ErrInvalidOrderItemRefund = errors.New("order: invalid order item refund")
)

// -------------------------------------------------------
// Order Item Refund Amount
// -------------------------------------------------------

type OrderItemRefundAmountSummary struct {
	MerchandiseAmount    int
	MerchandiseTaxAmount int
	RefundAmount         int
}

// CalculateOrderItemRefundAmount は Order item 単位の返品に対して、
// Stripe Refund へ渡す正規の部分返金額を返します。
//
// 返金対象:
//
//	商品本体価格
//	+ 対象商品へ配賦された消費税
//
// 返金対象外:
//
//	配送料
//	配送料に対する消費税
//
// Order 作成時の消費税は税率単位で集約して計算されるため、単純に各商品の
// lineAmount * taxRate / 100 を返すと、Order 全体の税額や Settlement の税配賦と
// 1円ずれる可能性があります。
//
// そのため税額は以下の順序で配賦します。
//
//  1. List商品だけを seller Account ごとに merchandise 8%, merchandise 10%,
//     shipping 10% として集約する。Resale商品は非課税・送料0として保持する。
//  2. 税率ごとの正規税額を最大剰余法で Account/component へ配賦する。
//  3. Account の merchandise component に配賦された税額を、同一 Account・
//     同一税率のList商品へ最大剰余法で再配賦する。
//  4. Resale商品には MerchandiseTaxAmount=0 を割り当てる。
//
// 第一段階の tie-break は Settlement calculator と同じく:
//
//  1. remainder 降順
//  2. AccountID 昇順
//  3. component kind 昇順
//
// 第二段階の tie-break は:
//
//  1. remainder 降順
//  2. Order item index 昇順
//
// とします。
//
// これにより、全 item の MerchandiseTaxAmount と配送料税額を合計すると、
// order.CalculatePaymentAmountSummary の ConsumptionTax と一致します。
func CalculateOrderItemRefundAmount(order orderdom.Order, orderItemIndex int) (OrderItemRefundAmountSummary, error) {
	if orderItemIndex < 0 || orderItemIndex >= len(order.Items) {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	paymentSummary, err := orderdom.CalculatePaymentAmountSummary(order)
	if err != nil {
		return OrderItemRefundAmountSummary{}, err
	}

	targetItem := order.Items[orderItemIndex]
	if targetItem.IsCancelled || targetItem.Price < 0 || targetItem.Qty <= 0 {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	merchandiseAmount, err := safeMultiplyPaymentAmount(targetItem.Price, targetItem.Qty)
	if err != nil {
		return OrderItemRefundAmountSummary{}, err
	}

	itemTaxes, shippingTax, err := calculateOrderItemTaxAllocations(order)
	if err != nil {
		return OrderItemRefundAmountSummary{}, err
	}

	merchandiseTaxAmount, ok := itemTaxes[orderItemIndex]
	if !ok || merchandiseTaxAmount < 0 {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	totalAllocatedTax := shippingTax
	for _, taxAmount := range itemTaxes {
		totalAllocatedTax, err = safeAddPaymentAmount(totalAllocatedTax, taxAmount)
		if err != nil {
			return OrderItemRefundAmountSummary{}, err
		}
	}

	if totalAllocatedTax != paymentSummary.ConsumptionTax {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	refundAmount, err := safeAddPaymentAmount(merchandiseAmount, merchandiseTaxAmount)
	if err != nil {
		return OrderItemRefundAmountSummary{}, err
	}
	if refundAmount <= 0 {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	return OrderItemRefundAmountSummary{
		MerchandiseAmount:    merchandiseAmount,
		MerchandiseTaxAmount: merchandiseTaxAmount,
		RefundAmount:         refundAmount,
	}, nil
}

// -------------------------------------------------------
// Opened Return Refund Amount
// -------------------------------------------------------

// OpenedReturnRefundAmountSummary represents the authoritative financial
// amounts for one opened-return policy.
//
// StripeRefundAmount is the amount that can be refunded against the original
// purchaser Charge.
//
// TotalSellerBurdenAmount additionally includes return-shipping cost that is not
// part of the original Charge.
//
// Current return-shipping policy:
//   - no separate return-shipping quote snapshot exists in Order yet
//   - return shipping is therefore modeled using the same tax-exclusive amount
//     as the target item's original outbound shipping
//   - return shipping consumption tax is calculated at the standard 10% rate
//
// Once an authoritative return-shipping quote is persisted, the application
// layer should replace this modeled amount with that snapshot.
type OpenedReturnRefundAmountSummary struct {
	Policy OpenedReturnRefundPolicy

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	StripeRefundAmount      int
	TotalSellerBurdenAmount int
}

// CalculateOpenedReturnRefundAmount calculates one opened-return refund option
// from the persisted Order snapshot.
//
// The frontend may select only policy. Monetary amounts are never accepted from
// the frontend.
//
// Policies:
//
// half_merchandise:
//   - 50% of merchandise + proportional merchandise tax
//   - shipping is excluded
//
// merchandise_only:
//   - full merchandise + merchandise tax
//   - shipping is excluded
//
// merchandise_round_trip_shipping:
//   - Stripe Refund:
//     merchandise + merchandise tax + outbound shipping + outbound shipping tax
//   - Additional seller burden:
//     modeled return shipping + return shipping tax
//
// Half-refund rounding:
// JPY cannot represent fractional yen, so the total purchaser refund is rounded
// up to the nearest yen. Merchandise is also halved with the same rule and the
// remaining amount is treated as merchandise tax.
func CalculateOpenedReturnRefundAmount(
	order orderdom.Order,
	orderItemIndex int,
	policy OpenedReturnRefundPolicy,
) (OpenedReturnRefundAmountSummary, error) {
	if err := ValidateOpenedReturnRefundPolicy(policy); err != nil {
		return OpenedReturnRefundAmountSummary{}, err
	}

	fullMerchandise, err := CalculateOrderItemRefundAmount(order, orderItemIndex)
	if err != nil {
		return OpenedReturnRefundAmountSummary{}, err
	}

	switch policy {
	case OpenedReturnRefundHalfMerchandise:
		stripeRefundAmount := halfPaymentAmountRoundedUp(fullMerchandise.RefundAmount)
		merchandiseAmount := halfPaymentAmountRoundedUp(fullMerchandise.MerchandiseAmount)
		merchandiseTaxAmount := stripeRefundAmount - merchandiseAmount

		if stripeRefundAmount <= 0 ||
			merchandiseAmount < 0 ||
			merchandiseTaxAmount < 0 ||
			merchandiseAmount > fullMerchandise.MerchandiseAmount ||
			merchandiseTaxAmount > fullMerchandise.MerchandiseTaxAmount {
			return OpenedReturnRefundAmountSummary{}, ErrInvalidOrderItemRefund
		}

		return OpenedReturnRefundAmountSummary{
			Policy:                  policy,
			MerchandiseAmount:       merchandiseAmount,
			MerchandiseTaxAmount:    merchandiseTaxAmount,
			StripeRefundAmount:      stripeRefundAmount,
			TotalSellerBurdenAmount: stripeRefundAmount,
		}, nil

	case OpenedReturnRefundMerchandiseOnly:
		return OpenedReturnRefundAmountSummary{
			Policy:                  policy,
			MerchandiseAmount:       fullMerchandise.MerchandiseAmount,
			MerchandiseTaxAmount:    fullMerchandise.MerchandiseTaxAmount,
			StripeRefundAmount:      fullMerchandise.RefundAmount,
			TotalSellerBurdenAmount: fullMerchandise.RefundAmount,
		}, nil

	case OpenedReturnRefundMerchandiseRoundTripShipping:
		shippingAmounts, err := calculateOrderItemShippingAmounts(order)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		_, totalShippingTax, err := calculateOrderItemTaxAllocations(order)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		shippingTaxes, err := allocateOpenedReturnShippingTaxToItems(shippingAmounts, totalShippingTax)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		outboundShippingAmount, ok := shippingAmounts[orderItemIndex]
		if !ok || outboundShippingAmount < 0 {
			return OpenedReturnRefundAmountSummary{}, ErrInvalidOrderItemRefund
		}

		outboundShippingTaxAmount, ok := shippingTaxes[orderItemIndex]
		if !ok || outboundShippingTaxAmount < 0 {
			return OpenedReturnRefundAmountSummary{}, ErrInvalidOrderItemRefund
		}

		stripeRefundAmount, err := safeAddPaymentAmount(fullMerchandise.RefundAmount, outboundShippingAmount)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		stripeRefundAmount, err = safeAddPaymentAmount(stripeRefundAmount, outboundShippingTaxAmount)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		// Until Order stores a dedicated return-shipping quote, use the
		// target item's original outbound shipping amount as the modeled
		// reverse-route shipping amount.
		returnShippingAmount := outboundShippingAmount

		returnShippingTaxProduct, err := safeMultiplyPaymentAmount(
			returnShippingAmount,
			orderdom.ConsumptionTaxRateStandard,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		returnShippingTaxAmount := returnShippingTaxProduct / 100

		totalSellerBurdenAmount, err := safeAddPaymentAmount(stripeRefundAmount, returnShippingAmount)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		totalSellerBurdenAmount, err = safeAddPaymentAmount(totalSellerBurdenAmount, returnShippingTaxAmount)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		return OpenedReturnRefundAmountSummary{
			Policy:                    policy,
			MerchandiseAmount:         fullMerchandise.MerchandiseAmount,
			MerchandiseTaxAmount:      fullMerchandise.MerchandiseTaxAmount,
			OutboundShippingAmount:    outboundShippingAmount,
			OutboundShippingTaxAmount: outboundShippingTaxAmount,
			ReturnShippingAmount:      returnShippingAmount,
			ReturnShippingTaxAmount:   returnShippingTaxAmount,
			StripeRefundAmount:        stripeRefundAmount,
			TotalSellerBurdenAmount:   totalSellerBurdenAmount,
		}, nil

	default:
		return OpenedReturnRefundAmountSummary{}, ErrInvalidOpenedReturnRefundPolicy
	}
}
