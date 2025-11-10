// backend\internal\domain\list\entity.go
package list

import (
	"errors"
	"fmt"
	"strings"
	"time"

	listimagedom "narratives/internal/domain/listImage"
)

// ListStatus mirrors TS: 'listing' | 'suspended' | 'deleted'
type ListStatus string

const (
	StatusListing   ListStatus = "listing"
	StatusSuspended ListStatus = "suspended"
	StatusDeleted   ListStatus = "deleted"
)

func IsValidStatus(s ListStatus) bool {
	switch s {
	case StatusListing, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// ListPrice mirrors TS
type ListPrice struct {
	ModelNumber string
	Price       int // JPY
}

// List mirrors TS (dates are time.Time; updated*/deleted* are optional)
type List struct {
	ID          string
	InventoryID string
	Status      ListStatus
	AssigneeID  string
	ImageID     string
	Description string
	Prices      []ListPrice
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedBy   *string
	UpdatedAt   *time.Time
	DeletedAt   *time.Time
	DeletedBy   *string
}

// Errors
var (
	ErrInvalidID          = errors.New("list: invalid id")
	ErrInvalidInventoryID = errors.New("list: invalid inventoryId")
	ErrInvalidImageID     = errors.New("list: invalid imageId")
	ErrInvalidDescription = errors.New("list: invalid description")
	ErrInvalidPrices      = errors.New("list: invalid prices")
	ErrInvalidModelNumber = errors.New("list: invalid modelNumber")
	ErrInvalidPrice       = errors.New("list: invalid price")
	ErrInvalidStatus      = errors.New("list: invalid status")
	ErrInvalidCreatedBy   = errors.New("list: invalid createdBy")
	ErrInvalidCreatedAt   = errors.New("list: invalid createdAt")
	ErrInvalidAssigneeID  = errors.New("list: invalid assigneeId")
	ErrInvalidUpdatedAt   = errors.New("list: invalid updatedAt")
	ErrInvalidUpdatedBy   = errors.New("list: invalid updatedBy")
	ErrInvalidDeletedAt   = errors.New("list: invalid deletedAt")
	ErrInvalidDeletedBy   = errors.New("list: invalid deletedBy")

	// ImageID が空のとき
	ErrEmptyImageID = errors.New("list: imageId must not be empty")
	// ListImage の listId が List.ID と一致しないとき
	ErrImageBelongsToOtherList = errors.New("list: image belongs to another list")
)

// Policy (align with listConstants.ts as needed)
var (
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000
)

// Constructors

// New creates a List with required fields only. Optional updated*/deleted* are nil.
// imageId refers to ListImage.id
func New(
	id, inventoryID, imageID, description string,
	prices []ListPrice,
	createdBy string,
	createdAt time.Time,
	status ListStatus,
	assigneeID string,
) (List, error) {
	if status == "" {
		status = StatusListing
	}
	l := List{
		ID:          strings.TrimSpace(id),
		InventoryID: strings.TrimSpace(inventoryID),
		Status:      status,
		AssigneeID:  strings.TrimSpace(assigneeID),
		ImageID:     strings.TrimSpace(imageID),
		Description: strings.TrimSpace(description),
		Prices:      aggregatePrices(prices),
		CreatedBy:   strings.TrimSpace(createdBy),
		CreatedAt:   createdAt.UTC(),
		UpdatedBy:   nil,
		UpdatedAt:   nil,
		DeletedAt:   nil,
		DeletedBy:   nil,
	}
	if err := l.validate(); err != nil {
		return List{}, err
	}
	return l, nil
}

func NewFromStringTime(
	id, inventoryID, imageID, description string,
	prices []ListPrice,
	createdBy string,
	createdAt string,
	status ListStatus,
	assigneeID string,
) (List, error) {
	t, err := parseTime(createdAt)
	if err != nil {
		return List{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	return New(id, inventoryID, imageID, description, prices, createdBy, t, status, assigneeID)
}

// Behavior

func (l *List) UpdateImageID(imageID string, now time.Time) error {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return ErrInvalidImageID
	}
	l.ImageID = imageID
	l.touch(now)
	return nil
}

// SetPrimaryImage は、与えられた ListImage を代表画像として設定します。
// - img.ID（主キー）を List.ImageID に設定
// - img.ListID が List.ID と一致しない場合はエラー
func (l *List) SetPrimaryImage(img listimagedom.ListImage) error {
	if l == nil {
		return nil
	}
	id := strings.TrimSpace(img.ID)
	if id == "" {
		return ErrEmptyImageID
	}
	if strings.TrimSpace(img.ListID) != strings.TrimSpace(l.ID) {
		return ErrImageBelongsToOtherList
	}
	l.ImageID = id
	return nil
}

// ValidateImageLink は ImageID の必須性のみを判定します（存在性は上位で検証）。
func (l List) ValidateImageLink() error {
	if strings.TrimSpace(l.ImageID) == "" {
		return ErrEmptyImageID
	}
	return nil
}

func (l *List) UpdateDescription(desc string, now time.Time) error {
	desc = strings.TrimSpace(desc)
	if desc == "" || len(desc) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	l.Description = desc
	l.touch(now)
	return nil
}

func (l *List) ReplacePrices(prices []ListPrice, now time.Time) error {
	agg := aggregatePrices(prices)
	if err := validatePrices(agg); err != nil {
		return err
	}
	l.Prices = agg
	l.touch(now)
	return nil
}

func (l *List) SetPrice(modelNumber string, price int, now time.Time) error {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" {
		return ErrInvalidModelNumber
	}
	if !priceAllowed(price) {
		return ErrInvalidPrice
	}
	found := false
	for i := range l.Prices {
		if l.Prices[i].ModelNumber == modelNumber {
			l.Prices[i].Price = price
			found = true
			break
		}
	}
	if !found {
		l.Prices = append(l.Prices, ListPrice{ModelNumber: modelNumber, Price: price})
	}
	l.Prices = aggregatePrices(l.Prices)
	l.touch(now)
	return nil
}

func (l *List) RemovePrice(modelNumber string, now time.Time) {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" {
		return
	}
	out := l.Prices[:0]
	for _, p := range l.Prices {
		if p.ModelNumber != modelNumber {
			out = append(out, p)
		}
	}
	l.Prices = out
	l.touch(now)
}

func (l *List) Assign(assigneeID string, now time.Time) error {
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ErrInvalidAssigneeID
	}
	l.AssigneeID = assigneeID
	l.touch(now)
	return nil
}

