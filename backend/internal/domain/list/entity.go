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

// ListPrice mirrors TS (price per inventoryId)
type ListPrice struct {
	Price int // JPY
}

// List mirrors requested shape
// - Prices: map[inventoryId]ListPrice
// - Title: listing title
type List struct {
	ID         string
	Status     ListStatus
	AssigneeID string
	Title      string

	ImageID     string
	Description string

	Prices map[string]ListPrice // key = inventoryId

	CreatedBy string
	CreatedAt time.Time

	UpdatedBy *string
	UpdatedAt *time.Time
	DeletedAt *time.Time
	DeletedBy *string
}

// Errors
var (
	ErrInvalidID          = errors.New("list: invalid id")
	ErrInvalidStatus      = errors.New("list: invalid status")
	ErrInvalidAssigneeID  = errors.New("list: invalid assigneeId")
	ErrInvalidTitle       = errors.New("list: invalid title")
	ErrInvalidImageID     = errors.New("list: invalid imageId")
	ErrInvalidDescription = errors.New("list: invalid description")

	ErrInvalidPrices           = errors.New("list: invalid prices")
	ErrInvalidPrice            = errors.New("list: invalid price")
	ErrInvalidPriceInventoryID = errors.New("list: invalid inventoryId in prices")

	ErrInvalidCreatedBy = errors.New("list: invalid createdBy")
	ErrInvalidCreatedAt = errors.New("list: invalid createdAt")

	ErrInvalidUpdatedAt = errors.New("list: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("list: invalid updatedBy")
	ErrInvalidDeletedAt = errors.New("list: invalid deletedAt")
	ErrInvalidDeletedBy = errors.New("list: invalid deletedBy")

	// ImageID が空のとき
	ErrEmptyImageID = errors.New("list: imageId must not be empty")
	// ListImage の listId が List.ID と一致しないとき
	ErrImageBelongsToOtherList = errors.New("list: image belongs to another list")
)

// Policy (align with listConstants.ts as needed)
var (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000
)

// Constructors

// New creates a List with required fields only. Optional updated*/deleted* are nil.
// imageId refers to ListImage.id
func New(
	id string,
	status ListStatus,
	assigneeID string,
	title string,
	imageID string,
	description string,
	prices map[string]ListPrice,
	createdBy string,
	createdAt time.Time,
) (List, error) {
	if status == "" {
		status = StatusListing
	}

	l := List{
		ID:         strings.TrimSpace(id),
		Status:     status,
		AssigneeID: strings.TrimSpace(assigneeID),
		Title:      strings.TrimSpace(title),

		ImageID:     strings.TrimSpace(imageID),
		Description: strings.TrimSpace(description),

		Prices: normalizePrices(prices),

		CreatedBy: strings.TrimSpace(createdBy),
		CreatedAt: createdAt.UTC(),

		UpdatedBy: nil,
		UpdatedAt: nil,
		DeletedAt: nil,
		DeletedBy: nil,
	}

	if err := l.validate(); err != nil {
		return List{}, err
	}
	return l, nil
}

func NewFromStringTime(
	id string,
	status ListStatus,
	assigneeID string,
	title string,
	imageID string,
	description string,
	prices map[string]ListPrice,
	createdBy string,
	createdAt string,
) (List, error) {
	t, err := parseTime(createdAt)
	if err != nil {
		return List{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	return New(id, status, assigneeID, title, imageID, description, prices, createdBy, t)
}

// Behavior

func (l *List) UpdateTitle(title string, now time.Time) error {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > MaxTitleLength {
		return ErrInvalidTitle
	}
	l.Title = title
	l.touch(now)
	return nil
}

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

func (l *List) ReplacePrices(prices map[string]ListPrice, now time.Time) error {
	np := normalizePrices(prices)
	if err := validatePrices(np); err != nil {
		return err
	}
	l.Prices = np
	l.touch(now)
	return nil
}

// SetPrice sets price by inventoryId.
func (l *List) SetPrice(inventoryID string, price int, now time.Time) error {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ErrInvalidPriceInventoryID
	}
	if !priceAllowed(price) {
		return ErrInvalidPrice
	}
	if l.Prices == nil {
		l.Prices = make(map[string]ListPrice, 1)
	}
	l.Prices[inventoryID] = ListPrice{Price: price}
	l.touch(now)
	return nil
}

func (l *List) RemovePrice(inventoryID string, now time.Time) {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" || l.Prices == nil {
		return
	}
	delete(l.Prices, inventoryID)
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
	if !IsValidStatus(l.Status) {
		return ErrInvalidStatus
	}
	if l.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}
	if l.Title == "" || len(l.Title) > MaxTitleLength {
		return ErrInvalidTitle
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
	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
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

func validatePrices(prices map[string]ListPrice) error {
	if prices == nil {
		// allow empty map / nil (必要ならここを「必須」に変更)
		return nil
	}
	for inventoryID, p := range prices {
		if strings.TrimSpace(inventoryID) == "" {
			return ErrInvalidPriceInventoryID
		}
		if !priceAllowed(p.Price) {
			return ErrInvalidPrice
		}
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

func normalizePrices(in map[string]ListPrice) map[string]ListPrice {
	if in == nil {
		return nil
	}
	out := make(map[string]ListPrice, len(in))
	for k, v := range in {
		id := strings.TrimSpace(k)
		if id == "" {
			continue
		}
		if !priceAllowed(v.Price) {
			continue
		}
		out[id] = ListPrice{Price: v.Price}
	}
	return out
}
