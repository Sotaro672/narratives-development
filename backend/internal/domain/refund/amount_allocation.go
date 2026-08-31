// backend/internal/domain/refund/amount_allocation.go
package refund

import (
	"sort"

	orderdom "narratives/internal/domain/order"
)

// -------------------------------------------------------
// Opened Return Shipping Allocation
// -------------------------------------------------------

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
// List items must map to their persisted shipping key. Resale items must map to
// a zero-yen resale shipping snapshot. The resulting allocation must reconcile
// to ShippingQuoteSnapshot.Amount, and every active Order item receives exactly
// one shipping allocation entry.
func calculateOrderItemShippingAmounts(order orderdom.Order) (map[int]int, error) {
	snapshot := order.ShippingQuoteSnapshot
	if snapshot.Currency != orderdom.ShippingQuoteCurrencyJPY || snapshot.Amount < 0 || len(snapshot.Items) == 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	quotes := make(map[refundShippingKey]openedReturnShippingQuote)
	resaleQuotedQty := make(map[string]int)

	for _, quoteItem := range snapshot.Items {
		if quoteItem.Qty <= 0 || quoteItem.UnitAmount < 0 || quoteItem.Amount < 0 || quoteItem.Currency != orderdom.ShippingQuoteCurrencyJPY {
			return nil, ErrInvalidOrderItemRefund
		}

		expectedAmount, err := safeMultiplyPaymentAmount(quoteItem.UnitAmount, quoteItem.Qty)
		if err != nil {
			return nil, err
		}
		if expectedAmount != quoteItem.Amount {
			return nil, ErrInvalidOrderItemRefund
		}

		switch quoteItem.Type {
		case orderdom.OrderItemTypeList:
			if quoteItem.ListID == "" || quoteItem.InventoryID == "" || quoteItem.ModelID == "" || quoteItem.ResaleID != "" {
				return nil, ErrInvalidOrderItemRefund
			}

			key := refundShippingKey{
				ListID:      quoteItem.ListID,
				InventoryID: quoteItem.InventoryID,
				ModelID:     quoteItem.ModelID,
			}

			existing := quotes[key]
			if existing.Qty > 0 && existing.UnitAmount != quoteItem.UnitAmount {
				return nil, ErrInvalidOrderItemRefund
			}

			nextQty, err := safeAddPaymentAmount(existing.Qty, quoteItem.Qty)
			if err != nil {
				return nil, err
			}

			nextAmount, err := safeAddPaymentAmount(existing.Amount, quoteItem.Amount)
			if err != nil {
				return nil, err
			}

			quotes[key] = openedReturnShippingQuote{
				UnitAmount: quoteItem.UnitAmount,
				Qty:        nextQty,
				Amount:     nextAmount,
			}

		case orderdom.OrderItemTypeResale:
			if quoteItem.ResaleID == "" || quoteItem.ListID != "" || quoteItem.InventoryID != "" || quoteItem.ModelID != "" ||
				quoteItem.Qty != 1 || quoteItem.UnitAmount != 0 || quoteItem.Amount != 0 {
				return nil, ErrInvalidOrderItemRefund
			}

			resaleQuotedQty[quoteItem.ResaleID], err = safeAddPaymentAmount(
				resaleQuotedQty[quoteItem.ResaleID],
				quoteItem.Qty,
			)
			if err != nil {
				return nil, err
			}

		default:
			return nil, ErrInvalidOrderItemRefund
		}
	}

	result := make(map[int]int)
	usedQty := make(map[refundShippingKey]int)
	usedAmount := make(map[refundShippingKey]int)
	usedResaleQty := make(map[string]int)
	totalAmount := 0

	for index, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			if item.ListID == "" || item.InventoryID == "" || item.ModelID == "" || item.ResaleID != "" || item.Qty <= 0 {
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

			itemShippingAmount, err := safeMultiplyPaymentAmount(quote.UnitAmount, item.Qty)
			if err != nil {
				return nil, err
			}

			result[index] = itemShippingAmount

			usedQty[key], err = safeAddPaymentAmount(usedQty[key], item.Qty)
			if err != nil {
				return nil, err
			}

			usedAmount[key], err = safeAddPaymentAmount(usedAmount[key], itemShippingAmount)
			if err != nil {
				return nil, err
			}

			totalAmount, err = safeAddPaymentAmount(totalAmount, itemShippingAmount)
			if err != nil {
				return nil, err
			}

		case orderdom.OrderItemTypeResale:
			if item.ResaleID == "" || item.ListID != "" || item.InventoryID != "" || item.ModelID != "" || item.Qty != 1 {
				return nil, ErrInvalidOrderItemRefund
			}
			if resaleQuotedQty[item.ResaleID] == 0 {
				return nil, ErrInvalidOrderItemRefund
			}

			var err error
			usedResaleQty[item.ResaleID], err = safeAddPaymentAmount(usedResaleQty[item.ResaleID], item.Qty)
			if err != nil {
				return nil, err
			}

			result[index] = 0

		default:
			return nil, ErrInvalidOrderItemRefund
		}
	}

	if len(result) == 0 || totalAmount != snapshot.Amount {
		return nil, ErrInvalidOrderItemRefund
	}

	for key, quote := range quotes {
		if usedQty[key] != quote.Qty || usedAmount[key] != quote.Amount {
			return nil, ErrInvalidOrderItemRefund
		}
	}

	for resaleID, quotedQty := range resaleQuotedQty {
		if usedResaleQty[resaleID] != quotedQty {
			return nil, ErrInvalidOrderItemRefund
		}
	}

	return result, nil
}

