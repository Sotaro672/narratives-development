// backend/internal/domain/settlement/calculator.go
package settlement

import (
	"context"
	"errors"
	"sort"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrCalculatorPlatformFeeMissing = errors.New(
		"settlement: platform fee calculator is not configured",
	)
	ErrCalculatorInvalidOrder = errors.New(
		"settlement: invalid order",
	)
	ErrCalculatorInvalidPayment = errors.New(
		"settlement: invalid payment",
	)
	ErrCalculatorPaymentOrderMismatch = errors.New(
		"settlement: payment does not belong to order",
	)
	ErrCalculatorPaymentAmountMismatch = errors.New(
		"settlement: payment amount does not match order amount",
	)
	ErrCalculatorUnsupportedOrderItem = errors.New(
		"settlement: unsupported order item",
	)
	ErrCalculatorInvalidSellerSnapshot = errors.New(
		"settlement: invalid seller snapshot",
	)
	ErrCalculatorSellerMismatch = errors.New(
		"settlement: seller snapshot mismatch",
	)
	ErrCalculatorInvalidShippingQuote = errors.New(
		"settlement: invalid shipping quote",
	)
	ErrCalculatorShippingQuoteMismatch = errors.New(
		"settlement: shipping quote does not match order items",
	)
	ErrCalculatorInvalidTaxRate = errors.New(
		"settlement: invalid consumption tax rate",
	)
	ErrCalculatorAmountOverflow = errors.New(
		"settlement: amount overflow",
	)
	ErrCalculatorInvalidPlatformFee = errors.New(
		"settlement: invalid platform fee",
	)
	ErrCalculatorAllocationEmpty = errors.New(
		"settlement: allocation is empty",
	)
	ErrCalculatorAllocationAmountMismatch = errors.New(
		"settlement: allocation total does not match payment amount",
	)
	ErrInvalidPlatformFeeRate = errors.New(
		"settlement: invalid platform fee rate",
	)
	ErrInvalidPlatformFeeBase = errors.New(
		"settlement: invalid platform fee base",
	)
)

// ============================================================
// Allocation
// ============================================================

// Allocation represents the amount attributable to one seller payout identity.
//
// Primary List sales are aggregated by AccountID.
// Resale transactions are aggregated by PayoutAccountID.
//
// GrossAmount:
//
//	MerchandiseAmount
//	+ MerchandiseTaxAmount
//	+ ShippingAmount
//	+ ShippingTaxAmount
//
// Resale merchandise is included in MerchandiseAmount but contributes neither
// MerchandiseTaxAmount nor ShippingAmount.
//
// TransferAmount:
//
//	GrossAmount - PlatformFeeAmount
type Allocation struct {
	Seller SellerIdentity

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	ShippingAmount    int
	ShippingTaxAmount int

	GrossAmount       int
	PlatformFeeAmount int
	TransferAmount    int
}

// ============================================================
// Platform fee
// ============================================================

// PlatformFeeInput contains the seller-level amount breakdown used to determine
// the AMOL platform fee.
//
// Seller may represent either a primary-sale Account or resale Avatar payout
// identity.
//
// The Settlement calculator intentionally does not hard-code a fee rate or fee
// base.
//
// For example:
//
//   - AMOL marketplace sale:
//     a percentage fee may apply
//
//   - external/self EC connection:
//     platform fee may be zero
//
// The application layer should select the appropriate PlatformFeeCalculator
// for the applicable sales policy.
type PlatformFeeInput struct {
	Seller SellerIdentity

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	ShippingAmount    int
	ShippingTaxAmount int

	GrossAmount int
}

type PlatformFeeCalculator interface {
	CalculatePlatformFee(
		ctx context.Context,
		in PlatformFeeInput,
	) (int, error)
}

// PlatformFeeBase specifies the base used by
// PercentagePlatformFeeCalculator.
type PlatformFeeBase string

