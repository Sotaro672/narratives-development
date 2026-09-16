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

// CalculateOrderItemRefundAmount は Order item 単位の商品代金全額返金に対して、
// 商品本体価格と対象商品へ配賦された消費税を返します。
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
func CalculateOrderItemRefundAmount(
	order orderdom.Order,
	orderItemIndex int,
) (OrderItemRefundAmountSummary, error) {
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

	merchandiseAmount, err := safeMultiplyPaymentAmount(
		targetItem.Price,
		targetItem.Qty,
	)
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
		totalAllocatedTax, err = safeAddPaymentAmount(
			totalAllocatedTax,
			taxAmount,
		)
		if err != nil {
			return OrderItemRefundAmountSummary{}, err
		}
	}

	if totalAllocatedTax != paymentSummary.ConsumptionTax {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	refundAmount, err := safeAddPaymentAmount(
		merchandiseAmount,
		merchandiseTaxAmount,
	)
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
// Return Refund Amount
// -------------------------------------------------------

// ReturnRefundAmountSummary represents the authoritative financial amounts for
// one return refund selection.
//
// MerchandiseRefundAmount is the tax-inclusive merchandise refund amount
// explicitly selected by the seller.
//
// MerchandiseAmount and MerchandiseTaxAmount are derived automatically from the
// authoritative Order snapshot so that:
//
//	MerchandiseAmount + MerchandiseTaxAmount
//	= MerchandiseRefundAmount
//
// StripeRefundAmount is the amount refunded against the purchaser's original
// Stripe Charge. It contains:
//
//	MerchandiseRefundAmount
//	+ outbound shipping and tax when RefundOutboundShipping=true
//
// TotalSellerBurdenAmount additionally contains return-shipping cost when
// CoverReturnShipping=true.
//
// Current return-shipping behavior:
//   - Order does not yet persist a dedicated return-shipping quote
//   - the target item's original outbound shipping amount is therefore used as
//     the modeled return-shipping amount
//   - return-shipping consumption tax is calculated at the standard 10% rate
//
// Once an authoritative return-shipping quote is persisted, the modeled
// return-shipping calculation should be replaced with that snapshot.
type ReturnRefundAmountSummary struct {
	MerchandiseRefundAmount int

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	StripeRefundAmount      int
	TotalSellerBurdenAmount int
}

// CalculateReturnRefundAmount calculates the authoritative refund amounts from
// the persisted Order snapshot and the seller's return refund selection.
//
// The frontend may specify only:
//
//   - tax-inclusive merchandise refund amount
//   - whether outbound shipping is refunded
//   - whether return shipping is covered
//
// The frontend must never provide:
//
//   - merchandise tax amount
//   - outbound shipping amount
//   - outbound shipping tax
//   - return shipping amount
//   - return shipping tax
//   - Stripe refund amount
//   - total seller burden
//
// MerchandiseRefundAmount must satisfy:
//
//	1 <= MerchandiseRefundAmount <= original merchandise amount including tax
//
// Consumption tax is automatically derived proportionally from the original
// merchandise/tax allocation. Integer rounding is deterministic and the tax
// portion is always the remainder so that the tax-inclusive requested refund
// amount is preserved exactly.
func CalculateReturnRefundAmount(
	order orderdom.Order,
	orderItemIndex int,
	selection ReturnRefundSelection,
) (ReturnRefundAmountSummary, error) {
	if err := ValidateReturnRefundSelection(selection); err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	fullMerchandise, err := CalculateOrderItemRefundAmount(
		order,
		orderItemIndex,
	)
	if err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	if selection.MerchandiseRefundAmount > fullMerchandise.RefundAmount {
		return ReturnRefundAmountSummary{}, ErrInvalidReturnRefundAmount
	}

	merchandiseAmount, merchandiseTaxAmount, err :=
		calculateProportionalMerchandiseRefund(
			fullMerchandise,
			selection.MerchandiseRefundAmount,
		)
	if err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	result := ReturnRefundAmountSummary{
		MerchandiseRefundAmount: selection.MerchandiseRefundAmount,
		MerchandiseAmount:       merchandiseAmount,
		MerchandiseTaxAmount:    merchandiseTaxAmount,
		StripeRefundAmount:      selection.MerchandiseRefundAmount,
		TotalSellerBurdenAmount: selection.MerchandiseRefundAmount,
	}

	if !selection.RefundOutboundShipping &&
		!selection.CoverReturnShipping {
		return result, nil
	}

	shippingAmounts, err := calculateOrderItemShippingAmounts(order)
	if err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	_, totalShippingTax, err := calculateOrderItemTaxAllocations(order)
	if err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	shippingTaxes, err := allocateOpenedReturnShippingTaxToItems(
		shippingAmounts,
		totalShippingTax,
	)
	if err != nil {
		return ReturnRefundAmountSummary{}, err
	}

	originalOutboundShippingAmount, ok :=
		shippingAmounts[orderItemIndex]
	if !ok || originalOutboundShippingAmount < 0 {
		return ReturnRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	originalOutboundShippingTaxAmount, ok :=
		shippingTaxes[orderItemIndex]
	if !ok || originalOutboundShippingTaxAmount < 0 {
		return ReturnRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	if selection.RefundOutboundShipping {
		result.OutboundShippingAmount =
			originalOutboundShippingAmount
		result.OutboundShippingTaxAmount =
			originalOutboundShippingTaxAmount

		result.StripeRefundAmount, err = safeAddPaymentAmount(
			result.StripeRefundAmount,
			result.OutboundShippingAmount,
		)
		if err != nil {
			return ReturnRefundAmountSummary{}, err
		}

		result.StripeRefundAmount, err = safeAddPaymentAmount(
			result.StripeRefundAmount,
			result.OutboundShippingTaxAmount,
		)
		if err != nil {
			return ReturnRefundAmountSummary{}, err
		}

		result.TotalSellerBurdenAmount =
			result.StripeRefundAmount
	}

	if selection.CoverReturnShipping {
		// Until Order stores a dedicated return-shipping quote, use the
		// target item's original outbound shipping amount as the modeled
		// reverse-route shipping amount.
		result.ReturnShippingAmount =
			originalOutboundShippingAmount

		returnShippingTaxProduct, err := safeMultiplyPaymentAmount(
			result.ReturnShippingAmount,
			orderdom.ConsumptionTaxRateStandard,
		)
		if err != nil {
			return ReturnRefundAmountSummary{}, err
		}

		result.ReturnShippingTaxAmount =
			returnShippingTaxProduct / 100

		result.TotalSellerBurdenAmount, err = safeAddPaymentAmount(
			result.StripeRefundAmount,
			result.ReturnShippingAmount,
		)
		if err != nil {
			return ReturnRefundAmountSummary{}, err
		}

		result.TotalSellerBurdenAmount, err = safeAddPaymentAmount(
			result.TotalSellerBurdenAmount,
			result.ReturnShippingTaxAmount,
		)
		if err != nil {
			return ReturnRefundAmountSummary{}, err
		}
	}

	return result, nil
}

// calculateProportionalMerchandiseRefund splits one tax-inclusive merchandise
// refund amount into merchandise and consumption-tax portions.
//
// The original Order allocation is authoritative. The frontend-selected amount
// is therefore distributed using the ratio between the original merchandise
// amount and the original tax-inclusive merchandise total.
//
// Example:
//
//	Original:
//	  merchandise = 10,000
//	  tax         = 1,000
//	  total       = 11,000
//
//	Requested refund:
//	  4,400
//
//	Result:
//	  merchandise refund = 4,000
//	  tax refund         =   400
//
// For amounts that cannot be divided exactly in whole yen, the merchandise
// portion is rounded down and the remaining yen is assigned to consumption tax.
// This guarantees:
//
//	merchandise refund + tax refund = requested tax-inclusive refund
func calculateProportionalMerchandiseRefund(
	full OrderItemRefundAmountSummary,
	requestedRefundAmount int,
) (int, int, error) {
	if requestedRefundAmount <= 0 ||
		full.MerchandiseAmount < 0 ||
		full.MerchandiseTaxAmount < 0 ||
		full.RefundAmount <= 0 ||
		requestedRefundAmount > full.RefundAmount {
		return 0, 0, ErrInvalidReturnRefundAmount
	}

	// A full refund must reuse the authoritative original allocation exactly.
	if requestedRefundAmount == full.RefundAmount {
		return full.MerchandiseAmount,
			full.MerchandiseTaxAmount,
			nil
	}

	// Resale merchandise is currently non-taxable. If the authoritative Order
	// allocation contains no merchandise tax, the complete requested amount is
	// merchandise.
	if full.MerchandiseTaxAmount == 0 {
		if requestedRefundAmount > full.MerchandiseAmount {
			return 0, 0, ErrInvalidReturnRefundAmount
		}

		return requestedRefundAmount, 0, nil
	}

	weightedMerchandiseAmount, err := safeMultiplyPaymentAmount(
		requestedRefundAmount,
		full.MerchandiseAmount,
	)
	if err != nil {
		return 0, 0, err
	}

	merchandiseAmount :=
		weightedMerchandiseAmount / full.RefundAmount

	merchandiseTaxAmount :=
		requestedRefundAmount - merchandiseAmount

	if merchandiseAmount < 0 ||
		merchandiseTaxAmount < 0 ||
		merchandiseAmount > full.MerchandiseAmount ||
		merchandiseTaxAmount > full.MerchandiseTaxAmount {
		return 0, 0, ErrInvalidReturnRefundAmount
	}

	calculatedRefundAmount, err := safeAddPaymentAmount(
		merchandiseAmount,
		merchandiseTaxAmount,
	)
	if err != nil {
		return 0, 0, err
	}

	if calculatedRefundAmount != requestedRefundAmount {
		return 0, 0, ErrInvalidReturnRefundAmount
	}

	return merchandiseAmount,
		merchandiseTaxAmount,
		nil
}