func allocateOpenedReturnShippingTaxToItems(shippingAmounts map[int]int, targetTax int) (map[int]int, error) {
	if len(shippingAmounts) == 0 || targetTax < 0 {
		return nil, ErrInvalidOrderItemRefund
	}

	items := make([]openedReturnShippingTaxItem, 0, len(shippingAmounts))
	allocatedTax := 0

	for index, amount := range shippingAmounts {
		if index < 0 || amount < 0 {
			return nil, ErrInvalidOrderItemRefund
		}

		product, err := safeMultiplyPaymentAmount(amount, orderdom.ConsumptionTaxRateStandard)
		if err != nil {
			return nil, err
		}

		tax := product / 100
		remainder := product % 100

		allocatedTax, err = safeAddPaymentAmount(allocatedTax, tax)
		if err != nil {
			return nil, err
		}

		items = append(items, openedReturnShippingTaxItem{
			Index:     index,
			Base:      amount,
			Tax:       tax,
			Remainder: remainder,
		})
	}

	residual := targetTax - allocatedTax
	if residual < 0 || residual > len(items) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Remainder != items[j].Remainder {
			return items[i].Remainder > items[j].Remainder
		}
		return items[i].Index < items[j].Index
	})

	for index := 0; index < residual; index++ {
		items[index].Tax++
	}

	result := make(map[int]int, len(items))
	allocatedResultTax := 0

	for _, item := range items {
		result[item.Index] = item.Tax

		var err error
		allocatedResultTax, err = safeAddPaymentAmount(allocatedResultTax, item.Tax)
		if err != nil {
			return nil, err
		}
	}

	if allocatedResultTax != targetTax {
		return nil, ErrInvalidOrderItemRefund
	}

	return result, nil
}

func halfPaymentAmountRoundedUp(amount int) int {
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

func calculateOrderItemTaxAllocations(order orderdom.Order) (map[int]int, int, error) {
	builders := make(map[string]*refundTaxAccountBuilder)
	bindings := make(map[refundShippingKey]refundShippingBinding)
	resaleBindings := make(map[string]int)
	taxableItems := make([]refundTaxItem, 0, len(order.Items))
	resaleIndexes := make([]int, 0)
	activeItemCount := 0

	for index, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		activeItemCount++

		if item.Price < 0 || item.Qty <= 0 {
			return nil, 0, ErrInvalidOrderItemRefund
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			if item.ListID == "" || item.InventoryID == "" || item.ModelID == "" || item.ResaleID != "" || item.SellerSnapshot.AccountID == "" {
				return nil, 0, ErrInvalidOrderItemRefund
			}

			lineAmount, err := safeMultiplyPaymentAmount(item.Price, item.Qty)
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
			case orderdom.ConsumptionTaxRateReduced:
				builder.MerchandiseAmount8, err = safeAddPaymentAmount(
					builder.MerchandiseAmount8,
					lineAmount,
				)

			case orderdom.ConsumptionTaxRateStandard:
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

			taxableItems = append(taxableItems, refundTaxItem{
				Index:     index,
				AccountID: accountID,
				Rate:      item.ConsumptionTaxRate,
				Base:      lineAmount,
			})

			key := refundShippingKey{
				ListID:      item.ListID,
				InventoryID: item.InventoryID,
				ModelID:     item.ModelID,
			}

			binding, exists := bindings[key]
			if exists && binding.AccountID != accountID {
				return nil, 0, ErrInvalidOrderItemRefund
			}

			nextQty, err := safeAddPaymentAmount(binding.Qty, item.Qty)
			if err != nil {
				return nil, 0, err
			}

			bindings[key] = refundShippingBinding{
				AccountID: accountID,
				Qty:       nextQty,
			}

		case orderdom.OrderItemTypeResale:
			if item.ResaleID == "" || item.ListID != "" || item.InventoryID != "" || item.ModelID != "" || item.Qty != 1 {
				return nil, 0, ErrInvalidOrderItemRefund
			}

			var err error
			resaleBindings[item.ResaleID], err = safeAddPaymentAmount(
				resaleBindings[item.ResaleID],
				item.Qty,
			)
			if err != nil {
				return nil, 0, err
			}

			resaleIndexes = append(resaleIndexes, index)

		default:
			return nil, 0, ErrInvalidOrderItemRefund
		}
	}

	if activeItemCount == 0 {
		return nil, 0, ErrInvalidOrderItemRefund
	}

	if err := applyRefundShippingAmounts(order, builders, bindings, resaleBindings); err != nil {
		return nil, 0, err
	}

	groupTaxes, shippingTax, err := allocateRefundTaxComponents(builders)
	if err != nil {
		return nil, 0, err
	}

	itemTaxes, err := allocateRefundTaxToItems(taxableItems, groupTaxes)
	if err != nil {
		return nil, 0, err
	}

	for _, index := range resaleIndexes {
		if _, exists := itemTaxes[index]; exists {
			return nil, 0, ErrInvalidOrderItemRefund
		}
		itemTaxes[index] = 0
	}

	if len(itemTaxes) != activeItemCount {
		return nil, 0, ErrInvalidOrderItemRefund
	}

	return itemTaxes, shippingTax, nil
}

