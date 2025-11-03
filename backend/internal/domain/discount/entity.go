package discount

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DiscountItem mirrors web-app/src/shared/types/discount.ts
type DiscountItem struct {
	ModelNumber string
	Discount    int // percent (0..100 by default policy)
}

// Discount mirrors web-app/src/shared/types/discount.ts
// export interface Discount {
//   id: string;              // 主キー（discount_xxx）
//   listId: string;          // 出品ID
//   discounts: DiscountItem[]; // modelNumberごとの割引率配列
//   description?: string;    // 割引の説明
//   discountedBy: string;    // 割引設定者のメンバーID
//   discountedAt: string;    // 割引設定日時（ISO文字列）
//   updatedAt: string;       // 最終更新日時（ISO文字列）
//   updatedBy: string;       // 最終更新者のメンバーID
// }
type Discount struct {
	ID           string         `json:"id"`
	ListID       string         `json:"listId"`
	Discounts    []DiscountItem `json:"discounts"`
	Description  *string        `json:"description,omitempty"`
	DiscountedBy string         `json:"discountedBy"`
	DiscountedAt time.Time      `json:"discountedAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	UpdatedBy    string         `json:"updatedBy"`
}

// Errors
var (
	ErrInvalidID           = errors.New("discount: invalid id")
	ErrInvalidListID       = errors.New("discount: invalid listId")
	ErrInvalidItems        = errors.New("discount: invalid discounts")
	ErrInvalidModelNumber  = errors.New("discount: invalid modelNumber")
	ErrInvalidDiscount     = errors.New("discount: invalid discount percent")
	ErrInvalidDescription  = errors.New("discount: invalid description")
	ErrInvalidDiscountedBy = errors.New("discount: invalid discountedBy")
	ErrInvalidDiscountedAt = errors.New("discount: invalid discountedAt")
	ErrInvalidUpdatedAt    = errors.New("discount: invalid updatedAt")
	ErrInvalidUpdatedBy    = errors.New("discount: invalid updatedBy")
)

// Policy (sync with shared/constants/discountConstants.ts as needed)
var (
	// e.g. ID like "discount_XXXX"
	DiscountIDPrefix = "discount_"
	EnforceIDPrefix  = false

	// 0..100% by default
	MinPercent = 0
	MaxPercent = 100

	// Model number format (nil disables)
	ModelNumberRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

	// Description length (0 disables)
	MaxDescriptionLength = 1000

	// Require at least one item (0 disables)
	MinItemsRequired = 1
)

// Constructors

func New(
	id, listID string,
	items []DiscountItem,
	description *string,
	discountedBy string,
	discountedAt time.Time,
	updatedAt time.Time,
	updatedBy string,
) (Discount, error) {
	d := Discount{
		ID:           strings.TrimSpace(id),
		ListID:       strings.TrimSpace(listID),
		Discounts:    aggregateItems(items),
		Description:  normalizePtr(description),
		DiscountedBy: strings.TrimSpace(discountedBy),
		DiscountedAt: discountedAt.UTC(),
		UpdatedAt:    updatedAt.UTC(),
		UpdatedBy:    strings.TrimSpace(updatedBy),
	}
	if err := d.validate(); err != nil {
		return Discount{}, err
	}
	return d, nil
}

func NewWithNow(
	id, listID string,
	items []DiscountItem,
	description *string,
	discountedBy, updatedBy string,
	now time.Time,
) (Discount, error) {
	now = now.UTC()
	return New(id, listID, items, description, discountedBy, now, now, updatedBy)
}

func NewFromStringTimes(
	id, listID string,
	items []DiscountItem,
	description *string,
	discountedBy, discountedAtStr, updatedBy, updatedAtStr string,
) (Discount, error) {
	da, err := parseTime(discountedAtStr)
	if err != nil {
		return Discount{}, fmt.Errorf("%w: %v", ErrInvalidDiscountedAt, err)
	}
	ua, err := parseTime(updatedAtStr)
	if err != nil {
		return Discount{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, listID, items, description, discountedBy, da, ua, updatedBy)
}

// Behavior

func (d *Discount) ReplaceItems(items []DiscountItem) error {
	agg := aggregateItems(items)
	if err := validateItems(agg); err != nil {
		return err
	}
	d.Discounts = agg
	return nil
}

func (d *Discount) SetItem(modelNumber string, percent int) error {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" || (ModelNumberRe != nil && !ModelNumberRe.MatchString(modelNumber)) {
		return ErrInvalidModelNumber
	}
	if !percentOK(percent) {
		return ErrInvalidDiscount
	}
	found := false
	for i := range d.Discounts {
		if d.Discounts[i].ModelNumber == modelNumber {
			d.Discounts[i].Discount = percent
			found = true
			break
		}
	}
	if !found {
		d.Discounts = append(d.Discounts, DiscountItem{ModelNumber: modelNumber, Discount: percent})
	}
	d.Discounts = aggregateItems(d.Discounts)
	return nil
}

func (d *Discount) RemoveItem(modelNumber string) {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" || len(d.Discounts) == 0 {
		return
	}
	out := d.Discounts[:0]
	for _, it := range d.Discounts {
		if it.ModelNumber != modelNumber {
			out = append(out, it)
		}
	}
	d.Discounts = out
}

func (d *Discount) UpdateMeta(listID, discountedBy string) error {
	listID = strings.TrimSpace(listID)
	discountedBy = strings.TrimSpace(discountedBy)
	if listID == "" {
		return ErrInvalidListID
	}
	if discountedBy == "" {
		return ErrInvalidDiscountedBy
	}
	d.ListID = listID
	d.DiscountedBy = discountedBy
	return nil
}

func (d *Discount) UpdateDescription(desc *string) error {
	nd := normalizePtr(desc)
	if nd != nil && MaxDescriptionLength > 0 && len([]rune(*nd)) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	d.Description = nd
	return nil
}

func (d *Discount) SetDiscountedAt(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidDiscountedAt
	}
	d.DiscountedAt = at.UTC()
	return nil
}

func (d *Discount) SetUpdated(at time.Time, by string) error {
	if at.IsZero() {
		return ErrInvalidUpdatedAt
	}
	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidUpdatedBy
	}
	d.UpdatedAt = at.UTC()
	d.UpdatedBy = by
	return nil
}

// Validation

func (d Discount) validate() error {
	if d.ID == "" {
		return ErrInvalidID
	}
	if EnforceIDPrefix && DiscountIDPrefix != "" && !strings.HasPrefix(d.ID, DiscountIDPrefix) {
		return ErrInvalidID
	}
	if d.ListID == "" {
		return ErrInvalidListID
	}
	if err := validateItems(d.Discounts); err != nil {
		return err
	}
	if d.Description != nil && MaxDescriptionLength > 0 && len([]rune(*d.Description)) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	if d.DiscountedBy == "" {
		return ErrInvalidDiscountedBy
	}
	if d.DiscountedAt.IsZero() {
		return ErrInvalidDiscountedAt
	}
	if strings.TrimSpace(d.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if d.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	// Optional temporal relation: updatedAt should not be before discountedAt
	if d.UpdatedAt.Before(d.DiscountedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Helpers

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func percentOK(v int) bool {
	if v < MinPercent {
		return false
	}
	if MaxPercent > 0 && v > MaxPercent {
		return false
	}
	return true
}

func validateItems(items []DiscountItem) error {
	if MinItemsRequired > 0 && len(items) < MinItemsRequired {
		return ErrInvalidItems
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		mn := strings.TrimSpace(it.ModelNumber)
		if mn == "" || (ModelNumberRe != nil && !ModelNumberRe.MatchString(mn)) {
			return ErrInvalidModelNumber
		}
		if !percentOK(it.Discount) {
			return ErrInvalidDiscount
		}
		if _, ok := seen[mn]; ok {
			return ErrInvalidItems
		}
		seen[mn] = struct{}{}
	}
	return nil
}

func aggregateItems(items []DiscountItem) []DiscountItem {
	// last write wins per modelNumber
	tmp := make(map[string]int, len(items))
	order := make([]string, 0, len(items))
	for _, it := range items {
		mn := strings.TrimSpace(it.ModelNumber)
		if mn == "" {
			continue
		}
		if _, ok := tmp[mn]; !ok {
			order = append(order, mn)
		}
		if percentOK(it.Discount) {
			tmp[mn] = it.Discount
		}
	}
	out := make([]DiscountItem, 0, len(tmp))
	for _, mn := range order {
		out = append(out, DiscountItem{ModelNumber: mn, Discount: tmp[mn]})
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrInvalidDiscountedAt
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", ErrInvalidDiscountedAt, s)
}

// ========================================
// SQL DDL
// ========================================

const DiscountsTableDDL = `
CREATE TABLE IF NOT EXISTS discounts (
  id TEXT PRIMARY KEY,                 -- 例: 'discount_xxx'
  list_id TEXT NOT NULL,               -- 出品ID（型はシステム都合に合わせてUUID等へ変更可）
  description TEXT NULL,               -- 割引の説明
  discounted_by TEXT NOT NULL,         -- 設定者のメンバーID（型はシステム都合に合わせてUUID等へ変更可）
  discounted_at TIMESTAMPTZ NOT NULL,  -- 設定日時
  updated_by TEXT NOT NULL,            -- 最終更新者ID
  updated_at TIMESTAMPTZ NOT NULL      -- 最終更新日時
);

-- modelNumberごとの割引率（正規化）
CREATE TABLE IF NOT EXISTS discount_items (
  discount_id TEXT NOT NULL REFERENCES discounts(id) ON DELETE CASCADE,
  model_number TEXT NOT NULL,
  percent INT NOT NULL CHECK (percent >= 0 AND percent <= 100),
  PRIMARY KEY (discount_id, model_number)
);

-- Search/Sort helpers
CREATE INDEX IF NOT EXISTS idx_discounts_list_id ON discounts (list_id);
CREATE INDEX IF NOT EXISTS idx_discounts_discounted_at ON discounts (discounted_at);
CREATE INDEX IF NOT EXISTS idx_discounts_updated_at ON discounts (updated_at);
CREATE INDEX IF NOT EXISTS idx_discount_items_model_number ON discount_items (model_number);
`
