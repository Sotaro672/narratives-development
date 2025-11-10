// backend\internal\domain\sale\entity.go
package sale

import (
	"errors"
	"regexp"
	"strings"
)

// ========================================
// Types (mirror TS)
// ========================================

type SalePrice struct {
	ModelNumber string
	Price       int // JPY
}

type Sale struct {
	ID         string
	ListID     string
	DiscountID *string
	Prices     []SalePrice
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID          = errors.New("sale: invalid id")
	ErrInvalidListID      = errors.New("sale: invalid listId")
	ErrInvalidDiscountID  = errors.New("sale: invalid discountId")
	ErrInvalidPrices      = errors.New("sale: invalid prices")
	ErrInvalidModelNumber = errors.New("sale: invalid modelNumber")
	ErrInvalidPrice       = errors.New("sale: invalid price")
)

// ========================================
// Policy (align with saleConstants.ts)
// ========================================

var (
	// Price bounds (0 disables upper bound)
	MinPrice = 0
	MaxPrice = 10_000_000

	// At least one price is required
	MinPricesRequired = 1

	// Model number format (nil disables check)
	ModelNumberRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

// ========================================
// Constructors
// ========================================

func New(
	id, listID string,
	discountID *string,
	prices []SalePrice,
) (Sale, error) {
	s := Sale{
		ID:         strings.TrimSpace(id),
		ListID:     strings.TrimSpace(listID),
		DiscountID: normalizePtr(discountID),
		Prices:     aggregatePrices(prices),
	}
	if err := s.validate(); err != nil {
		return Sale{}, err
	}
	return s, nil
}

// ========================================
// Behavior (mutators)
// ========================================

func (s *Sale) UpdateListID(listID string) error {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return ErrInvalidListID
	}
	s.ListID = listID
	return nil
}

func (s *Sale) SetDiscountID(discountID string) error {
	discountID = strings.TrimSpace(discountID)
	if discountID == "" {
		return ErrInvalidDiscountID
	}
	s.DiscountID = &discountID
	return nil
}

func (s *Sale) ClearDiscountID() {
	s.DiscountID = nil
}

func (s *Sale) ReplacePrices(prices []SalePrice) error {
	agg := aggregatePrices(prices)
	if err := validatePrices(agg); err != nil {
		return err
	}
	s.Prices = agg
	return nil
}

func (s *Sale) SetPrice(modelNumber string, price int) error {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" || (ModelNumberRe != nil && !ModelNumberRe.MatchString(modelNumber)) {
		return ErrInvalidModelNumber
	}
	if !priceAllowed(price) {
		return ErrInvalidPrice
	}
	found := false
	for i := range s.Prices {
		if s.Prices[i].ModelNumber == modelNumber {
			s.Prices[i].Price = price
			found = true
			break
		}
	}
	if !found {
		s.Prices = append(s.Prices, SalePrice{ModelNumber: modelNumber, Price: price})
	}
	s.Prices = aggregatePrices(s.Prices)
	return nil
}

func (s *Sale) RemovePrice(modelNumber string) {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" || len(s.Prices) == 0 {
		return
	}
	out := s.Prices[:0]
	for _, p := range s.Prices {
		if p.ModelNumber != modelNumber {
			out = append(out, p)
		}
	}
	s.Prices = out
}

// ========================================
// Validation
// ========================================

func (s Sale) validate() error {
	if s.ID == "" {
		return ErrInvalidID
	}
	if s.ListID == "" {
		return ErrInvalidListID
	}
	if s.DiscountID != nil && strings.TrimSpace(*s.DiscountID) == "" {
		return ErrInvalidDiscountID
	}
	if err := validatePrices(s.Prices); err != nil {
		return err
	}
	return nil
}

func validatePrices(prices []SalePrice) error {
	if len(prices) < MinPricesRequired {
		return ErrInvalidPrices
	}
	seen := make(map[string]struct{}, len(prices))
	for _, p := range prices {
		mn := strings.TrimSpace(p.ModelNumber)
		if mn == "" || (ModelNumberRe != nil && !ModelNumberRe.MatchString(mn)) {
			return ErrInvalidModelNumber
		}
		if !priceAllowed(p.Price) {
			return ErrInvalidPrice
		}
		if _, ok := seen[mn]; ok {
			return ErrInvalidPrices
		}
		seen[mn] = struct{}{}
	}
	return nil
}

// ========================================
// Helpers
// ========================================

func priceAllowed(v int) bool {
	if v < MinPrice {
		return false
	}
	if MaxPrice > 0 && v > MaxPrice {
		return false
	}
	return true
}

func aggregatePrices(prices []SalePrice) []SalePrice {
	// last write wins by modelNumber
	tmp := make(map[string]int, len(prices))
	order := make([]string, 0, len(prices))
	for _, p := range prices {
		mn := strings.TrimSpace(p.ModelNumber)
		if mn == "" {
			continue
		}
		if _, ok := tmp[mn]; !ok {
			order = append(order, mn)
		}
		if priceAllowed(p.Price) {
			tmp[mn] = p.Price
		}
	}
	out := make([]SalePrice, 0, len(tmp))
	for _, mn := range order {
		out = append(out, SalePrice{ModelNumber: mn, Price: tmp[mn]})
	}
	return out
}

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

// SalesTableDDL defines the SQL for the sales table migration.
const SalesTableDDL = `
-- Migration: Initialize Sale domain
-- Mirrors backend/internal/domain/sale/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS sales (
  id          TEXT  PRIMARY KEY,
  list_id     TEXT  NOT NULL,
  discount_id TEXT,
  prices      JSONB NOT NULL DEFAULT '[]'::jsonb,

  -- Non-empty checks
  CONSTRAINT chk_sales_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(list_id)) > 0
  ),

  -- prices must be a JSON array
  CONSTRAINT chk_sales_prices_array CHECK (jsonb_typeof(prices) = 'array')
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sales_list_id ON sales(list_id);

COMMIT;
`