func (l *List) Suspend(now time.Time) error {
	l.Status = StatusSuspended
	l.touch(now)
	return nil
}

func (l *List) Resume(now time.Time) error {
	l.Status = StatusListing
	l.touch(now)
	return nil
}

// Validation

func (l List) validate() error {
	if l.ID == "" {
		return ErrInvalidID
	}
	if l.InventoryID == "" {
		return ErrInvalidInventoryID
	}
	if l.ImageID == "" {
		return ErrInvalidImageID
	}
	if l.Description == "" || len(l.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	if err := validatePrices(l.Prices); err != nil {
		return err
	}
	if l.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if l.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}
	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if !IsValidStatus(l.Status) {
		return ErrInvalidStatus
	}
	// Optional fields validation
	if l.UpdatedAt != nil && (l.UpdatedAt.IsZero() || l.UpdatedAt.Before(l.CreatedAt)) {
		return ErrInvalidUpdatedAt
	}
	if l.UpdatedBy != nil && strings.TrimSpace(*l.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if l.DeletedAt != nil && l.DeletedAt.Before(l.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if l.DeletedBy != nil && strings.TrimSpace(*l.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

func validatePrices(prices []ListPrice) error {
	seen := make(map[string]struct{}, len(prices))
	for _, p := range prices {
		if strings.TrimSpace(p.ModelNumber) == "" {
			return ErrInvalidModelNumber
		}
		if !priceAllowed(p.Price) {
			return ErrInvalidPrice
		}
		if _, ok := seen[p.ModelNumber]; ok {
			return ErrInvalidPrices
		}
		seen[p.ModelNumber] = struct{}{}
	}
	return nil
}

func priceAllowed(v int) bool {
	return v >= MinPrice && v <= MaxPrice
}

// Helpers

// touch updates UpdatedAt, leaving UpdatedBy unchanged (nil unless set by other layer).
func (l *List) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t := now.UTC()
	l.UpdatedAt = &t
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrInvalidCreatedAt
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
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func aggregatePrices(prices []ListPrice) []ListPrice {
	// last write wins per modelNumber
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
	out := make([]ListPrice, 0, len(tmp))
	for _, mn := range order {
		out = append(out, ListPrice{ModelNumber: mn, Price: tmp[mn]})
	}
	return out
}

// ========================================
// SQL DDL
// ========================================
const ListsTableDDL = `
-- Migration: Initialize List domain
-- Mirrors backend/internal/domain/list/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS lists (
  id            TEXT        PRIMARY KEY,
  inventory_id  TEXT        NOT NULL,
  status        TEXT        NOT NULL CHECK (status IN ('listing','suspended','deleted')),
  assignee_id   TEXT        NOT NULL,
  image_id      TEXT        NOT NULL,
  description   TEXT        NOT NULL,
  created_by    TEXT        NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL,
  updated_by    TEXT        NULL,
  updated_at    TIMESTAMPTZ NULL,
  deleted_at    TIMESTAMPTZ NULL,
  deleted_by    TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_lists_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(inventory_id)) > 0
    AND char_length(trim(assignee_id)) > 0
    AND char_length(trim(image_id)) > 0
    AND char_length(trim(description)) > 0
    AND char_length(trim(created_by)) > 0
  ),

  -- Description length policy (aligns with MaxDescriptionLength)
  CONSTRAINT chk_lists_description_len CHECK (char_length(description) <= 2000),

  -- Time order
  CONSTRAINT chk_lists_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Normalized prices per modelNumber
CREATE TABLE IF NOT EXISTS list_prices (
  list_id      TEXT    NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  model_number TEXT    NOT NULL,
  price        INTEGER NOT NULL CHECK (price >= 0 AND price <= 10000000),
  PRIMARY KEY (list_id, model_number),
  CONSTRAINT chk_list_prices_model_non_empty CHECK (char_length(trim(model_number)) > 0)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_lists_inventory_id ON lists (inventory_id);
CREATE INDEX IF NOT EXISTS idx_lists_status       ON lists (status);
CREATE INDEX IF NOT EXISTS idx_lists_assignee_id  ON lists (assignee_id);
CREATE INDEX IF NOT EXISTS idx_lists_created_at   ON lists (created_at);
CREATE INDEX IF NOT EXISTS idx_lists_updated_at   ON lists (updated_at);

CREATE INDEX IF NOT EXISTS idx_list_prices_model_number ON list_prices (model_number);

COMMIT;
`
