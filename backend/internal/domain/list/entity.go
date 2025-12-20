// backend/internal/domain/list/entity.go
package list

import (
	"errors"
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

// ListPriceRow is the ONLY supported JSON shape from frontend.
// prices: [{ modelId: string, price: number }, ...]
type ListPriceRow struct {
	ModelID string `json:"modelId"`
	Price   int    `json:"price"` // JPY
}

// List mirrors requested shape.
// - ID: list document id (server-generated on Create; client may omit)
// - InventoryID: inventory document id (ex: productBlueprintId__tokenBlueprintId)
// - Prices: array (ONLY)
type List struct {
	ID     string     `json:"id,omitempty"`
	Status ListStatus `json:"status,omitempty"`

	AssigneeID string `json:"assigneeId,omitempty"`
	Title      string `json:"title,omitempty"`

	// ✅ 1 inventory can have multiple lists (A/B test)
	InventoryID string `json:"inventoryId,omitempty"`

	// Optional at create; can be set later via SetPrimaryImage
	ImageID string `json:"imageId,omitempty"`

	Description string         `json:"description,omitempty"`
	Prices      []ListPriceRow `json:"prices,omitempty"`

	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`

	UpdatedBy *string    `json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// Errors
var (
	// For persisted entity
	ErrInvalidID = errors.New("list: invalid id")

	ErrInvalidStatus      = errors.New("list: invalid status")
	ErrInvalidAssigneeID  = errors.New("list: invalid assigneeId")
	ErrInvalidTitle       = errors.New("list: invalid title")
	ErrInvalidInventoryID = errors.New("list: invalid inventoryId")
	ErrInvalidDescription = errors.New("list: invalid description")

	ErrInvalidPrices       = errors.New("list: invalid prices")
	ErrInvalidPrice        = errors.New("list: invalid price")
	ErrInvalidPriceModelID = errors.New("list: invalid modelId in prices")

	ErrInvalidCreatedBy = errors.New("list: invalid createdBy")
	ErrInvalidCreatedAt = errors.New("list: invalid createdAt")

	ErrInvalidUpdatedAt = errors.New("list: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("list: invalid updatedBy")
	ErrInvalidDeletedAt = errors.New("list: invalid deletedAt")
	ErrInvalidDeletedBy = errors.New("list: invalid deletedBy")

	// Image linkage errors
	ErrEmptyImageID            = errors.New("list: imageId must not be empty")
	ErrImageBelongsToOtherList = errors.New("list: image belongs to another list")
)

// Policy (align with listConstants.ts as needed)
var (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000
)

// =====================
// Constructors
// =====================

// NewForCreate creates a List for Create flow.
// - ID can be empty (server generates)
// - CreatedAt can be zero (repo fills)
// - ImageID can be empty (set later)
func NewForCreate(
	status ListStatus,
	assigneeID string,
	title string,
	inventoryID string,
	description string,
	prices []ListPriceRow,
	createdBy string,
) (List, error) {
	if status == "" {
		status = StatusListing
	}
	l := List{
		ID:          "",
		Status:      status,
		AssigneeID:  strings.TrimSpace(assigneeID),
		Title:       strings.TrimSpace(title),
		InventoryID: strings.TrimSpace(inventoryID),
		ImageID:     "",
		Description: strings.TrimSpace(description),
		Prices:      normalizePriceRows(prices),
		CreatedBy:   strings.TrimSpace(createdBy),
		CreatedAt:   time.Time{}, // repo fills
	}
	if err := l.ValidateForCreate(); err != nil {
		return List{}, err
	}
	return l, nil
}

// =====================
// Behaviors
// =====================

func (l *List) UpdateTitle(title string, now time.Time) error {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > MaxTitleLength {
		return ErrInvalidTitle
	}
	l.Title = title
	l.touch(now)
	return nil
}

func (l *List) UpdateInventoryID(inventoryID string, now time.Time) error {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ErrInvalidInventoryID
	}
	l.InventoryID = inventoryID
	l.touch(now)
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

func (l *List) ReplacePrices(prices []ListPriceRow, now time.Time) error {
	np := normalizePriceRows(prices)
	if err := validatePriceRows(np); err != nil {
		return err
	}
	l.Prices = np
	l.touch(now)
	return nil
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

// SetPrimaryImage sets List.ImageID.
// - img.ID -> List.ImageID
// - img.ListID must match List.ID (persisted list only)
func (l *List) SetPrimaryImage(img listimagedom.ListImage) error {
	if l == nil {
		return nil
	}
	id := strings.TrimSpace(img.ID)
	if id == "" {
		return ErrEmptyImageID
	}
	// ListID整合性は persisted 前提（IDが空ならチェック不能なので許容しない）
	if strings.TrimSpace(l.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(img.ListID) != strings.TrimSpace(l.ID) {
		return ErrImageBelongsToOtherList
	}
	l.ImageID = id
	return nil
}

// ValidateImageLink checks only "if ImageID is set, it's non-empty trimmed".
// Existence check is upper layer responsibility.
func (l List) ValidateImageLink() error {
	if strings.TrimSpace(l.ImageID) == "" {
		return ErrEmptyImageID
	}
	return nil
}

// =====================
// Validation
// =====================

// ValidateForCreate validates fields required at Create time.
// - ID can be empty
// - CreatedAt can be zero (repo fills)
// - ImageID can be empty
func (l List) ValidateForCreate() error {
	if l.Status == "" {
		// allow default
	} else if !IsValidStatus(l.Status) {
		return ErrInvalidStatus
	}

	if strings.TrimSpace(l.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}
	if strings.TrimSpace(l.Title) == "" || len(l.Title) > MaxTitleLength {
		return ErrInvalidTitle
	}
	if strings.TrimSpace(l.InventoryID) == "" {
		return ErrInvalidInventoryID
	}
	if strings.TrimSpace(l.Description) == "" || len(l.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	if err := validatePriceRows(l.Prices); err != nil {
		return err
	}
	if strings.TrimSpace(l.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	// Optional fields
	if l.UpdatedAt != nil && (l.UpdatedAt.IsZero() || (!l.CreatedAt.IsZero() && l.UpdatedAt.Before(l.CreatedAt))) {
		return ErrInvalidUpdatedAt
	}
	if l.UpdatedBy != nil && strings.TrimSpace(*l.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if l.DeletedAt != nil && (!l.CreatedAt.IsZero() && l.DeletedAt.Before(l.CreatedAt)) {
		return ErrInvalidDeletedAt
	}
	if l.DeletedBy != nil && strings.TrimSpace(*l.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// ValidateForPersist validates a fully persisted List.
// - ID required
// - CreatedAt required
func (l List) ValidateForPersist() error {
	if strings.TrimSpace(l.ID) == "" {
		return ErrInvalidID
	}
	if !IsValidStatus(l.Status) {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(l.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}
	if strings.TrimSpace(l.Title) == "" || len(l.Title) > MaxTitleLength {
		return ErrInvalidTitle
	}
	if strings.TrimSpace(l.InventoryID) == "" {
		return ErrInvalidInventoryID
	}
	if strings.TrimSpace(l.Description) == "" || len(l.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	if err := validatePriceRows(l.Prices); err != nil {
		return err
	}
	if strings.TrimSpace(l.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

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

func validatePriceRows(rows []ListPriceRow) error {
	if rows == nil {
		// allow empty
		return nil
	}
	for _, r := range rows {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			return ErrInvalidPriceModelID
		}
		if !priceAllowed(r.Price) {
			return ErrInvalidPrice
		}
	}
	return nil
}

func priceAllowed(v int) bool {
	return v >= MinPrice && v <= MaxPrice
}

// =====================
// Helpers
// =====================

func (l *List) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t := now.UTC()
	l.UpdatedAt = &t
}

func normalizePriceRows(in []ListPriceRow) []ListPriceRow {
	if in == nil {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]ListPriceRow, 0, len(in))

	for _, v := range in {
		mid := strings.TrimSpace(v.ModelID)
		if mid == "" {
			continue
		}
		if !priceAllowed(v.Price) {
			continue
		}
		// dedupe by modelId (keep first)
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		out = append(out, ListPriceRow{ModelID: mid, Price: v.Price})
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
