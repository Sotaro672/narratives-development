// backend/internal/domain/order/payment_amount.go
package order

import (
	"errors"
	"sort"

	refunddom "narratives/internal/domain/refund"
)

// -------------------------------------------------------
// Errors
// -------------------------------------------------------

var (
	ErrInvalidPaymentAmount = errors.New(
		"order: invalid payment amount",
	)

	ErrInvalidOrderItemRefund = errors.New(
		"order: invalid order item refund",
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

	if order.ShippingQuoteSnapshot.Currency != ShippingQuoteCurrencyJPY {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)
	shippingAmount := order.ShippingQuoteSnapshot.Amount

	if shippingAmount < 0 {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	subtotalAmount := 0
	taxableAmount8 := 0

	// 配送料は標準税率10%の課税対象。
	taxableAmount10 := shippingAmount
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

	if subtotalAmount > maxInt-shippingAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	subtotalWithShipping := subtotalAmount + shippingAmount

	if subtotalWithShipping > maxInt-taxAmount {
		return PaymentAmountSummary{}, ErrInvalidPaymentAmount
	}

	totalAmount := subtotalWithShipping + taxAmount

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
	summary, err := CalculatePaymentAmountSummary(order)
	if err != nil {
		return 0, err
	}

	return summary.TotalAmount, nil
}

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
// Order 作成時の消費税は税率単位で集約して計算されるため、
// 単純に各商品の lineAmount * taxRate / 100 を返すと、
// Order 全体の税額や Settlement の税配賦と1円ずれる可能性があります。
//
// そのため税額は以下の順序で配賦します。
//
//  1. seller Account ごとに merchandise 8%, merchandise 10%,
//     shipping 10% を集約する。
//  2. 税率ごとの正規税額を最大剰余法で Account/component へ配賦する。
//  3. Account の merchandise component に配賦された税額を、
//     同一 Account・同一税率の商品へ最大剰余法で再配賦する。
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
// CalculatePaymentAmountSummary の ConsumptionTax と一致します。
func CalculateOrderItemRefundAmount(
	order Order,
	orderItemIndex int,
) (OrderItemRefundAmountSummary, error) {
	if orderItemIndex < 0 || orderItemIndex >= len(order.Items) {
		return OrderItemRefundAmountSummary{}, ErrInvalidOrderItemRefund
	}

	paymentSummary, err := CalculatePaymentAmountSummary(order)
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
// Opened Return Refund Amount
// -------------------------------------------------------

// OpenedReturnRefundAmountSummary represents the authoritative financial
// amounts for one opened-return policy.
//
// StripeRefundAmount is the amount that can be refunded against the original
// purchaser Charge.
//
// TotalBrandBurdenAmount additionally includes return-shipping cost that is not
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
	Policy refunddom.OpenedReturnRefundPolicy

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	StripeRefundAmount     int
	TotalBrandBurdenAmount int
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
//   - Additional company burden:
//     modeled return shipping + return shipping tax
//
// Half-refund rounding:
// JPY cannot represent fractional yen, so the total purchaser refund is rounded
// up to the nearest yen. Merchandise is also halved with the same rule and the
// remaining amount is treated as merchandise tax.
func CalculateOpenedReturnRefundAmount(
	order Order,
	orderItemIndex int,
	policy refunddom.OpenedReturnRefundPolicy,
) (OpenedReturnRefundAmountSummary, error) {
	if err := refunddom.ValidateOpenedReturnRefundPolicy(policy); err != nil {
		return OpenedReturnRefundAmountSummary{}, err
	}

	fullMerchandise, err := CalculateOrderItemRefundAmount(
		order,
		orderItemIndex,
	)
	if err != nil {
		return OpenedReturnRefundAmountSummary{}, err
	}

	switch policy {
	case refunddom.OpenedReturnRefundHalfMerchandise:
		stripeRefundAmount := halfPaymentAmountRoundedUp(
			fullMerchandise.RefundAmount,
		)

		merchandiseAmount := halfPaymentAmountRoundedUp(
			fullMerchandise.MerchandiseAmount,
		)

		merchandiseTaxAmount :=
			stripeRefundAmount - merchandiseAmount

		if stripeRefundAmount <= 0 ||
			merchandiseAmount < 0 ||
			merchandiseTaxAmount < 0 ||
			merchandiseAmount > fullMerchandise.MerchandiseAmount ||
			merchandiseTaxAmount > fullMerchandise.MerchandiseTaxAmount {
			return OpenedReturnRefundAmountSummary{},
				ErrInvalidOrderItemRefund
		}

		return OpenedReturnRefundAmountSummary{
			Policy: policy,

			MerchandiseAmount:    merchandiseAmount,
			MerchandiseTaxAmount: merchandiseTaxAmount,

			StripeRefundAmount:     stripeRefundAmount,
			TotalBrandBurdenAmount: stripeRefundAmount,
		}, nil

	case refunddom.OpenedReturnRefundMerchandiseOnly:
		return OpenedReturnRefundAmountSummary{
			Policy: policy,

			MerchandiseAmount:    fullMerchandise.MerchandiseAmount,
			MerchandiseTaxAmount: fullMerchandise.MerchandiseTaxAmount,

			StripeRefundAmount:     fullMerchandise.RefundAmount,
			TotalBrandBurdenAmount: fullMerchandise.RefundAmount,
		}, nil

	case refunddom.OpenedReturnRefundMerchandiseRoundTripShipping:
		shippingAmounts, err := calculateOrderItemShippingAmounts(order)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		_, totalShippingTax, err := calculateOrderItemTaxAllocations(order)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		shippingTaxes, err := allocateOpenedReturnShippingTaxToItems(
			shippingAmounts,
			totalShippingTax,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		outboundShippingAmount, ok := shippingAmounts[orderItemIndex]
		if !ok || outboundShippingAmount < 0 {
			return OpenedReturnRefundAmountSummary{},
				ErrInvalidOrderItemRefund
		}

		outboundShippingTaxAmount, ok := shippingTaxes[orderItemIndex]
		if !ok || outboundShippingTaxAmount < 0 {
			return OpenedReturnRefundAmountSummary{},
				ErrInvalidOrderItemRefund
		}

		stripeRefundAmount, err := safeAddPaymentAmount(
			fullMerchandise.RefundAmount,
			outboundShippingAmount,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		stripeRefundAmount, err = safeAddPaymentAmount(
			stripeRefundAmount,
			outboundShippingTaxAmount,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		// Until Order stores a dedicated return-shipping quote, use the
		// target item's original outbound shipping amount as the modeled
		// reverse-route shipping amount.
		returnShippingAmount := outboundShippingAmount

		returnShippingTaxProduct, err := safeMultiplyPaymentAmount(
			returnShippingAmount,
			ConsumptionTaxRateStandard,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		returnShippingTaxAmount :=
			returnShippingTaxProduct / 100

		totalBrandBurdenAmount, err := safeAddPaymentAmount(
			stripeRefundAmount,
			returnShippingAmount,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		totalBrandBurdenAmount, err = safeAddPaymentAmount(
			totalBrandBurdenAmount,
			returnShippingTaxAmount,
		)
		if err != nil {
			return OpenedReturnRefundAmountSummary{}, err
		}

		return OpenedReturnRefundAmountSummary{
			Policy: policy,

			MerchandiseAmount:    fullMerchandise.MerchandiseAmount,
			MerchandiseTaxAmount: fullMerchandise.MerchandiseTaxAmount,

			OutboundShippingAmount:    outboundShippingAmount,
			OutboundShippingTaxAmount: outboundShippingTaxAmount,

			ReturnShippingAmount:    returnShippingAmount,
			ReturnShippingTaxAmount: returnShippingTaxAmount,

			StripeRefundAmount:     stripeRefundAmount,
			TotalBrandBurdenAmount: totalBrandBurdenAmount,
		}, nil

	default:
		return OpenedReturnRefundAmountSummary{},
			refunddom.ErrInvalidOpenedReturnRefundPolicy
	}
}

type openedReturnShippingQuote struct {
	UnitAmount int
	Qty        int
	Amount     int
}

type openedReturnShippingTaxItem struct {
	Index     int
	Base      int
	Tax       int
	Remainder int
}

// calculateOrderItemShippingAmounts allocates the persisted outbound shipping
// quote to active Order items using the quote's unit amount and each item's Qty.
//
// Every active List Order item must map to exactly one shipping key and the
// resulting allocation must reconcile to ShippingQuoteSnapshot.Amount.
func calculateOrderItemShippingAmounts(
	order Order,
) (map[int]int, error) {
	snapshot := order.ShippingQuoteSnapshot

	if snapshot.Currency != ShippingQuoteCurrencyJPY ||
		snapshot.Amount < 0 ||
		len(snapshot.Items) == 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	quotes := make(map[refundShippingKey]openedReturnShippingQuote)

	for _, quoteItem := range snapshot.Items {
		if quoteItem.ListID == "" ||
			quoteItem.InventoryID == "" ||
			quoteItem.ModelID == "" ||
			quoteItem.Qty <= 0 ||
			quoteItem.UnitAmount < 0 ||
			quoteItem.Amount < 0 ||
			quoteItem.Currency != ShippingQuoteCurrencyJPY {
			return nil, ErrInvalidOrderItemRefund
		}

		expectedAmount, err := safeMultiplyPaymentAmount(
			quoteItem.UnitAmount,
			quoteItem.Qty,
		)
		if err != nil {
			return nil, err
		}

		if expectedAmount != quoteItem.Amount {
			return nil, ErrInvalidOrderItemRefund
		}

		key := refundShippingKey{
			ListID:      quoteItem.ListID,
			InventoryID: quoteItem.InventoryID,
			ModelID:     quoteItem.ModelID,
		}

		existing := quotes[key]

		if existing.Qty > 0 &&
			existing.UnitAmount != quoteItem.UnitAmount {
			return nil, ErrInvalidOrderItemRefund
		}

		nextQty, err := safeAddPaymentAmount(
			existing.Qty,
			quoteItem.Qty,
		)
		if err != nil {
			return nil, err
		}

		nextAmount, err := safeAddPaymentAmount(
			existing.Amount,
			quoteItem.Amount,
		)
		if err != nil {
			return nil, err
		}

		quotes[key] = openedReturnShippingQuote{
			UnitAmount: quoteItem.UnitAmount,
			Qty:        nextQty,
			Amount:     nextAmount,
		}
	}

	result := make(map[int]int)
	usedQty := make(map[refundShippingKey]int)
	usedAmount := make(map[refundShippingKey]int)
	totalAmount := 0

	for index, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		if item.Type != OrderItemTypeList ||
			item.ListID == "" ||
			item.InventoryID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			return nil, ErrInvalidOrderItemRefund
		}

		key := refundShippingKey{
			ListID:      item.ListID,
			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,
		}

		quote, exists := quotes[key]
		if !exists {
			return nil, ErrInvalidOrderItemRefund
		}

		itemShippingAmount, err := safeMultiplyPaymentAmount(
			quote.UnitAmount,
			item.Qty,
		)
		if err != nil {
			return nil, err
		}

		result[index] = itemShippingAmount

		usedQty[key], err = safeAddPaymentAmount(
			usedQty[key],
			item.Qty,
		)
		if err != nil {
			return nil, err
		}

		usedAmount[key], err = safeAddPaymentAmount(
			usedAmount[key],
			itemShippingAmount,
		)
		if err != nil {
			return nil, err
		}

		totalAmount, err = safeAddPaymentAmount(
			totalAmount,
			itemShippingAmount,
		)
		if err != nil {
			return nil, err
		}
	}

	if len(result) == 0 ||
		totalAmount != snapshot.Amount {
		return nil, ErrInvalidOrderItemRefund
	}

	for key, quote := range quotes {
		if usedQty[key] != quote.Qty ||
			usedAmount[key] != quote.Amount {
			return nil, ErrInvalidOrderItemRefund
		}
	}

	return result, nil
}

// allocateOpenedReturnShippingTaxToItems allocates the already-authoritative
// shipping tax total to Order items.
//
// The target shipping tax is produced by calculateOrderItemTaxAllocations,
// which reconciles to the original Order's ConsumptionTax. This second-stage
// allocation therefore preserves the original payment tax exactly instead of
// recalculating each item's outbound shipping tax independently.
func allocateOpenedReturnShippingTaxToItems(
	shippingAmounts map[int]int,
	targetTax int,
) (map[int]int, error) {
	if len(shippingAmounts) == 0 ||
		targetTax < 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	items := make([]openedReturnShippingTaxItem, 0, len(shippingAmounts))
	allocatedTax := 0

	for index, amount := range shippingAmounts {
		if index < 0 || amount < 0 {
			return nil, ErrInvalidOrderItemRefund
		}

		product, err := safeMultiplyPaymentAmount(
			amount,
			ConsumptionTaxRateStandard,
		)
		if err != nil {
			return nil, err
		}

		tax := product / 100
		remainder := product % 100

		allocatedTax, err = safeAddPaymentAmount(
			allocatedTax,
			tax,
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			openedReturnShippingTaxItem{
				Index:     index,
				Base:      amount,
				Tax:       tax,
				Remainder: remainder,
			},
		)
	}

	residual := targetTax - allocatedTax

	if residual < 0 ||
		residual > len(items) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(
		items,
		func(i, j int) bool {
			if items[i].Remainder != items[j].Remainder {
				return items[i].Remainder > items[j].Remainder
			}

			return items[i].Index < items[j].Index
		},
	)

	for index := 0; index < residual; index++ {
		items[index].Tax++
	}

	result := make(map[int]int, len(items))
	allocatedResultTax := 0

	for _, item := range items {
		result[item.Index] = item.Tax

		var err error
		allocatedResultTax, err = safeAddPaymentAmount(
			allocatedResultTax,
			item.Tax,
		)
		if err != nil {
			return nil, err
		}
	}

	if allocatedResultTax != targetTax {
		return nil, ErrInvalidOrderItemRefund
	}

	return result, nil
}

func halfPaymentAmountRoundedUp(
	amount int,
) int {
	if amount <= 0 {
		return 0
	}

	return amount/2 + amount%2
}

// -------------------------------------------------------
// Refund Tax Allocation
// -------------------------------------------------------

type refundTaxComponentKind string

const (
	refundTaxComponentMerchandise8  refundTaxComponentKind = "merchandise_8"
	refundTaxComponentMerchandise10 refundTaxComponentKind = "merchandise_10"
	refundTaxComponentShipping10    refundTaxComponentKind = "shipping_10"
)

type refundTaxComponent struct {
	AccountID string
	Kind      refundTaxComponentKind
	Base      int
	Tax       int
	Remainder int
}

type refundTaxItem struct {
	Index     int
	AccountID string
	Rate      int
	Base      int
	Tax       int
	Remainder int
}

type refundTaxGroupKey struct {
	AccountID string
	Rate      int
}

type refundTaxAccountBuilder struct {
	MerchandiseAmount8  int
	MerchandiseAmount10 int
	ShippingAmount      int
}

type refundShippingKey struct {
	ListID      string
	InventoryID string
	ModelID     string
}

type refundShippingBinding struct {
	AccountID string
	Qty       int
}

func calculateOrderItemTaxAllocations(
	order Order,
) (map[int]int, int, error) {
	builders := make(map[string]*refundTaxAccountBuilder)
	bindings := make(map[refundShippingKey]refundShippingBinding)
	items := make([]refundTaxItem, 0, len(order.Items))

	for index, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		if item.Type != OrderItemTypeList ||
			item.ListID == "" ||
			item.InventoryID == "" ||
			item.ModelID == "" ||
			item.SellerSnapshot.AccountID == "" ||
			item.Price < 0 ||
			item.Qty <= 0 {
			return nil, 0, ErrInvalidOrderItemRefund
		}

		lineAmount, err := safeMultiplyPaymentAmount(
			item.Price,
			item.Qty,
		)
		if err != nil {
			return nil, 0, err
		}

		accountID := item.SellerSnapshot.AccountID
		builder := builders[accountID]

		if builder == nil {
			builder = &refundTaxAccountBuilder{}
			builders[accountID] = builder
		}

		switch item.ConsumptionTaxRate {
		case ConsumptionTaxRateReduced:
			builder.MerchandiseAmount8, err = safeAddPaymentAmount(
				builder.MerchandiseAmount8,
				lineAmount,
			)

		case ConsumptionTaxRateStandard:
			builder.MerchandiseAmount10, err = safeAddPaymentAmount(
				builder.MerchandiseAmount10,
				lineAmount,
			)

		default:
			return nil, 0, ErrInvalidOrderItemRefund
		}

		if err != nil {
			return nil, 0, err
		}

		items = append(
			items,
			refundTaxItem{
				Index:     index,
				AccountID: accountID,
				Rate:      item.ConsumptionTaxRate,
				Base:      lineAmount,
			},
		)

		key := refundShippingKey{
			ListID:      item.ListID,
			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,
		}

		binding, exists := bindings[key]

		if exists && binding.AccountID != accountID {
			return nil, 0, ErrInvalidOrderItemRefund
		}

		nextQty, err := safeAddPaymentAmount(
			binding.Qty,
			item.Qty,
		)
		if err != nil {
			return nil, 0, err
		}

		bindings[key] = refundShippingBinding{
			AccountID: accountID,
			Qty:       nextQty,
		}
	}

	if len(items) == 0 || len(builders) == 0 {
		return nil, 0, ErrInvalidOrderItemRefund
	}

	if err := applyRefundShippingAmounts(
		order,
		builders,
		bindings,
	); err != nil {
		return nil, 0, err
	}

	groupTaxes, shippingTax, err := allocateRefundTaxComponents(builders)
	if err != nil {
		return nil, 0, err
	}

	itemTaxes, err := allocateRefundTaxToItems(
		items,
		groupTaxes,
	)
	if err != nil {
		return nil, 0, err
	}

	return itemTaxes, shippingTax, nil
}

func applyRefundShippingAmounts(
	order Order,
	builders map[string]*refundTaxAccountBuilder,
	bindings map[refundShippingKey]refundShippingBinding,
) error {
	snapshot := order.ShippingQuoteSnapshot

	if snapshot.Currency != ShippingQuoteCurrencyJPY ||
		snapshot.Amount < 0 ||
		len(snapshot.Items) == 0 {
		return ErrInvalidOrderItemRefund
	}

	quotedQty := make(map[refundShippingKey]int, len(bindings))
	shippingTotal := 0

	for _, item := range snapshot.Items {
		if item.ListID == "" ||
			item.InventoryID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 ||
			item.UnitAmount < 0 ||
			item.Amount < 0 ||
			item.Currency != ShippingQuoteCurrencyJPY {
			return ErrInvalidOrderItemRefund
		}

		expectedAmount, err := safeMultiplyPaymentAmount(
			item.UnitAmount,
			item.Qty,
		)
		if err != nil {
			return err
		}

		if expectedAmount != item.Amount {
			return ErrInvalidOrderItemRefund
		}

		key := refundShippingKey{
			ListID:      item.ListID,
			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,
		}

		binding, exists := bindings[key]
		if !exists {
			return ErrInvalidOrderItemRefund
		}

		builder := builders[binding.AccountID]
		if builder == nil {
			return ErrInvalidOrderItemRefund
		}

		builder.ShippingAmount, err = safeAddPaymentAmount(
			builder.ShippingAmount,
			item.Amount,
		)
		if err != nil {
			return err
		}

		quotedQty[key], err = safeAddPaymentAmount(
			quotedQty[key],
			item.Qty,
		)
		if err != nil {
			return err
		}

		shippingTotal, err = safeAddPaymentAmount(
			shippingTotal,
			item.Amount,
		)
		if err != nil {
			return err
		}
	}

	if shippingTotal != snapshot.Amount {
		return ErrInvalidOrderItemRefund
	}

	for key, binding := range bindings {
		if quotedQty[key] != binding.Qty {
			return ErrInvalidOrderItemRefund
		}
	}

	return nil
}

func allocateRefundTaxComponents(
	builders map[string]*refundTaxAccountBuilder,
) (map[refundTaxGroupKey]int, int, error) {
	reducedComponents := make([]refundTaxComponent, 0, len(builders))
	standardComponents := make([]refundTaxComponent, 0, len(builders)*2)

	for accountID, builder := range builders {
		if builder == nil {
			return nil, 0, ErrInvalidOrderItemRefund
		}

		if builder.MerchandiseAmount8 > 0 {
			reducedComponents = append(
				reducedComponents,
				refundTaxComponent{
					AccountID: accountID,
					Kind:      refundTaxComponentMerchandise8,
					Base:      builder.MerchandiseAmount8,
				},
			)
		}

		if builder.MerchandiseAmount10 > 0 {
			standardComponents = append(
				standardComponents,
				refundTaxComponent{
					AccountID: accountID,
					Kind:      refundTaxComponentMerchandise10,
					Base:      builder.MerchandiseAmount10,
				},
			)
		}

		if builder.ShippingAmount > 0 {
			standardComponents = append(
				standardComponents,
				refundTaxComponent{
					AccountID: accountID,
					Kind:      refundTaxComponentShipping10,
					Base:      builder.ShippingAmount,
				},
			)
		}
	}

	var err error

	reducedComponents, err = allocateRefundTaxByRate(
		reducedComponents,
		ConsumptionTaxRateReduced,
	)
	if err != nil {
		return nil, 0, err
	}

	standardComponents, err = allocateRefundTaxByRate(
		standardComponents,
		ConsumptionTaxRateStandard,
	)
	if err != nil {
		return nil, 0, err
	}

	groupTaxes := make(map[refundTaxGroupKey]int)
	shippingTax := 0

	for _, component := range reducedComponents {
		groupTaxes[refundTaxGroupKey{
			AccountID: component.AccountID,
			Rate:      ConsumptionTaxRateReduced,
		}] = component.Tax
	}

	for _, component := range standardComponents {
		switch component.Kind {
		case refundTaxComponentMerchandise10:
			groupTaxes[refundTaxGroupKey{
				AccountID: component.AccountID,
				Rate:      ConsumptionTaxRateStandard,
			}] = component.Tax

		case refundTaxComponentShipping10:
			shippingTax, err = safeAddPaymentAmount(
				shippingTax,
				component.Tax,
			)
			if err != nil {
				return nil, 0, err
			}

		default:
			return nil, 0, ErrInvalidOrderItemRefund
		}
	}

	return groupTaxes, shippingTax, nil
}

func allocateRefundTaxByRate(
	components []refundTaxComponent,
	rate int,
) ([]refundTaxComponent, error) {
	if rate <= 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	if len(components) == 0 {
		return components, nil
	}

	totalBase := 0
	allocatedTax := 0

	for index := range components {
		component := &components[index]

		if component.Base < 0 {
			return nil, ErrInvalidOrderItemRefund
		}

		var err error

		totalBase, err = safeAddPaymentAmount(
			totalBase,
			component.Base,
		)
		if err != nil {
			return nil, err
		}

		product, err := safeMultiplyPaymentAmount(
			component.Base,
			rate,
		)
		if err != nil {
			return nil, err
		}

		component.Tax = product / 100
		component.Remainder = product % 100

		allocatedTax, err = safeAddPaymentAmount(
			allocatedTax,
			component.Tax,
		)
		if err != nil {
			return nil, err
		}
	}

	totalProduct, err := safeMultiplyPaymentAmount(
		totalBase,
		rate,
	)
	if err != nil {
		return nil, err
	}

	canonicalTax := totalProduct / 100
	residual := canonicalTax - allocatedTax

	if residual < 0 || residual > len(components) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(
		components,
		func(i, j int) bool {
			if components[i].Remainder != components[j].Remainder {
				return components[i].Remainder > components[j].Remainder
			}

			if components[i].AccountID != components[j].AccountID {
				return components[i].AccountID < components[j].AccountID
			}

			return components[i].Kind < components[j].Kind
		},
	)

	for index := 0; index < residual; index++ {
		components[index].Tax++
	}

	return components, nil
}

func allocateRefundTaxToItems(
	items []refundTaxItem,
	groupTaxes map[refundTaxGroupKey]int,
) (map[int]int, error) {
	groups := make(map[refundTaxGroupKey][]refundTaxItem)

	for _, item := range items {
		key := refundTaxGroupKey{
			AccountID: item.AccountID,
			Rate:      item.Rate,
		}

		groups[key] = append(
			groups[key],
			item,
		)
	}

	result := make(map[int]int, len(items))

	for key, groupItems := range groups {
		targetTax, exists := groupTaxes[key]
		if !exists {
			return nil, ErrInvalidOrderItemRefund
		}

		allocatedItems, err := allocateRefundItemTaxGroup(
			groupItems,
			key.Rate,
			targetTax,
		)
		if err != nil {
			return nil, err
		}

		for _, item := range allocatedItems {
			if _, exists := result[item.Index]; exists {
				return nil, ErrInvalidOrderItemRefund
			}

			result[item.Index] = item.Tax
		}
	}

	if len(result) != len(items) {
		return nil, ErrInvalidOrderItemRefund
	}

	return result, nil
}

func allocateRefundItemTaxGroup(
	items []refundTaxItem,
	rate int,
	targetTax int,
) ([]refundTaxItem, error) {
	if len(items) == 0 || rate <= 0 || targetTax < 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	allocatedTax := 0

	for index := range items {
		item := &items[index]

		if item.Base < 0 {
			return nil, ErrInvalidOrderItemRefund
		}

		product, err := safeMultiplyPaymentAmount(
			item.Base,
			rate,
		)
		if err != nil {
			return nil, err
		}

		item.Tax = product / 100
		item.Remainder = product % 100

		allocatedTax, err = safeAddPaymentAmount(
			allocatedTax,
			item.Tax,
		)
		if err != nil {
			return nil, err
		}
	}

	residual := targetTax - allocatedTax

	if residual < 0 || residual > len(items) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(
		items,
		func(i, j int) bool {
			if items[i].Remainder != items[j].Remainder {
				return items[i].Remainder > items[j].Remainder
			}

			return items[i].Index < items[j].Index
		},
	)

	for index := 0; index < residual; index++ {
		items[index].Tax++
	}

	return items, nil
}

// -------------------------------------------------------
// Safe Integer Operations
// -------------------------------------------------------

func safeAddPaymentAmount(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)

	if left > maxInt-right {
		return 0, ErrInvalidPaymentAmount
	}

	return left + right, nil
}

func safeMultiplyPaymentAmount(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrInvalidPaymentAmount
	}

	if left == 0 || right == 0 {
		return 0, nil
	}

	maxInt := int(^uint(0) >> 1)

	if left > maxInt/right {
		return 0, ErrInvalidPaymentAmount
	}

	return left * right, nil
}
