// backend/internal/domain/settlement/calculator.go
package settlement

import (
	"context"
	"errors"
	"sort"
	"strconv"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	transportationdom "narratives/internal/domain/transportation"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrCalculatorPlatformFeeMissing       = errors.New("settlement: platform fee calculator is not configured")
	ErrCalculatorInvalidOrder             = errors.New("settlement: invalid order")
	ErrCalculatorInvalidPayment           = errors.New("settlement: invalid payment")
	ErrCalculatorPaymentOrderMismatch     = errors.New("settlement: payment does not belong to order")
	ErrCalculatorPaymentAmountMismatch    = errors.New("settlement: payment amount does not match order amount")
	ErrCalculatorUnsupportedOrderItem     = errors.New("settlement: unsupported order item")
	ErrCalculatorInvalidSellerSnapshot    = errors.New("settlement: invalid seller snapshot")
	ErrCalculatorSellerMismatch           = errors.New("settlement: seller snapshot mismatch")
	ErrCalculatorInvalidShippingQuote     = errors.New("settlement: invalid shipping quote")
	ErrCalculatorShippingQuoteMismatch    = errors.New("settlement: shipping quote does not match order items")
	ErrCalculatorInvalidTaxRate           = errors.New("settlement: invalid consumption tax rate")
	ErrCalculatorAmountOverflow           = errors.New("settlement: amount overflow")
	ErrCalculatorInvalidPlatformFee       = errors.New("settlement: invalid platform fee")
	ErrCalculatorInvalidBrandFee          = errors.New("settlement: invalid brand fee")
	ErrCalculatorInvalidBrandRevenue      = errors.New("settlement: invalid brand revenue snapshot")
	ErrCalculatorAllocationEmpty          = errors.New("settlement: allocation is empty")
	ErrCalculatorAllocationAmountMismatch = errors.New("settlement: allocation total does not match payment amount")
	ErrInvalidPlatformFeeRate             = errors.New("settlement: invalid platform fee rate")
	ErrInvalidPlatformFeeBase             = errors.New("settlement: invalid platform fee base")
)

// ============================================================
// Allocation
// ============================================================

// Allocation represents the amount attributable to one primary-sale Stripe
// Settlement identity.
//
// Primary List sales are aggregated by AccountID.
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
	Seller SellerIdentity

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	ShippingAmount    int
	ShippingTaxAmount int

	GrossAmount       int
	PlatformFeeAmount int
	TransferAmount    int
}

// ResaleSellerIdentity identifies the consumer resale seller whose proceeds are
// represented by a SalesReceivable rather than a Stripe Settlement.
//
// No StripeAccountID exists here.
type ResaleSellerIdentity struct {
	AvatarID        string
	UserID          string
	PayoutAccountID string
}

func (s ResaleSellerIdentity) Validate() error {
	if s.AvatarID == "" || s.UserID == "" || s.PayoutAccountID == "" {
		return ErrCalculatorInvalidSellerSnapshot
	}
	if s.PayoutAccountID != s.UserID {
		return ErrCalculatorInvalidSellerSnapshot
	}
	return nil
}

// BrandRevenueIdentity identifies the immutable productBlueprint Brand payout
// destination captured in the resale Order item snapshot.
type BrandRevenueIdentity struct {
	BrandID         string
	CompanyID       string
	AccountID       string
	StripeAccountID string
}

func (b BrandRevenueIdentity) Validate() error {
	if b.BrandID == "" ||
		b.CompanyID == "" ||
		b.AccountID == "" ||
		b.StripeAccountID == "" {
		return ErrCalculatorInvalidBrandRevenue
	}
	if len(b.StripeAccountID) < len("acct_") ||
		b.StripeAccountID[:len("acct_")] != "acct_" {
		return ErrCalculatorInvalidBrandRevenue
	}
	return nil
}

