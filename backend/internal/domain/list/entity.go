// backend/internal/domain/list/entity.go
package list

import (
	"errors"
	"regexp"
	"strings"
	"time"
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
// - ReadableID: human-friendly id (NOT required to be unique)
// - InventoryID: inventory document id (ex: productBlueprintId__tokenBlueprintId)
// - Prices: array (ONLY)
type List struct {
	ID         string     `json:"id,omitempty"`
	ReadableID string     `json:"readableId,omitempty"`
	Status     ListStatus `json:"status,omitempty"`

	AssigneeID string `json:"assigneeId,omitempty"`
	Title      string `json:"title,omitempty"`

	// ✅ 1 inventory can have multiple lists (A/B test)
	InventoryID string `json:"inventoryId,omitempty"`

	// ✅ Policy: List.ImageID stores "image URL on bucket" (NOT image entity id)
	// Optional at create; can be set later via SetPrimaryImageURL
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

// ✅ NEW: GetID makes List satisfy interfaces like `interface{ GetID() string }`
// used by generic usecase helpers (e.g. ListUsecase.Update).
// NOTE: value receiver is important so `List` (not only `*List`) satisfies it.
func (l List) GetID() string {
	return strings.TrimSpace(l.ID)
}

// Errors
var (
	// For persisted entity
	ErrInvalidID = errors.New("list: invalid id")

	ErrInvalidReadableID   = errors.New("list: invalid readableId")
	ErrInvalidStatus       = errors.New("list: invalid status")
	ErrInvalidAssigneeID   = errors.New("list: invalid assigneeId")
	ErrInvalidTitle        = errors.New("list: invalid title")
	ErrInvalidInventoryID  = errors.New("list: invalid inventoryId")
	ErrInvalidDescription  = errors.New("list: invalid description")
	ErrInvalidPrices       = errors.New("list: invalid prices")
	ErrInvalidPrice        = errors.New("list: invalid price")
	ErrInvalidPriceModelID = errors.New("list: invalid modelId in prices")

	ErrInvalidCreatedBy = errors.New("list: invalid createdBy")
	ErrInvalidCreatedAt = errors.New("list: invalid createdAt")

	ErrInvalidUpdatedAt = errors.New("list: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("list: invalid updatedBy")
	ErrInvalidDeletedAt = errors.New("list: invalid deletedAt")
	ErrInvalidDeletedBy = errors.New("list: invalid deletedBy")

	// Image linkage errors (now "URL")
	ErrEmptyImageID   = errors.New("list: imageId must not be empty")
	ErrInvalidImageID = errors.New("list: invalid imageId (url)")
)

// Policy (align with listConstants.ts as needed)
var (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000

	// ✅ human-friendly id guard
	MaxReadableIDLength = 64

	// ✅ URL length guard (practical limit)
	MaxImageURLLength = 2048
)

// =====================
// Constructors
// =====================

// NewForCreate creates a List for Create flow.
// - ID can be empty (server generates)
// - CreatedAt can be zero (repo fills)
// - ReadableID can be empty (set later)
// - ImageID(URL) can be empty (set later)
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
		ReadableID:  "", // optional; can be set later
		Status:      status,
		AssigneeID:  strings.TrimSpace(assigneeID),
		Title:       strings.TrimSpace(title),
		InventoryID: strings.TrimSpace(inventoryID),
		ImageID:     "", // optional at create
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

// UpdateReadableID sets human-friendly id.
// - NOT required to be unique
// - empty is allowed (means "unset")
func (l *List) UpdateReadableID(readableID string, now time.Time) error {
	if l == nil {
		return nil
	}
	rid := strings.TrimSpace(readableID)
	if rid == "" {
		// allow clearing
		l.ReadableID = ""
		l.touch(now)
		return nil
	}
	if !isValidReadableID(rid) {
		return ErrInvalidReadableID
	}
	l.ReadableID = rid
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

// SetPrimaryImageURL sets List.ImageID as "image URL".
// - persisted list only (ID required) is recommended, but URL自体はセットできる
func (l *List) SetPrimaryImageURL(imageURL string, now time.Time) error {
	if l == nil {
		return nil
	}
	u := strings.TrimSpace(imageURL)
	if u == "" {
		return ErrEmptyImageID
	}
	if !isValidImageURL(u) {
		return ErrInvalidImageID
	}
	// persisted 前提で運用するなら、ここを必須にしてもよい
	if strings.TrimSpace(l.ID) == "" {
		return ErrInvalidID
	}
	l.ImageID = u
	l.touch(now)
	return nil
}

// ValidateImageLink checks only "if ImageID is set, it's a valid URL".
func (l List) ValidateImageLink() error {
	u := strings.TrimSpace(l.ImageID)
	if u == "" {
		return ErrEmptyImageID
	}
	if !isValidImageURL(u) {
		return ErrInvalidImageID
	}
	return nil
}

// =====================
// Validation
// =====================

// ValidateForCreate validates fields required at Create time.
// - ID can be empty
// - CreatedAt can be zero (repo fills)
// - ReadableID can be empty (set later)
// - ImageID can be empty (set later)
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
	if strings.TrimSpace(l.ReadableID) != "" {
		if !isValidReadableID(strings.TrimSpace(l.ReadableID)) {
			return ErrInvalidReadableID
		}
	}

	// Optional fields
	if strings.TrimSpace(l.ImageID) != "" {
		if !isValidImageURL(strings.TrimSpace(l.ImageID)) {
			return ErrInvalidImageID
		}
	}

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

	// Optional but if set must be valid (NOT unique)
	if strings.TrimSpace(l.ReadableID) != "" {
		if !isValidReadableID(strings.TrimSpace(l.ReadableID)) {
			return ErrInvalidReadableID
		}
	}

	// Optional but if set must be valid
	if strings.TrimSpace(l.ImageID) != "" {
		if !isValidImageURL(strings.TrimSpace(l.ImageID)) {
			return ErrInvalidImageID
		}
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

// isValidImageURL validates bucket URL.
// Accept:
// - https://... (public URL or signed URL)
// - http://...  (dev)
// - gs://...    (GCS URL)
func isValidImageURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	if len(u) > MaxImageURLLength {
		return false
	}
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "gs://") {
		return true
	}
	return false
}

var readableIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// isValidReadableID validates a human-friendly id (NOT unique).
// Example allowed:
// - "LIST-001"
// - "ab_test-A"
// - "spring_sale_2026"
func isValidReadableID(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if len(s) > MaxReadableIDLength {
		return false
	}
	return readableIDRe.MatchString(s)
}
