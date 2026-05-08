// backend/internal/domain/list/entity.go
package list

import (
	"strings"
	"time"
)

// ListPriceRow is the ONLY supported JSON shape from frontend.
// prices: [{ modelId: string, price: number }, ...]
type ListPriceRow struct {
	ModelID string `json:"modelId"`
	Price   int    `json:"price"` // JPY
}

// List mirrors requested shape.
//
// Firebase Storage 移行後の画像方針:
// - frontend が Firebase Storage へ直接 upload する
// - backend は signed URL / GCS bucket / GCS object を扱わない
// - /lists/{listId}/images/{imageId} の Firestore record に downloadURL / objectPath を保存する
// - List.ImageID は primary imageId、つまり /lists/{listId}/images/{imageId} の docID
// - Image URL は query 層で list image record から組み立てる
type List struct {
	ID         string     `json:"id,omitempty"`
	ReadableID string     `json:"readableId,omitempty"`
	Status     ListStatus `json:"status,omitempty"`

	AssigneeID string `json:"assigneeId,omitempty"`
	Title      string `json:"title,omitempty"`

	// 1 inventory can have multiple lists (A/B test)
	InventoryID string `json:"inventoryId,omitempty"`

	// Primary image ID (= /lists/{listId}/images/{imageId} docID)
	// json tag remains "imageId" for compatibility with existing frontend DTO shape.
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

// GetID makes List satisfy interfaces like interface{ GetID() string }.
func (l List) GetID() string {
	return l.ID
}

// NewForCreate creates a List for Create flow.
// - ID can be empty because repository generates it.
// - CreatedAt can be zero because repository fills it.
// - ReadableID can be empty.
// - ImageID can be empty because image can be attached later.
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
		ReadableID:  "",
		Status:      status,
		AssigneeID:  strings.TrimSpace(assigneeID),
		Title:       strings.TrimSpace(title),
		InventoryID: strings.TrimSpace(inventoryID),
		ImageID:     "",
		Description: strings.TrimSpace(description),
		Prices:      normalizePriceRows(prices),
		CreatedBy:   strings.TrimSpace(createdBy),
		CreatedAt:   time.Time{},
	}

	if err := l.ValidateForCreate(); err != nil {
		return List{}, err
	}

	return l, nil
}

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
// - empty is allowed and means unset
func (l *List) UpdateReadableID(readableID string, now time.Time) error {
	if l == nil {
		return nil
	}

	rid := strings.TrimSpace(readableID)
	if rid == "" {
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

// SetPrimaryImageID sets List.ImageID as primary imageId.
// - empty is NOT allowed here. Use ClearPrimaryImageID to unset.
func (l *List) SetPrimaryImageID(imageID string, now time.Time) error {
	if l == nil {
		return nil
	}

	id := strings.TrimSpace(imageID)
	if id == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(id) {
		return ErrInvalidImageID
	}

	if l.ID == "" {
		return ErrInvalidID
	}

	l.ImageID = id
	l.touch(now)
	return nil
}

// ClearPrimaryImageID unsets primary image id.
func (l *List) ClearPrimaryImageID(now time.Time) error {
	if l == nil {
		return nil
	}

	if l.ID == "" {
		return ErrInvalidID
	}

	l.ImageID = ""
	l.touch(now)
	return nil
}

// ValidateImageLink checks only if ImageID is set and valid.
func (l List) ValidateImageLink() error {
	id := strings.TrimSpace(l.ImageID)
	if id == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(id) {
		return ErrInvalidImageID
	}

	return nil
}

// ValidateForCreate validates fields required at Create time.
// - ID can be empty.
// - CreatedAt can be zero.
// - ReadableID can be empty.
// - ImageID can be empty.
func (l List) ValidateForCreate() error {
	if l.Status == "" {
		// allow default
	} else if !IsValidStatus(l.Status) {
		return ErrInvalidStatus
	}

	if l.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}

	if l.Title == "" || len(l.Title) > MaxTitleLength {
		return ErrInvalidTitle
	}

	if l.InventoryID == "" {
		return ErrInvalidInventoryID
	}

	if l.Description == "" || len(l.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}

	if err := validatePriceRows(l.Prices); err != nil {
		return err
	}

	if l.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if l.ReadableID != "" && !isValidReadableID(l.ReadableID) {
		return ErrInvalidReadableID
	}

	if l.ImageID != "" && !isValidImageID(l.ImageID) {
		return ErrInvalidImageID
	}

	if l.UpdatedAt != nil && (l.UpdatedAt.IsZero() || (!l.CreatedAt.IsZero() && l.UpdatedAt.Before(l.CreatedAt))) {
		return ErrInvalidUpdatedAt
	}

	if l.UpdatedBy != nil && *l.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	if l.DeletedAt != nil && (!l.CreatedAt.IsZero() && l.DeletedAt.Before(l.CreatedAt)) {
		return ErrInvalidDeletedAt
	}

	if l.DeletedBy != nil && *l.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}

	return nil
}

// ValidateForPersist validates a fully persisted List.
func (l List) ValidateForPersist() error {
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

	if l.InventoryID == "" {
		return ErrInvalidInventoryID
	}

	if l.Description == "" || len(l.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}

	if err := validatePriceRows(l.Prices); err != nil {
		return err
	}

	if l.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if l.ReadableID != "" && !isValidReadableID(l.ReadableID) {
		return ErrInvalidReadableID
	}

	if l.ImageID != "" && !isValidImageID(l.ImageID) {
		return ErrInvalidImageID
	}

	if l.UpdatedAt != nil && (l.UpdatedAt.IsZero() || l.UpdatedAt.Before(l.CreatedAt)) {
		return ErrInvalidUpdatedAt
	}

	if l.UpdatedBy != nil && *l.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	if l.DeletedAt != nil && l.DeletedAt.Before(l.CreatedAt) {
		return ErrInvalidDeletedAt
	}

	if l.DeletedBy != nil && *l.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}

	return nil
}