// ReceivableAllocation represents resale proceeds attributable to one resale
// Order item.
//
// One active resale Order item produces exactly one ReceivableAllocation and
// later exactly one SalesReceivable. Allocations are never aggregated by seller.
//
// Resale merchandise and resale shipping are non-taxable.
//
// Distribution base:
//
//	GrossAmount = MerchandiseAmount - ShippingAmount
//
// Distribution:
//
//	GrossAmount = PlatformFeeAmount + BrandFeeAmount + ReceivableAmount
//
// PlatformFeeAmount is AMOL's 5% share.
// BrandFeeAmount is the productBlueprint Brand's 5% share.
// BrandRevenue identifies the immutable Brand/Account/Stripe destination that
// receives BrandFeeAmount.
// ReceivableAmount is the remaining amount owed to the resale seller.
type ReceivableAllocation struct {
	OrderItemIndex int
	ResaleID       string
	BrandID        string

	Seller       ResaleSellerIdentity
	BrandRevenue BrandRevenueIdentity

	MerchandiseAmount int
	ShippingAmount    int

	GrossAmount       int
	PlatformFeeAmount int
	BrandFeeAmount    int
	ReceivableAmount  int
}

// Calculation contains both financial allocation types derived from one Order
// and Payment.
//
// Settlements contains primary List-sale Stripe Connect allocations only.
//
// Receivables contains consumer resale sales-receivable allocations only.
//
// Primary-sale Settlement GrossAmount plus resale MerchandiseAmount must equal
// Payment.Amount. Resale shipping is a fulfillment cost deducted from the resale
// merchandise amount and is not added to the buyer payment amount.
type Calculation struct {
	Settlements []Allocation
	Receivables []ReceivableAllocation
}

// ============================================================
// Platform fee
// ============================================================