const (
	// PlatformFeeBaseMerchandise applies the fee to the merchandise amount
	// only. Resale merchandise is included in this base.
	PlatformFeeBaseMerchandise PlatformFeeBase = "merchandise"

	// PlatformFeeBaseMerchandiseWithTax applies the fee to merchandise
	// including merchandise consumption tax, but excludes shipping.
	// Resale merchandise has no merchandise consumption tax.
	PlatformFeeBaseMerchandiseWithTax PlatformFeeBase = "merchandise_with_tax"

	// PlatformFeeBaseGross applies the fee to the complete seller-level gross
	// amount including merchandise, tax, shipping, and shipping tax.
	PlatformFeeBaseGross PlatformFeeBase = "gross"
)

// PercentagePlatformFeeCalculator provides a simple integer percentage policy.
//
// No default percentage is defined here. The caller must explicitly provide
// the rate and fee base.
type PercentagePlatformFeeCalculator struct {
	rate int
	base PlatformFeeBase
}

func NewPercentagePlatformFeeCalculator(
	rate int,
	base PlatformFeeBase,
) (*PercentagePlatformFeeCalculator, error) {
	if rate < 0 || rate > 100 {
		return nil, ErrInvalidPlatformFeeRate
	}
	if !isValidPlatformFeeBase(base) {
		return nil, ErrInvalidPlatformFeeBase
	}

	return &PercentagePlatformFeeCalculator{
		rate: rate,
		base: base,
	}, nil
}

func (c *PercentagePlatformFeeCalculator) CalculatePlatformFee(
	ctx context.Context,
	in PlatformFeeInput,
) (int, error) {
	if c == nil {
		return 0, ErrCalculatorPlatformFeeMissing
	}
	if c.rate < 0 || c.rate > 100 {
		return 0, ErrInvalidPlatformFeeRate
	}
	if !isValidPlatformFeeBase(c.base) {
		return 0, ErrInvalidPlatformFeeBase
	}
	if err := in.Seller.Validate(); err != nil {
		return 0, ErrCalculatorInvalidSellerSnapshot
	}

	var baseAmount int

	switch c.base {
	case PlatformFeeBaseMerchandise:
		baseAmount = in.MerchandiseAmount

	case PlatformFeeBaseMerchandiseWithTax:
		value, err := safeAddNonNegative(
			in.MerchandiseAmount,
			in.MerchandiseTaxAmount,
		)
		if err != nil {
			return 0, err
		}
		baseAmount = value

	case PlatformFeeBaseGross:
		baseAmount = in.GrossAmount

	default:
		return 0, ErrInvalidPlatformFeeBase
	}

	if baseAmount < 0 {
		return 0, ErrCalculatorInvalidPlatformFee
	}

	product, err := safeMultiplyNonNegative(baseAmount, c.rate)
	if err != nil {
		return 0, err
	}

	return product / 100, nil
}

func isValidPlatformFeeBase(base PlatformFeeBase) bool {
	switch base {
	case PlatformFeeBaseMerchandise,
		PlatformFeeBaseMerchandiseWithTax,
		PlatformFeeBaseGross:
		return true
	default:
		return false
	}
}

// ============================================================
// Calculator
// ============================================================

// Calculator calculates one Settlement Allocation per seller payout identity.
//
// Primary List sales use an Account seller identity and are aggregated by
// AccountID. Resale transactions use an Avatar seller identity and are
// aggregated by PayoutAccountID.
//
// The calculator uses Order snapshots only. Current Brand, Account, Avatar, or
// payout-account state is intentionally not resolved here because historical
// Orders must continue to settle against their fixed SellerSnapshot.
//
// Tax rounding follows the same order-level policy as
// order.CalculatePaymentAmount:
//
// - reduced-rate List merchandise is aggregated at 8%
// - standard-rate List merchandise and List shipping are aggregated at 10%
// - resale merchandise is non-taxable
// - resale shipping is zero
// - integer division truncates fractions
//
// After the canonical tax amount is calculated, tax yen are distributed to
// seller/component pairs using the largest-remainder method. Ties are resolved
// by deterministic seller key and component kind.
type Calculator struct {
	platformFeeCalculator PlatformFeeCalculator
}

func NewCalculator(
	platformFeeCalculator PlatformFeeCalculator,
) *Calculator {
	return &Calculator{
		platformFeeCalculator: platformFeeCalculator,
	}
}