func applyRefundShippingAmounts(
	order orderdom.Order,
	builders map[string]*refundTaxAccountBuilder,
	bindings map[refundShippingKey]refundShippingBinding,
	resaleBindings map[string]int,
) error {
	snapshot := order.ShippingQuoteSnapshot
	if snapshot.Currency != orderdom.ShippingQuoteCurrencyJPY || snapshot.Amount < 0 || len(snapshot.Items) == 0 {
		return ErrInvalidOrderItemRefund
	}

	quotedQty := make(map[refundShippingKey]int, len(bindings))
	quotedResaleQty := make(map[string]int, len(resaleBindings))
	shippingTotal := 0

	for _, item := range snapshot.Items {
		if item.Qty <= 0 || item.UnitAmount < 0 || item.Amount < 0 || item.Currency != orderdom.ShippingQuoteCurrencyJPY {
			return ErrInvalidOrderItemRefund
		}

		expectedAmount, err := safeMultiplyPaymentAmount(item.UnitAmount, item.Qty)
		if err != nil {
			return err
		}
		if expectedAmount != item.Amount {
			return ErrInvalidOrderItemRefund
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			if item.ListID == "" || item.InventoryID == "" || item.ModelID == "" || item.ResaleID != "" {
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

			builder.ShippingAmount, err = safeAddPaymentAmount(builder.ShippingAmount, item.Amount)
			if err != nil {
				return err
			}

			quotedQty[key], err = safeAddPaymentAmount(quotedQty[key], item.Qty)
			if err != nil {
				return err
			}

			shippingTotal, err = safeAddPaymentAmount(shippingTotal, item.Amount)
			if err != nil {
				return err
			}

		case orderdom.OrderItemTypeResale:
			if item.ResaleID == "" || item.ListID != "" || item.InventoryID != "" || item.ModelID != "" ||
				item.Qty != 1 || item.UnitAmount != 0 || item.Amount != 0 {
				return ErrInvalidOrderItemRefund
			}
			if resaleBindings[item.ResaleID] == 0 {
				return ErrInvalidOrderItemRefund
			}

			quotedResaleQty[item.ResaleID], err = safeAddPaymentAmount(
				quotedResaleQty[item.ResaleID],
				item.Qty,
			)
			if err != nil {
				return err
			}

		default:
			return ErrInvalidOrderItemRefund
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

	for resaleID, expectedQty := range resaleBindings {
		if quotedResaleQty[resaleID] != expectedQty {
			return ErrInvalidOrderItemRefund
		}
	}

	return nil
}

func allocateRefundTaxComponents(builders map[string]*refundTaxAccountBuilder) (map[refundTaxGroupKey]int, int, error) {
	reducedComponents := make([]refundTaxComponent, 0, len(builders))
	standardComponents := make([]refundTaxComponent, 0, len(builders)*2)

	for accountID, builder := range builders {
		if builder == nil {
			return nil, 0, ErrInvalidOrderItemRefund
		}

		if builder.MerchandiseAmount8 > 0 {
			reducedComponents = append(reducedComponents, refundTaxComponent{
				AccountID: accountID,
				Kind:      refundTaxComponentMerchandise8,
				Base:      builder.MerchandiseAmount8,
			})
		}

		if builder.MerchandiseAmount10 > 0 {
			standardComponents = append(standardComponents, refundTaxComponent{
				AccountID: accountID,
				Kind:      refundTaxComponentMerchandise10,
				Base:      builder.MerchandiseAmount10,
			})
		}

		if builder.ShippingAmount > 0 {
			standardComponents = append(standardComponents, refundTaxComponent{
				AccountID: accountID,
				Kind:      refundTaxComponentShipping10,
				Base:      builder.ShippingAmount,
			})
		}
	}

	var err error

	reducedComponents, err = allocateRefundTaxByRate(
		reducedComponents,
		orderdom.ConsumptionTaxRateReduced,
	)
	if err != nil {
		return nil, 0, err
	}

	standardComponents, err = allocateRefundTaxByRate(
		standardComponents,
		orderdom.ConsumptionTaxRateStandard,
	)
	if err != nil {
		return nil, 0, err
	}

	groupTaxes := make(map[refundTaxGroupKey]int)
	shippingTax := 0

	for _, component := range reducedComponents {
		groupTaxes[refundTaxGroupKey{
			AccountID: component.AccountID,
			Rate:      orderdom.ConsumptionTaxRateReduced,
		}] = component.Tax
	}

	for _, component := range standardComponents {
		switch component.Kind {
		case refundTaxComponentMerchandise10:
			groupTaxes[refundTaxGroupKey{
				AccountID: component.AccountID,
				Rate:      orderdom.ConsumptionTaxRateStandard,
			}] = component.Tax

		case refundTaxComponentShipping10:
			shippingTax, err = safeAddPaymentAmount(shippingTax, component.Tax)
			if err != nil {
				return nil, 0, err
			}

		default:
			return nil, 0, ErrInvalidOrderItemRefund
		}
	}

	return groupTaxes, shippingTax, nil
}

func allocateRefundTaxByRate(components []refundTaxComponent, rate int) ([]refundTaxComponent, error) {
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
		totalBase, err = safeAddPaymentAmount(totalBase, component.Base)
		if err != nil {
			return nil, err
		}

		product, err := safeMultiplyPaymentAmount(component.Base, rate)
		if err != nil {
			return nil, err
		}

		component.Tax = product / 100
		component.Remainder = product % 100

		allocatedTax, err = safeAddPaymentAmount(allocatedTax, component.Tax)
		if err != nil {
			return nil, err
		}
	}

	totalProduct, err := safeMultiplyPaymentAmount(totalBase, rate)
	if err != nil {
		return nil, err
	}

	canonicalTax := totalProduct / 100
	residual := canonicalTax - allocatedTax

	if residual < 0 || residual > len(components) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(components, func(i, j int) bool {
		if components[i].Remainder != components[j].Remainder {
			return components[i].Remainder > components[j].Remainder
		}
		if components[i].AccountID != components[j].AccountID {
			return components[i].AccountID < components[j].AccountID
		}
		return components[i].Kind < components[j].Kind
	})

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
		groups[key] = append(groups[key], item)
	}

	result := make(map[int]int, len(items))

	for key, groupItems := range groups {
		targetTax, exists := groupTaxes[key]
		if !exists {
			return nil, ErrInvalidOrderItemRefund
		}

		allocatedItems, err := allocateRefundItemTaxGroup(groupItems, key.Rate, targetTax)
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

		product, err := safeMultiplyPaymentAmount(item.Base, rate)
		if err != nil {
			return nil, err
		}

		item.Tax = product / 100
		item.Remainder = product % 100

		allocatedTax, err = safeAddPaymentAmount(allocatedTax, item.Tax)
		if err != nil {
			return nil, err
		}
	}

	residual := targetTax - allocatedTax
	if residual < 0 || residual > len(items) {
		return nil, ErrInvalidOrderItemRefund
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Remainder != items[j].Remainder {
			return items[i].Remainder > items[j].Remainder
		}
		return items[i].Index < items[j].Index
	})

	for index := 0; index < residual; index++ {
		items[index].Tax++
	}

	return items, nil
}

// -------------------------------------------------------
// Safe Integer Operations
// -------------------------------------------------------

func safeAddPaymentAmount(left int, right int) (int, error) {
	if left < 0 || right < 0 {
		return 0, orderdom.ErrInvalidPaymentAmount
	}

	maxInt := int(^uint(0) >> 1)
	if left > maxInt-right {
		return 0, orderdom.ErrInvalidPaymentAmount
	}

	return left + right, nil
}

func safeMultiplyPaymentAmount(left int, right int) (int, error) {
	if left < 0 || right < 0 {
		return 0, orderdom.ErrInvalidPaymentAmount
	}

	if left == 0 || right == 0 {
		return 0, nil
	}

	maxInt := int(^uint(0) >> 1)
	if left > maxInt/right {
		return 0, orderdom.ErrInvalidPaymentAmount
	}

	return left * right, nil
}