// PlatformFeeInput contains the allocation-level amount breakdown used to
// determine the AMOL platform fee.
//
// Exactly one seller identity must be present:
//
//   - Seller for a primary List sale
//   - ResaleSeller for a consumer resale
//
// Seller identifies the primary List-sale Account seller used by Settlement
// calculations.
//
// Primary-sale fees use the configured calculator. Resale distribution uses
// fixed 5% AMOL and 5% productBlueprint Brand shares in Calculator.finalizeAllocations.
type PlatformFeeInput struct {
	Seller       SellerIdentity
	ResaleSeller ResaleSellerIdentity

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

// PlatformFeeBase specifies the base used by PercentagePlatformFeeCalculator.
type PlatformFeeBase string

const (
	// PlatformFeeBaseMerchandise applies the fee to merchandise only.
	PlatformFeeBaseMerchandise PlatformFeeBase = "merchandise"

	// PlatformFeeBaseMerchandiseWithTax applies the fee to merchandise including
	// merchandise consumption tax but excludes shipping.
	PlatformFeeBaseMerchandiseWithTax PlatformFeeBase = "merchandise_with_tax"

	// PlatformFeeBaseGross applies the fee to the complete allocation gross.
	PlatformFeeBaseGross PlatformFeeBase = "gross"
)

// PercentagePlatformFeeCalculator provides a simple integer percentage policy.
//
// No default percentage is defined here. The caller must explicitly provide the
// rate and fee base.
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
	if err := validatePlatformFeeSeller(in); err != nil {
		return 0, err
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

func validatePlatformFeeSeller(in PlatformFeeInput) error {
	hasSettlementSeller := !isZeroSellerIdentity(in.Seller)
	hasResaleSeller := !isZeroResaleSellerIdentity(in.ResaleSeller)

	if hasSettlementSeller == hasResaleSeller {
		return ErrCalculatorInvalidSellerSnapshot
	}

	if hasSettlementSeller {
		if err := in.Seller.Validate(); err != nil {
			return ErrCalculatorInvalidSellerSnapshot
		}
		return nil
	}

	if err := in.ResaleSeller.Validate(); err != nil {
		return ErrCalculatorInvalidSellerSnapshot
	}
	return nil
}

func isZeroSellerIdentity(s SellerIdentity) bool {
	return s.Type == "" &&
		s.CompanyID == "" &&
		s.AccountID == "" &&
		s.StripeAccountID == ""
}

func isZeroResaleSellerIdentity(s ResaleSellerIdentity) bool {
	return s.AvatarID == "" &&
		s.UserID == "" &&
		s.PayoutAccountID == ""
}

func isZeroBrandRevenueIdentity(b BrandRevenueIdentity) bool {
	return b.BrandID == "" &&
		b.CompanyID == "" &&
		b.AccountID == "" &&
		b.StripeAccountID == ""
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

const (
	resalePlatformFeeRate = 5
	resaleBrandFeeRate    = 5
)

// ============================================================
// Calculator
// ============================================================

// Calculator derives primary-sale Settlement allocations and resale
// SalesReceivable allocations from one authoritative Order and Payment.
//
// Primary List sales:
//   - aggregated by AccountID
//   - represented by Allocation
//   - later persisted as Settlement
//
// Resale transactions:
//   - allocated independently by Order item
//   - represented by ReceivableAllocation
//   - later persisted as one SalesReceivable per resale Order item
//
// The calculator uses Order snapshots only. Current Brand, Account, Avatar, or
// payout-account state is intentionally not resolved here.
//
// Tax rounding follows the same order-level policy as
// order.CalculatePaymentAmount:
//
//   - reduced-rate List merchandise is aggregated at 8%
//   - standard-rate List merchandise and List shipping are aggregated at 10%
//   - resale merchandise is non-taxable
//   - resale shipping is non-taxable and deducted from resale proceeds
//   - integer division truncates fractions
//
// Tax yen are distributed by the largest-remainder method. Ties are resolved by
// deterministic seller key and component kind.
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

// Calculate retains the existing SettlementCalculator contract.
//
// Only primary-sale Settlement allocations are returned.
//
// Call CalculateAll when resale SalesReceivable allocations are also required.
//
// The complete Order/Payment amount is still validated inside CalculateAll
// before this method returns.
func (c *Calculator) Calculate(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) ([]Allocation, error) {
	result, err := c.CalculateAll(ctx, order, payment)
	if err != nil {
		return nil, err
	}
	return result.Settlements, nil
}

// CalculateReceivables returns only consumer resale receivable allocations.
func (c *Calculator) CalculateReceivables(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) ([]ReceivableAllocation, error) {
	result, err := c.CalculateAll(ctx, order, payment)
	if err != nil {
		return nil, err
	}
	return result.Receivables, nil
}

// CalculateAll calculates both primary-sale Stripe Settlement allocations and
// consumer resale SalesReceivable allocations.
func (c *Calculator) CalculateAll(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) (Calculation, error) {
	if c == nil || c.platformFeeCalculator == nil {
		return Calculation{}, ErrCalculatorPlatformFeeMissing
	}

	if err := validateCalculatorSource(order, payment); err != nil {
		return Calculation{}, err
	}

	builders, shippingBindings, resaleBindings, err :=
		buildMerchandiseAllocations(order)
	if err != nil {
		return Calculation{}, err
	}

	if err := applyShippingAllocations(
		order,
		builders,
		shippingBindings,
		resaleBindings,
	); err != nil {
		return Calculation{}, err
	}

	if err := allocateConsumptionTax(builders); err != nil {
		return Calculation{}, err
	}

	result, err := c.finalizeAllocations(ctx, builders)
	if err != nil {
		return Calculation{}, err
	}

	if len(result.Settlements) == 0 && len(result.Receivables) == 0 {
		return Calculation{}, ErrCalculatorAllocationEmpty
	}

	total := 0

	for _, allocation := range result.Settlements {
		total, err = safeAddNonNegative(total, allocation.GrossAmount)
		if err != nil {
			return Calculation{}, err
		}
	}

	for _, allocation := range result.Receivables {
		total, err = safeAddNonNegative(total, allocation.MerchandiseAmount)
		if err != nil {
			return Calculation{}, err
		}
	}

	if total != payment.Amount {
		return Calculation{}, ErrCalculatorAllocationAmountMismatch
	}

	return result, nil
}

// ============================================================
// Internal allocation state
// ============================================================

type allocationSellerKind string

const (
	allocationSellerKindAccount allocationSellerKind = "account"
	allocationSellerKindResale  allocationSellerKind = "resale"
)

type allocationSeller struct {
	Kind allocationSellerKind

	SettlementSeller SellerIdentity
	ResaleSeller     ResaleSellerIdentity

	ResaleOrderItemIndex int
	ResaleID             string
	ResaleBrandID        string
	ResaleBrandRevenue   BrandRevenueIdentity
}

func (s allocationSeller) Validate() error {
	switch s.Kind {
	case allocationSellerKindAccount:
		if !isZeroResaleSellerIdentity(s.ResaleSeller) ||
			s.ResaleID != "" ||
			s.ResaleBrandID != "" ||
			!isZeroBrandRevenueIdentity(s.ResaleBrandRevenue) {
			return ErrCalculatorInvalidSellerSnapshot
		}
		if err := s.SettlementSeller.Validate(); err != nil {
			return ErrCalculatorInvalidSellerSnapshot
		}
		if s.SettlementSeller.Type != SellerTypeAccount {
			return ErrCalculatorInvalidSellerSnapshot
		}
		return nil

	case allocationSellerKindResale:
		if !isZeroSellerIdentity(s.SettlementSeller) ||
			s.ResaleOrderItemIndex < 0 ||
			s.ResaleID == "" ||
			s.ResaleBrandID == "" {
			return ErrCalculatorInvalidSellerSnapshot
		}
		if err := s.ResaleSeller.Validate(); err != nil {
			return err
		}
		if err := s.ResaleBrandRevenue.Validate(); err != nil {
			return err
		}
		if s.ResaleBrandRevenue.BrandID != s.ResaleBrandID {
			return ErrCalculatorInvalidBrandRevenue
		}
		return nil

	default:
		return ErrCalculatorInvalidSellerSnapshot
	}
}

func (s allocationSeller) Key() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}

	switch s.Kind {
	case allocationSellerKindAccount:
		key, err := s.SettlementSeller.Key()
		if err != nil {
			return "", ErrCalculatorInvalidSellerSnapshot
		}
		return string(allocationSellerKindAccount) + ":" + key, nil

	case allocationSellerKindResale:
		return string(allocationSellerKindResale) + "_item:" +
			strconv.Itoa(s.ResaleOrderItemIndex), nil

	default:
		return "", ErrCalculatorInvalidSellerSnapshot
	}
}

type allocationBuilder struct {
	Seller allocationSeller

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

type resaleShippingBinding struct {
	SellerKey string
	Qty       int
}

func buildMerchandiseAllocations(
	order orderdom.Order,
) (
	map[string]*allocationBuilder,
	map[shippingKey]shippingBinding,
	map[string]resaleShippingBinding,
	error,
) {
	builders := make(map[string]*allocationBuilder)
	shippingBindings := make(map[shippingKey]shippingBinding)
	resaleBindings := make(map[string]resaleShippingBinding)

	for itemIndex, item := range order.Items {
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

			allocationIdentity := allocationSeller{
				Kind:             allocationSellerKindAccount,
				SettlementSeller: seller,
			}

			key, err := allocationIdentity.Key()
			if err != nil {
				return nil, nil, nil, err
			}

			builder, err := getOrCreateAllocationBuilder(
				builders,
				key,
				allocationIdentity,
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
				item.Price <= 0 {
				return nil, nil, nil, ErrCalculatorInvalidOrder
			}

			seller, err := resolveResaleSellerIdentity(item.SellerSnapshot)
			if err != nil {
				return nil, nil, nil, err
			}

			brandRevenue, err := resolveBrandRevenueIdentity(
				item.BrandRevenueSnapshot,
				item.BrandID,
			)
			if err != nil {
				return nil, nil, nil, err
			}

			allocationIdentity := allocationSeller{
				Kind:                 allocationSellerKindResale,
				ResaleSeller:         seller,
				ResaleOrderItemIndex: itemIndex,
				ResaleID:             item.ResaleID,
				ResaleBrandID:        item.BrandID,
				ResaleBrandRevenue:   brandRevenue,
			}

			key, err := allocationIdentity.Key()
			if err != nil {
				return nil, nil, nil, err
			}

			builder, err := getOrCreateAllocationBuilder(
				builders,
				key,
				allocationIdentity,
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

			if _, exists := resaleBindings[item.ResaleID]; exists {
				return nil, nil, nil, ErrCalculatorInvalidOrder
			}

			resaleBindings[item.ResaleID] = resaleShippingBinding{
				SellerKey: key,
				Qty:       1,
			}

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
) (ResaleSellerIdentity, error) {
	if seller.AvatarID == "" ||
		seller.UserID == "" ||
		seller.PayoutAccountID == "" ||
		seller.PayoutAccountID != seller.UserID ||
		seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" ||
		seller.StripeAccountID != "" {
		return ResaleSellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	resolved := ResaleSellerIdentity{
		AvatarID:        seller.AvatarID,
		UserID:          seller.UserID,
		PayoutAccountID: seller.PayoutAccountID,
	}

	if err := resolved.Validate(); err != nil {
		return ResaleSellerIdentity{}, ErrCalculatorInvalidSellerSnapshot
	}

	return resolved, nil
}

func resolveBrandRevenueIdentity(
	snapshot orderdom.BrandRevenueSnapshot,
	expectedBrandID string,
) (BrandRevenueIdentity, error) {
	if expectedBrandID == "" ||
		snapshot.BrandID == "" ||
		snapshot.BrandID != expectedBrandID ||
		snapshot.CompanyID == "" ||
		snapshot.AccountID == "" ||
		snapshot.StripeAccountID == "" {
		return BrandRevenueIdentity{}, ErrCalculatorInvalidBrandRevenue
	}

	resolved := BrandRevenueIdentity{
		BrandID:         snapshot.BrandID,
		CompanyID:       snapshot.CompanyID,
		AccountID:       snapshot.AccountID,
		StripeAccountID: snapshot.StripeAccountID,
	}
	if err := resolved.Validate(); err != nil {
		return BrandRevenueIdentity{}, err
	}

	return resolved, nil
}

func getOrCreateAllocationBuilder(
	builders map[string]*allocationBuilder,
	key string,
	seller allocationSeller,
) (*allocationBuilder, error) {
	if key == "" {
		return nil, ErrCalculatorInvalidSellerSnapshot
	}
	if err := seller.Validate(); err != nil {
		return nil, err
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
	resaleBindings map[string]resaleShippingBinding,
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
			if builder.Seller.Kind != allocationSellerKindAccount {
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
				item.TransportationID != "" ||
				item.Qty != 1 ||
				item.UnitAmount <= 0 ||
				item.Amount <= 0 ||
				item.UnitAmount != item.Amount ||
				item.Currency != orderdom.ShippingQuoteCurrencyJPY {
				return ErrCalculatorInvalidShippingQuote
			}

			carrier := transportationdom.Carrier(item.Carrier)
			if !transportationdom.IsValidResaleShippingCarrier(carrier) ||
				!transportationdom.IsValidResaleBoxSize(item.Size) {
				return ErrCalculatorInvalidShippingQuote
			}

			expectedQuote, err := transportationdom.CalculateResaleFlatRate(
				carrier,
				item.Size,
			)
			if err != nil ||
				expectedQuote.Amount != int64(item.Amount) {
				return ErrCalculatorInvalidShippingQuote
			}

			binding, exists := resaleBindings[item.ResaleID]
			if !exists || binding.Qty != 1 {
				return ErrCalculatorShippingQuoteMismatch
			}
			if quotedResales[item.ResaleID] != 0 {
				return ErrCalculatorShippingQuoteMismatch
			}

			builder, exists := builders[binding.SellerKey]
			if !exists || builder == nil ||
				builder.Seller.Kind != allocationSellerKindResale {
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

			shippingTotal, err =
				safeAddNonNegative(
					shippingTotal,
					item.Amount,
				)
			if err != nil {
				return err
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

	for resaleID, binding := range resaleBindings {
		if quotedResales[resaleID] != binding.Qty {
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
			switch builder.Seller.Kind {
			case allocationSellerKindAccount:
				standardComponents = append(
					standardComponents,
					taxComponent{
						SellerKey: sellerKey,
						Kind:      taxComponentShipping10,
						Base:      builder.ShippingAmount,
					},
				)
			case allocationSellerKindResale:
				// Resale shipping is non-taxable.
			default:
				return ErrCalculatorInvalidOrder
			}
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
) (Calculation, error) {
	sellerKeys := make(
		[]string,
		0,
		len(builders),
	)

	for sellerKey := range builders {
		sellerKeys = append(sellerKeys, sellerKey)
	}

	sort.Strings(sellerKeys)

	result := Calculation{
		Settlements: make([]Allocation, 0),
		Receivables: make([]ReceivableAllocation, 0),
	}

	for _, sellerKey := range sellerKeys {
		builder := builders[sellerKey]
		if builder == nil {
			return Calculation{}, ErrCalculatorInvalidOrder
		}
		if err := builder.Seller.Validate(); err != nil {
			return Calculation{}, ErrCalculatorInvalidSellerSnapshot
		}

		taxableMerchandiseAmount, err :=
			safeAddNonNegative(
				builder.MerchandiseAmount8,
				builder.MerchandiseAmount10,
			)
		if err != nil {
			return Calculation{}, err
		}

		merchandiseAmount, err :=
			safeAddNonNegative(
				taxableMerchandiseAmount,
				builder.MerchandiseAmountNonTaxable,
			)
		if err != nil {
			return Calculation{}, err
		}

		switch builder.Seller.Kind {
		case allocationSellerKindAccount:
			if builder.MerchandiseAmountNonTaxable != 0 {
				return Calculation{}, ErrCalculatorInvalidOrder
			}

			grossAmount, err :=
				safeAddNonNegative(
					merchandiseAmount,
					builder.MerchandiseTaxAmount,
				)
			if err != nil {
				return Calculation{}, err
			}

			grossAmount, err =
				safeAddNonNegative(
					grossAmount,
					builder.ShippingAmount,
				)
			if err != nil {
				return Calculation{}, err
			}

			grossAmount, err =
				safeAddNonNegative(
					grossAmount,
					builder.ShippingTaxAmount,
				)
			if err != nil {
				return Calculation{}, err
			}

			if grossAmount <= 0 {
				return Calculation{}, ErrCalculatorAllocationEmpty
			}

			platformFeeAmount, err :=
				c.platformFeeCalculator.CalculatePlatformFee(
					ctx,
					PlatformFeeInput{
						Seller:               builder.Seller.SettlementSeller,
						MerchandiseAmount:    merchandiseAmount,
						MerchandiseTaxAmount: builder.MerchandiseTaxAmount,
						ShippingAmount:       builder.ShippingAmount,
						ShippingTaxAmount:    builder.ShippingTaxAmount,
						GrossAmount:          grossAmount,
					},
				)
			if err != nil {
				return Calculation{}, err
			}

			if platformFeeAmount < 0 ||
				platformFeeAmount >= grossAmount {
				return Calculation{}, ErrCalculatorInvalidPlatformFee
			}

			transferAmount := grossAmount - platformFeeAmount
			if transferAmount <= 0 {
				return Calculation{}, ErrCalculatorInvalidPlatformFee
			}

			result.Settlements = append(
				result.Settlements,
				Allocation{
					Seller: builder.Seller.SettlementSeller,

					MerchandiseAmount:    merchandiseAmount,
					MerchandiseTaxAmount: builder.MerchandiseTaxAmount,

					ShippingAmount:    builder.ShippingAmount,
					ShippingTaxAmount: builder.ShippingTaxAmount,

					GrossAmount:       grossAmount,
					PlatformFeeAmount: platformFeeAmount,
					TransferAmount:    transferAmount,
				},
			)

		case allocationSellerKindResale:
			if builder.MerchandiseAmount8 != 0 ||
				builder.MerchandiseAmount10 != 0 ||
				builder.MerchandiseTaxAmount != 0 ||
				builder.ShippingTaxAmount != 0 ||
				builder.ShippingAmount <= 0 {
				return Calculation{}, ErrCalculatorInvalidOrder
			}

			grossAmount, err := safeSubtractNonNegative(
				merchandiseAmount,
				builder.ShippingAmount,
			)
			if err != nil || grossAmount <= 0 {
				return Calculation{}, ErrCalculatorInvalidOrder
			}

			platformFeeProduct, err := safeMultiplyNonNegative(
				grossAmount,
				resalePlatformFeeRate,
			)
			if err != nil {
				return Calculation{}, err
			}

			brandFeeProduct, err := safeMultiplyNonNegative(
				grossAmount,
				resaleBrandFeeRate,
			)
			if err != nil {
				return Calculation{}, err
			}

			platformFeeAmount := platformFeeProduct / 100
			brandFeeAmount := brandFeeProduct / 100

			if platformFeeAmount < 0 ||
				platformFeeAmount > grossAmount {
				return Calculation{}, ErrCalculatorInvalidPlatformFee
			}
			if brandFeeAmount < 0 ||
				brandFeeAmount > grossAmount {
				return Calculation{}, ErrCalculatorInvalidBrandFee
			}

			totalFees, err := safeAddNonNegative(
				platformFeeAmount,
				brandFeeAmount,
			)
			if err != nil || totalFees >= grossAmount {
				return Calculation{}, ErrCalculatorInvalidPlatformFee
			}

			receivableAmount, err := safeSubtractNonNegative(
				grossAmount,
				totalFees,
			)
			if err != nil || receivableAmount <= 0 {
				return Calculation{}, ErrCalculatorInvalidPlatformFee
			}

			result.Receivables = append(
				result.Receivables,
				ReceivableAllocation{
					OrderItemIndex: builder.Seller.ResaleOrderItemIndex,
					ResaleID:       builder.Seller.ResaleID,
					BrandID:        builder.Seller.ResaleBrandID,
					Seller:         builder.Seller.ResaleSeller,
					BrandRevenue:   builder.Seller.ResaleBrandRevenue,

					MerchandiseAmount: merchandiseAmount,
					ShippingAmount:    builder.ShippingAmount,

					GrossAmount:       grossAmount,
					PlatformFeeAmount: platformFeeAmount,
					BrandFeeAmount:    brandFeeAmount,
					ReceivableAmount:  receivableAmount,
				},
			)

		default:
			return Calculation{}, ErrCalculatorInvalidSellerSnapshot
		}
	}

	sort.Slice(result.Receivables, func(i, j int) bool {
		if result.Receivables[i].OrderItemIndex == result.Receivables[j].OrderItemIndex {
			return result.Receivables[i].ResaleID < result.Receivables[j].ResaleID
		}
		return result.Receivables[i].OrderItemIndex < result.Receivables[j].OrderItemIndex
	})

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

func safeSubtractNonNegative(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 || right > left {
		return 0, ErrCalculatorInvalidOrder
	}

	return left - right, nil
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