func (c *Calculator) Calculate(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) ([]Allocation, error) {
	if c == nil || c.platformFeeCalculator == nil {
		return nil, ErrCalculatorPlatformFeeMissing
	}

	if err := validateCalculatorSource(order, payment); err != nil {
		return nil, err
	}

	builders, shippingBindings, resaleBindings, err :=
		buildMerchandiseAllocations(order)
	if err != nil {
		return nil, err
	}

	if err := applyShippingAllocations(
		order,
		builders,
		shippingBindings,
		resaleBindings,
	); err != nil {
		return nil, err
	}

	if err := allocateConsumptionTax(builders); err != nil {
		return nil, err
	}

	allocations, err := c.finalizeAllocations(ctx, builders)
	if err != nil {
		return nil, err
	}

	if len(allocations) == 0 {
		return nil, ErrCalculatorAllocationEmpty
	}

	total := 0

	for _, allocation := range allocations {
		total, err = safeAddNonNegative(total, allocation.GrossAmount)
		if err != nil {
			return nil, err
		}
	}

	if total != payment.Amount {
		return nil, ErrCalculatorAllocationAmountMismatch
	}

	return allocations, nil
}

// ============================================================
// Internal allocation state
// ============================================================

type allocationBuilder struct {
	Seller SellerIdentity

	MerchandiseAmount8          int
	MerchandiseAmount10         int
	MerchandiseAmountNonTaxable int

	ShippingAmount int

	MerchandiseTaxAmount int
	ShippingTaxAmount    int
}

type shippingKey struct {
	ListID      string
	InventoryID string
	ModelID     string
}

type shippingBinding struct {
	SellerKey string
	Qty       int
}

func buildMerchandiseAllocations(
	order orderdom.Order,
) (
	map[string]*allocationBuilder,
	map[shippingKey]shippingBinding,
	map[string]int,
	error,
) {
	builders := make(map[string]*allocationBuilder)
	shippingBindings := make(map[shippingKey]shippingBinding)
	resaleBindings := make(map[string]int)

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		lineAmount, err := safeMultiplyNonNegative(item.Price, item.Qty)
		if err != nil {
			return nil, nil, nil, err
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			if item.ListID == "" ||
				item.InventoryID == "" ||
				item.ModelID == "" ||
				item.Qty <= 0 ||
				item.Price < 0 {
				return nil, nil, nil, ErrCalculatorInvalidOrder
			}

			seller, err := resolveListSellerIdentity(item.SellerSnapshot)
			if err != nil {
				return nil, nil, nil, err
			}

			key, err := calculatorSellerKey(seller)
			if err != nil {
				return nil, nil, nil, err
			}

			builder, err := getOrCreateAllocationBuilder(
				builders,
				key,
				seller,
			)
			if err != nil {
				return nil, nil, nil, err
			}

			switch item.ConsumptionTaxRate {
			case orderdom.ConsumptionTaxRateReduced:
				builder.MerchandiseAmount8, err =
					safeAddNonNegative(
						builder.MerchandiseAmount8,
						lineAmount,
					)

			case orderdom.ConsumptionTaxRateStandard:
				builder.MerchandiseAmount10, err =
					safeAddNonNegative(
						builder.MerchandiseAmount10,
						lineAmount,
					)

			default:
				return nil, nil, nil, ErrCalculatorInvalidTaxRate
			}

			if err != nil {
				return nil, nil, nil, err
			}

			shippingItemKey := shippingKey{
				ListID:      item.ListID,
				InventoryID: item.InventoryID,
				ModelID:     item.ModelID,
			}

			binding, exists := shippingBindings[shippingItemKey]
			if exists && binding.SellerKey != key {
				return nil, nil, nil, ErrCalculatorSellerMismatch
			}

			nextQty, err := safeAddNonNegative(binding.Qty, item.Qty)
			if err != nil {
				return nil, nil, nil, err
			}

			shippingBindings[shippingItemKey] = shippingBinding{
				SellerKey: key,
				Qty:       nextQty,
			}

		case orderdom.OrderItemTypeResale:
			if item.ResaleID == "" ||
				item.ProductID == "" ||
				item.ProductBlueprintID == "" ||
				item.TokenBlueprintID == "" ||
				item.BrandID == "" ||
				item.Qty != 1 ||
				item.Price < 0 {
				return nil, nil, nil, ErrCalculatorInvalidOrder
			}

			seller, err := resolveResaleSellerIdentity(item.SellerSnapshot)
			if err != nil {
				return nil, nil, nil, err
			}

			key, err := calculatorSellerKey(seller)
			if err != nil {
				return nil, nil, nil, err
			}

			builder, err := getOrCreateAllocationBuilder(
				builders,
				key,
				seller,
			)
			if err != nil {
				return nil, nil, nil, err
			}

			builder.MerchandiseAmountNonTaxable, err =
				safeAddNonNegative(
					builder.MerchandiseAmountNonTaxable,
					lineAmount,
				)
			if err != nil {
				return nil, nil, nil, err
			}

			if resaleBindings[item.ResaleID] != 0 {
				return nil, nil, nil, ErrCalculatorInvalidOrder
			}

			resaleBindings[item.ResaleID] = 1

		default:
			return nil, nil, nil, ErrCalculatorUnsupportedOrderItem
		}
	}

	if len(builders) == 0 {
		return nil, nil, nil, ErrCalculatorAllocationEmpty
	}

	return builders, shippingBindings, resaleBindings, nil
}

