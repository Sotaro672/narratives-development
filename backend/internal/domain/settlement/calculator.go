// backend/internal/domain/settlement/calculator.go
package settlement

import (
	"context"
	"errors"
	"sort"
	"strings"

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

// Allocation represents the amount attributable to one seller Account.
//
// Multiple Brands using the same AccountID are aggregated into one Allocation.
//
// GrossAmount:
//
//	MerchandiseAmount
//	+ MerchandiseTaxAmount
//	+ ShippingAmount
//	+ ShippingTaxAmount
//
// TransferAmount:
//
//	GrossAmount - PlatformFeeAmount
type Allocation struct {
	CompanyID string
	AccountID string

	StripeAccountID string

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

// PlatformFeeInput contains the Account-level amount breakdown used to
// determine the AMOL platform fee.
//
// The Settlement calculator intentionally does not hard-code a fee rate or
// fee base.
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
	CompanyID string
	AccountID string

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
	// PlatformFeeBaseMerchandise applies the fee to the tax-exclusive
	// merchandise amount only.
	PlatformFeeBaseMerchandise PlatformFeeBase = "merchandise"

	// PlatformFeeBaseMerchandiseWithTax applies the fee to merchandise
	// including merchandise consumption tax, but excludes shipping.
	PlatformFeeBaseMerchandiseWithTax PlatformFeeBase = "merchandise_with_tax"

	// PlatformFeeBaseGross applies the fee to the complete Account-level gross
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

	product, err := safeMultiplyNonNegative(
		baseAmount,
		c.rate,
	)
	if err != nil {
		return 0, err
	}

	return product / 100, nil
}

func isValidPlatformFeeBase(
	base PlatformFeeBase,
) bool {
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

// Calculator calculates one Settlement Allocation per AccountID.
//
// The calculator uses Order snapshots only. Current Brand or Account state is
// intentionally not resolved here because historical Orders must continue to
// settle against their fixed SellerSnapshot.
//
// Tax rounding follows the same order-level policy as
// order.CalculatePaymentAmount:
//
// - reduced-rate merchandise is aggregated at 8%
// - standard-rate merchandise and shipping are aggregated at 10%
// - integer division truncates fractions
//
// After the canonical tax amount is calculated, tax yen are distributed to
// Account/component pairs using the largest-remainder method. Ties are resolved
// by AccountID and component kind so the result is deterministic.
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

	if err := validateCalculatorSource(
		order,
		payment,
	); err != nil {
		return nil, err
	}

	builders, shippingBindings, err :=
		buildMerchandiseAllocations(order)
	if err != nil {
		return nil, err
	}

	if err := applyShippingAllocations(
		order,
		builders,
		shippingBindings,
	); err != nil {
		return nil, err
	}

	if err := allocateConsumptionTax(
		builders,
	); err != nil {
		return nil, err
	}

	allocations, err := c.finalizeAllocations(
		ctx,
		builders,
	)
	if err != nil {
		return nil, err
	}

	if len(allocations) == 0 {
		return nil, ErrCalculatorAllocationEmpty
	}

	total := 0

	for _, allocation := range allocations {
		total, err = safeAddNonNegative(
			total,
			allocation.GrossAmount,
		)
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
	CompanyID string
	AccountID string

	StripeAccountID string

	MerchandiseAmount8  int
	MerchandiseAmount10 int

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
	AccountID string
	Qty       int
}

func buildMerchandiseAllocations(
	order orderdom.Order,
) (
	map[string]*allocationBuilder,
	map[shippingKey]shippingBinding,
	error,
) {
	builders := make(
		map[string]*allocationBuilder,
	)

	shippingBindings := make(
		map[shippingKey]shippingBinding,
	)

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		if item.Type != orderdom.OrderItemTypeList {
			return nil, nil,
				ErrCalculatorUnsupportedOrderItem
		}

		seller := item.SellerSnapshot

		if seller.BrandID == "" ||
			seller.CompanyID == "" ||
			seller.AccountID == "" ||
			seller.StripeAccountID == "" ||
			!strings.HasPrefix(
				seller.StripeAccountID,
				"acct_",
			) {
			return nil, nil,
				ErrCalculatorInvalidSellerSnapshot
		}

		if item.ListID == "" ||
			item.InventoryID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 ||
			item.Price < 0 {
			return nil, nil,
				ErrCalculatorInvalidOrder
		}

		lineAmount, err := safeMultiplyNonNegative(
			item.Price,
			item.Qty,
		)
		if err != nil {
			return nil, nil, err
		}

		builder, exists :=
			builders[seller.AccountID]

		if !exists {
			builder = &allocationBuilder{
				CompanyID:       seller.CompanyID,
				AccountID:       seller.AccountID,
				StripeAccountID: seller.StripeAccountID,
			}

			builders[seller.AccountID] =
				builder
		} else {
			if builder.CompanyID != seller.CompanyID ||
				builder.StripeAccountID != seller.StripeAccountID {
				return nil, nil,
					ErrCalculatorSellerMismatch
			}
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
			return nil, nil,
				ErrCalculatorInvalidTaxRate
		}

		if err != nil {
			return nil, nil, err
		}

		key := shippingKey{
			ListID:      item.ListID,
			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,
		}

		binding, exists :=
			shippingBindings[key]

		if exists &&
			binding.AccountID != seller.AccountID {
			return nil, nil,
				ErrCalculatorSellerMismatch
		}

		nextQty, err := safeAddNonNegative(
			binding.Qty,
			item.Qty,
		)
		if err != nil {
			return nil, nil, err
		}

		shippingBindings[key] = shippingBinding{
			AccountID: seller.AccountID,
			Qty:       nextQty,
		}
	}

	if len(builders) == 0 {
		return nil, nil,
			ErrCalculatorAllocationEmpty
	}

	return builders,
		shippingBindings,
		nil
}

func applyShippingAllocations(
	order orderdom.Order,
	builders map[string]*allocationBuilder,
	bindings map[shippingKey]shippingBinding,
) error {
	snapshot :=
		order.ShippingQuoteSnapshot

	if snapshot.Currency !=
		orderdom.ShippingQuoteCurrencyJPY {
		return ErrCalculatorInvalidShippingQuote
	}

	if len(snapshot.Items) == 0 {
		return ErrCalculatorInvalidShippingQuote
	}

	quotedQty := make(
		map[shippingKey]int,
		len(bindings),
	)

	shippingTotal := 0

	for _, item := range snapshot.Items {
		if item.ListID == "" ||
			item.InventoryID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 ||
			item.UnitAmount < 0 ||
			item.Amount < 0 ||
			item.Currency !=
				orderdom.ShippingQuoteCurrencyJPY {
			return ErrCalculatorInvalidShippingQuote
		}

		expectedAmount, err :=
			safeMultiplyNonNegative(
				item.UnitAmount,
				item.Qty,
			)
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

		builder, exists :=
			builders[binding.AccountID]
		if !exists {
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
	}

	if shippingTotal != snapshot.Amount {
		return ErrCalculatorShippingQuoteMismatch
	}

	for key, binding := range bindings {
		if quotedQty[key] != binding.Qty {
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
	taxComponentMerchandise8 taxComponentKind = "merchandise_8"

	taxComponentMerchandise10 taxComponentKind = "merchandise_10"

	taxComponentShipping10 taxComponentKind = "shipping_10"
)

type taxComponent struct {
	AccountID string
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

	for accountID, builder := range builders {
		if builder.MerchandiseAmount8 > 0 {
			reducedComponents = append(
				reducedComponents,
				taxComponent{
					AccountID: accountID,
					Kind:      taxComponentMerchandise8,
					Base:      builder.MerchandiseAmount8,
				},
			)
		}

		if builder.MerchandiseAmount10 > 0 {
			standardComponents = append(
				standardComponents,
				taxComponent{
					AccountID: accountID,
					Kind:      taxComponentMerchandise10,
					Base:      builder.MerchandiseAmount10,
				},
			)
		}

		if builder.ShippingAmount > 0 {
			standardComponents = append(
				standardComponents,
				taxComponent{
					AccountID: accountID,
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
		builder := builders[component.AccountID]
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
		builder := builders[component.AccountID]
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

		component.Tax =
			product / 100

		component.Remainder =
			product % 100

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

	canonicalTax :=
		totalProduct / 100

	residual :=
		canonicalTax -
			allocatedTax

	if residual < 0 ||
		residual > len(components) {
		return nil, ErrCalculatorAllocationAmountMismatch
	}

	sort.SliceStable(
		components,
		func(i, j int) bool {
			if components[i].Remainder !=
				components[j].Remainder {
				return components[i].Remainder >
					components[j].Remainder
			}

			if components[i].AccountID !=
				components[j].AccountID {
				return components[i].AccountID <
					components[j].AccountID
			}

			return components[i].Kind <
				components[j].Kind
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
	accountIDs := make(
		[]string,
		0,
		len(builders),
	)

	for accountID := range builders {
		accountIDs = append(
			accountIDs,
			accountID,
		)
	}

	sort.Strings(accountIDs)

	result := make(
		[]Allocation,
		0,
		len(accountIDs),
	)

	for _, accountID := range accountIDs {
		builder := builders[accountID]
		if builder == nil {
			return nil, ErrCalculatorInvalidOrder
		}

		merchandiseAmount, err :=
			safeAddNonNegative(
				builder.MerchandiseAmount8,
				builder.MerchandiseAmount10,
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
					CompanyID: builder.CompanyID,
					AccountID: builder.AccountID,

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
				CompanyID: builder.CompanyID,
				AccountID: builder.AccountID,

				StripeAccountID: builder.StripeAccountID,

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
		orderdom.CalculatePaymentAmount(
			order,
		)
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