func resolveListSellerIdentity(
	seller orderdom.SellerSnapshot,
) (SellerIdentity, error) {
	if seller.BrandID == "" ||
		seller.CompanyID == "" ||
		seller.AccountID == "" ||
		seller.AvatarID != "" ||
		seller.UserID != "" ||
		seller.PayoutAccountID != "" {
		return SellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	resolved := SellerIdentity{
		Type:            SellerTypeAccount,
		CompanyID:       seller.CompanyID,
		AccountID:       seller.AccountID,
		StripeAccountID: seller.StripeAccountID,
	}

	if err := resolved.Validate(); err != nil {
		return SellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	return resolved, nil
}

func resolveResaleSellerIdentity(
	seller orderdom.SellerSnapshot,
) (SellerIdentity, error) {
	if seller.AvatarID == "" ||
		seller.UserID == "" ||
		seller.PayoutAccountID == "" ||
		seller.PayoutAccountID != seller.UserID ||
		seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" {
		return SellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	resolved := SellerIdentity{
		Type:            SellerTypeAvatar,
		AvatarID:        seller.AvatarID,
		UserID:          seller.UserID,
		PayoutAccountID: seller.PayoutAccountID,
		StripeAccountID: seller.StripeAccountID,
	}

	if err := resolved.Validate(); err != nil {
		return SellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	return resolved, nil
}

func calculatorSellerKey(
	seller SellerIdentity,
) (string, error) {
	if err := seller.Validate(); err != nil {
		return "", ErrCalculatorInvalidSellerSnapshot
	}

	key, err := seller.Key()
	if err != nil {
		return "", ErrCalculatorInvalidSellerSnapshot
	}

	return string(seller.Type) + ":" + key, nil
}

func getOrCreateAllocationBuilder(
	builders map[string]*allocationBuilder,
	key string,
	seller SellerIdentity,
) (*allocationBuilder, error) {
	if key == "" {
		return nil, ErrCalculatorInvalidSellerSnapshot
	}

	builder, exists := builders[key]
	if !exists {
		builder = &allocationBuilder{
			Seller: seller,
		}
		builders[key] = builder
		return builder, nil
	}

	if builder == nil || builder.Seller != seller {
		return nil, ErrCalculatorSellerMismatch
	}

	return builder, nil
}

func applyShippingAllocations(
	order orderdom.Order,
	builders map[string]*allocationBuilder,
	bindings map[shippingKey]shippingBinding,
	resaleBindings map[string]int,
) error {
	snapshot := order.ShippingQuoteSnapshot

	if snapshot.Currency != orderdom.ShippingQuoteCurrencyJPY {
		return ErrCalculatorInvalidShippingQuote
	}
	if len(snapshot.Items) == 0 {
		return ErrCalculatorInvalidShippingQuote
	}

	quotedQty := make(map[shippingKey]int, len(bindings))
	quotedResales := make(map[string]int, len(resaleBindings))
	shippingTotal := 0

	for _, item := range snapshot.Items {
		switch item.Type {
		case "", orderdom.OrderItemTypeList:
			if item.ListID == "" ||
				item.InventoryID == "" ||
				item.ModelID == "" ||
				item.ResaleID != "" ||
				item.OriginShippingAddressID == "" ||
				item.DestinationShippingAddressID == "" ||
				item.Qty <= 0 ||
				item.UnitAmount < 0 ||
				item.Amount < 0 ||
				item.Currency != orderdom.ShippingQuoteCurrencyJPY {
				return ErrCalculatorInvalidShippingQuote
			}

			expectedAmount, err :=
				safeMultiplyNonNegative(item.UnitAmount, item.Qty)
			if err != nil {
				return err
			}
			if expectedAmount != item.Amount {
				return ErrCalculatorInvalidShippingQuote
			}

			key := shippingKey{
				ListID:      item.ListID,
				InventoryID: item.InventoryID,
				ModelID:     item.ModelID,
			}

			binding, exists := bindings[key]
			if !exists {
				return ErrCalculatorShippingQuoteMismatch
			}

			builder, exists := builders[binding.SellerKey]
			if !exists || builder == nil {
				return ErrCalculatorShippingQuoteMismatch
			}

			builder.ShippingAmount, err =
				safeAddNonNegative(
					builder.ShippingAmount,
					item.Amount,
				)
			if err != nil {
				return err
			}

			quotedQty[key], err =
				safeAddNonNegative(
					quotedQty[key],
					item.Qty,
				)
			if err != nil {
				return err
			}

			shippingTotal, err =
				safeAddNonNegative(
					shippingTotal,
					item.Amount,
				)
			if err != nil {
				return err
			}

		case orderdom.OrderItemTypeResale:
			if item.ResaleID == "" ||
				item.ListID != "" ||
				item.InventoryID != "" ||
				item.ModelID != "" ||
				item.OriginShippingAddressID != "" ||
				item.DestinationShippingAddressID == "" ||
				item.Carrier != "" ||
				item.TransportationID != "" ||
				item.Size != 0 ||
				item.Qty != 1 ||
				item.UnitAmount != 0 ||
				item.Amount != 0 ||
				item.Currency != orderdom.ShippingQuoteCurrencyJPY {
				return ErrCalculatorInvalidShippingQuote
			}

			expectedQty, exists := resaleBindings[item.ResaleID]
			if !exists || expectedQty != 1 {
				return ErrCalculatorShippingQuoteMismatch
			}

			if quotedResales[item.ResaleID] != 0 {
				return ErrCalculatorShippingQuoteMismatch
			}

			quotedResales[item.ResaleID] = 1

		default:
			return ErrCalculatorInvalidShippingQuote
		}
	}

	if shippingTotal != snapshot.Amount {
		return ErrCalculatorShippingQuoteMismatch
	}

	for key, binding := range bindings {
		if quotedQty[key] != binding.Qty {
			return ErrCalculatorShippingQuoteMismatch
		}
	}

	for resaleID, expectedQty := range resaleBindings {
		if quotedResales[resaleID] != expectedQty {
			return ErrCalculatorShippingQuoteMismatch
		}
	}

	return nil
}

// ============================================================
// Tax allocation
// ============================================================

type taxComponentKind string

const (
	taxComponentMerchandise8  taxComponentKind = "merchandise_8"
	taxComponentMerchandise10 taxComponentKind = "merchandise_10"
	taxComponentShipping10    taxComponentKind = "shipping_10"
)

type taxComponent struct {
	SellerKey string
	Kind      taxComponentKind
	Base      int
	Tax       int
	Remainder int
}

func allocateConsumptionTax(
	builders map[string]*allocationBuilder,
) error {
	reducedComponents := make(
		[]taxComponent,
		0,
		len(builders),
	)

	standardComponents := make(
		[]taxComponent,
		0,
		len(builders)*2,
	)

	for sellerKey, builder := range builders {
		if builder == nil {
			return ErrCalculatorInvalidOrder
		}

		if builder.MerchandiseAmount8 > 0 {
			reducedComponents = append(
				reducedComponents,
				taxComponent{
					SellerKey: sellerKey,
					Kind:      taxComponentMerchandise8,
					Base:      builder.MerchandiseAmount8,
				},
			)
		}

		if builder.MerchandiseAmount10 > 0 {
			standardComponents = append(
				standardComponents,
				taxComponent{
					SellerKey: sellerKey,
					Kind:      taxComponentMerchandise10,
					Base:      builder.MerchandiseAmount10,
				},
			)
		}

		if builder.ShippingAmount > 0 {
			standardComponents = append(
				standardComponents,
				taxComponent{
					SellerKey: sellerKey,
					Kind:      taxComponentShipping10,
					Base:      builder.ShippingAmount,
				},
			)
		}
	}

	reducedComponents, err :=
		allocateTaxByRate(
			reducedComponents,
			orderdom.ConsumptionTaxRateReduced,
		)
	if err != nil {
		return err
	}

	standardComponents, err =
		allocateTaxByRate(
			standardComponents,
			orderdom.ConsumptionTaxRateStandard,
		)
	if err != nil {
		return err
	}

	for _, component := range reducedComponents {
		builder := builders[component.SellerKey]
		if builder == nil {
			return ErrCalculatorInvalidOrder
		}

		builder.MerchandiseTaxAmount, err =
			safeAddNonNegative(
				builder.MerchandiseTaxAmount,
				component.Tax,
			)
		if err != nil {
			return err
		}
	}

	for _, component := range standardComponents {
		builder := builders[component.SellerKey]
		if builder == nil {
			return ErrCalculatorInvalidOrder
		}

		switch component.Kind {
		case taxComponentMerchandise10:
			builder.MerchandiseTaxAmount, err =
				safeAddNonNegative(
					builder.MerchandiseTaxAmount,
					component.Tax,
				)

		case taxComponentShipping10:
			builder.ShippingTaxAmount, err =
				safeAddNonNegative(
					builder.ShippingTaxAmount,
					component.Tax,
				)

		default:
			return ErrCalculatorInvalidTaxRate
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func allocateTaxByRate(
	components []taxComponent,
	rate int,
) ([]taxComponent, error) {
	if rate <= 0 {
		return nil, ErrCalculatorInvalidTaxRate
	}
	if len(components) == 0 {
		return components, nil
	}

	totalBase := 0
	allocatedTax := 0

	for index := range components {
		component := &components[index]

		if component.Base < 0 {
			return nil, ErrCalculatorInvalidOrder
		}

		var err error

		totalBase, err =
			safeAddNonNegative(
				totalBase,
				component.Base,
			)
		if err != nil {
			return nil, err
		}

		product, err :=
			safeMultiplyNonNegative(
				component.Base,
				rate,
			)
		if err != nil {
			return nil, err
		}

		component.Tax = product / 100
		component.Remainder = product % 100

		allocatedTax, err =
			safeAddNonNegative(
				allocatedTax,
				component.Tax,
			)
		if err != nil {
			return nil, err
		}
	}

	totalProduct, err :=
		safeMultiplyNonNegative(
			totalBase,
			rate,
		)
	if err != nil {
		return nil, err
	}

	canonicalTax := totalProduct / 100
	residual := canonicalTax - allocatedTax

	if residual < 0 || residual > len(components) {
		return nil, ErrCalculatorAllocationAmountMismatch
	}

	sort.SliceStable(
		components,
		func(i, j int) bool {
			if components[i].Remainder != components[j].Remainder {
				return components[i].Remainder > components[j].Remainder
			}
			if components[i].SellerKey != components[j].SellerKey {
				return components[i].SellerKey < components[j].SellerKey
			}
			return components[i].Kind < components[j].Kind
		},
	)

	for index := 0; index < residual; index++ {
		components[index].Tax++
	}

	return components, nil
}

// ============================================================
// Finalization
// ============================================================

func (c *Calculator) finalizeAllocations(
	ctx context.Context,
	builders map[string]*allocationBuilder,
) ([]Allocation, error) {
	sellerKeys := make(
		[]string,
		0,
		len(builders),
	)

	for sellerKey := range builders {
		sellerKeys = append(sellerKeys, sellerKey)
	}

	sort.Strings(sellerKeys)

	result := make(
		[]Allocation,
		0,
		len(sellerKeys),
	)

	for _, sellerKey := range sellerKeys {
		builder := builders[sellerKey]
		if builder == nil {
			return nil, ErrCalculatorInvalidOrder
		}
		if err := builder.Seller.Validate(); err != nil {
			return nil, ErrCalculatorInvalidSellerSnapshot
		}

		taxableMerchandiseAmount, err :=
			safeAddNonNegative(
				builder.MerchandiseAmount8,
				builder.MerchandiseAmount10,
			)
		if err != nil {
			return nil, err
		}

		merchandiseAmount, err :=
			safeAddNonNegative(
				taxableMerchandiseAmount,
				builder.MerchandiseAmountNonTaxable,
			)
		if err != nil {
			return nil, err
		}

		grossAmount, err :=
			safeAddNonNegative(
				merchandiseAmount,
				builder.MerchandiseTaxAmount,
			)
		if err != nil {
			return nil, err
		}

		grossAmount, err =
			safeAddNonNegative(
				grossAmount,
				builder.ShippingAmount,
			)
		if err != nil {
			return nil, err
		}

		grossAmount, err =
			safeAddNonNegative(
				grossAmount,
				builder.ShippingTaxAmount,
			)
		if err != nil {
			return nil, err
		}

		if grossAmount <= 0 {
			return nil, ErrCalculatorAllocationEmpty
		}

		platformFeeAmount, err :=
			c.platformFeeCalculator.CalculatePlatformFee(
				ctx,
				PlatformFeeInput{
					Seller: builder.Seller,

					MerchandiseAmount:    merchandiseAmount,
					MerchandiseTaxAmount: builder.MerchandiseTaxAmount,

					ShippingAmount:    builder.ShippingAmount,
					ShippingTaxAmount: builder.ShippingTaxAmount,

					GrossAmount: grossAmount,
				},
			)
		if err != nil {
			return nil, err
		}

		if platformFeeAmount < 0 ||
			platformFeeAmount >= grossAmount {
			return nil, ErrCalculatorInvalidPlatformFee
		}

		transferAmount :=
			grossAmount -
				platformFeeAmount

		if transferAmount <= 0 {
			return nil, ErrCalculatorInvalidPlatformFee
		}

		result = append(
			result,
			Allocation{
				Seller: builder.Seller,

				MerchandiseAmount:    merchandiseAmount,
				MerchandiseTaxAmount: builder.MerchandiseTaxAmount,

				ShippingAmount:    builder.ShippingAmount,
				ShippingTaxAmount: builder.ShippingTaxAmount,

				GrossAmount:       grossAmount,
				PlatformFeeAmount: platformFeeAmount,
				TransferAmount:    transferAmount,
			},
		)
	}

	return result, nil
}

// ============================================================
// Source validation
// ============================================================

func validateCalculatorSource(
	order orderdom.Order,
	payment paymentdom.Payment,
) error {
	if order.ID == "" ||
		len(order.Items) == 0 {
		return ErrCalculatorInvalidOrder
	}

	if payment.PaymentID == "" ||
		payment.Amount <= 0 {
		return ErrCalculatorInvalidPayment
	}

	if payment.PaymentID != order.ID {
		return ErrCalculatorPaymentOrderMismatch
	}

	canonicalAmount, err :=
		orderdom.CalculatePaymentAmount(order)
	if err != nil {
		return ErrCalculatorInvalidOrder
	}

	if canonicalAmount != payment.Amount {
		return ErrCalculatorPaymentAmountMismatch
	}

	return nil
}

// ============================================================
// Safe integer helpers
// ============================================================

func safeAddNonNegative(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrCalculatorInvalidOrder
	}

	maxInt := int(^uint(0) >> 1)

	if left > maxInt-right {
		return 0, ErrCalculatorAmountOverflow
	}

	return left + right, nil
}

func safeMultiplyNonNegative(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrCalculatorInvalidOrder
	}

	if left == 0 || right == 0 {
		return 0, nil
	}

	maxInt := int(^uint(0) >> 1)

	if left > maxInt/right {
		return 0, ErrCalculatorAmountOverflow
	}

	return left * right, nil
}
