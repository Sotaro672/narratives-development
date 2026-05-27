package list

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ListStatus mirrors frontend: 'listing' | 'suspended'
type ListStatus string

const (
	StatusListing   ListStatus = "listing"
	StatusSuspended ListStatus = "suspended"
)

func IsValidStatus(s ListStatus) bool {
	switch s {
	case StatusListing, StatusSuspended:
		return true
	default:
		return false
	}
}

// Errors
var (
	// List errors
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

	// Primary image linkage errors
	ErrEmptyImageID   = errors.New("list: imageId must not be empty")
	ErrInvalidImageID = errors.New("list: invalid imageId")

	// ListImage errors
	ErrInvalidListImageID           = errors.New("list: invalid listImage id")
	ErrInvalidListImageListID       = errors.New("list: invalid listImage listId")
	ErrInvalidListImageURL          = errors.New("list: invalid listImage url")
	ErrInvalidListImageDisplayOrder = errors.New("list: invalid listImage displayOrder")
	ErrInvalidListImageCreatedAt    = errors.New("list: invalid listImage createdAt")
	ErrInvalidListImageCreatedBy    = errors.New("list: invalid listImage createdBy")
	ErrInvalidListImageUpdatedAt    = errors.New("list: invalid listImage updatedAt")
	ErrInvalidListImageUpdatedBy    = errors.New("list: invalid listImage updatedBy")
	ErrListImageNotFound            = errors.New("list: listImage not found")
	ErrListImageConflict            = errors.New("list: listImage conflict")
)

// Policy
var (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000

	// human-friendly id guard
	MaxReadableIDLength = 64

	// primary image id guard
	MaxImageIDLength = 128
)

// ListPriceRow is the only supported price row shape from frontend.
//
// prices: [{ modelId: string, price: number }, ...]
type ListPriceRow struct {
	ModelID string `json:"modelId"`
	Price   int    `json:"price"`
}

// List is the list aggregate root.
//
// Delete policy:
// - List deletion is physical deletion.
// - Backend does not keep DeletedAt / DeletedBy.
// - Deleted lists are removed from Firestore.
//
// Image policy:
// - Frontend uploads list images directly to Firebase Storage.
// - Backend does not manage Storage object path, file name, content type, or size.
// - List.ImageID stores the primary image record id.
// - Display URL is resolved from ListImage.URL in query / mapper / handler layer.
type List struct {
	ID         string     `json:"id,omitempty"`
	ReadableID string     `json:"readableId,omitempty"`
	Status     ListStatus `json:"status,omitempty"`

	AssigneeID string `json:"assigneeId,omitempty"`
	Title      string `json:"title,omitempty"`

	// 1 inventory can have multiple lists.
	InventoryID string `json:"inventoryId,omitempty"`

	// Primary image record id.
	// This is not a URL.
	ImageID string `json:"imageId,omitempty"`

	Description string         `json:"description,omitempty"`
	Prices      []ListPriceRow `json:"prices,omitempty"`

	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`

	UpdatedBy *string    `json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

func (l List) GetID() string {
	return l.ID
}

// NewForCreate creates a List for create flow.
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
		AssigneeID:  assigneeID,
		Title:       title,
		InventoryID: inventoryID,
		ImageID:     "",
		Description: description,
		Prices:      normalizePriceRows(prices),
		CreatedBy:   createdBy,
		CreatedAt:   time.Time{},
	}

	if err := l.ValidateForCreate(); err != nil {
		return List{}, err
	}

	return l, nil
}

func (l *List) UpdateTitle(title string, now time.Time) error {
	if title == "" || len(title) > MaxTitleLength {
		return ErrInvalidTitle
	}

	l.Title = title
	l.touch(now)
	return nil
}

// UpdateReadableID sets human-friendly id.
// - It does not need to be globally unique.
// - Empty value is allowed and means unset.
func (l *List) UpdateReadableID(readableID string, now time.Time) error {
	if l == nil {
		return nil
	}

	if readableID == "" {
		l.ReadableID = ""
		l.touch(now)
		return nil
	}

	if !isValidReadableID(readableID) {
		return ErrInvalidReadableID
	}

	l.ReadableID = readableID
	l.touch(now)
	return nil
}

func (l *List) UpdateInventoryID(inventoryID string, now time.Time) error {
	if inventoryID == "" {
		return ErrInvalidInventoryID
	}

	l.InventoryID = inventoryID
	l.touch(now)
	return nil
}

func (l *List) UpdateDescription(desc string, now time.Time) error {
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

// SetPrimaryImageID sets List.ImageID as primary image record id.
// Empty value is not allowed here. Use ClearPrimaryImageID to unset.
func (l *List) SetPrimaryImageID(imageID string, now time.Time) error {
	if l == nil {
		return nil
	}

	if imageID == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(imageID) {
		return ErrInvalidImageID
	}

	if l.ID == "" {
		return ErrInvalidID
	}

	l.ImageID = imageID
	l.touch(now)
	return nil
}

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
	if l.ImageID == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(l.ImageID) {
		return ErrInvalidImageID
	}

	return nil
}

// ValidateForCreate validates fields required at create time.
// - ID can be empty.
// - CreatedAt can be zero.
// - ReadableID can be empty.
// - ImageID can be empty.
func (l List) ValidateForCreate() error {
	if l.Status != "" && !IsValidStatus(l.Status) {
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

	return nil
}

// ListImage is an image record under:
//
// /lists/{listId}/images/{imageId}
//
// Image policy:
// - URL is Firebase Storage getDownloadURL().
// - Backend persists only the display URL and ordering metadata.
// - Backend does not manage objectPath, fileName, contentType, or size.
type ListImage struct {
	ID           string     `json:"id"`
	ListID       string     `json:"listId"`
	URL          string     `json:"url"`
	DisplayOrder int        `json:"displayOrder"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    *string    `json:"updatedBy,omitempty"`
}

func NewListImage(
	id string,
	listID string,
	u string,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
) (ListImage, error) {
	li := ListImage{
		ID:           id,
		ListID:       listID,
		URL:          u,
		DisplayOrder: displayOrder,
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    createdBy,
	}

	if err := li.Validate(); err != nil {
		return ListImage{}, err
	}

	return li, nil
}

func (li *ListImage) UpdateURL(u string) error {
	if err := validateURL(u); err != nil {
		return err
	}

	li.URL = u
	return nil
}

func (li *ListImage) SetDisplayOrder(order int) error {
	if order < 0 {
		return ErrInvalidListImageDisplayOrder
	}

	li.DisplayOrder = order
	return nil
}

func (li *ListImage) Touch(now time.Time, actor string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	t := now.UTC()
	li.UpdatedAt = &t

	if actor != "" {
		li.UpdatedBy = &actor
	}
}

func (li ListImage) Validate() error {
	if li.ID == "" {
		return ErrInvalidListImageID
	}

	if strings.Contains(li.ID, "/") || strings.Contains(li.ID, "://") {
		return ErrInvalidListImageID
	}

	if li.ListID == "" {
		return ErrInvalidListImageListID
	}

	if err := validateURL(li.URL); err != nil {
		return err
	}

	if li.DisplayOrder < 0 {
		return ErrInvalidListImageDisplayOrder
	}

	if li.CreatedAt.IsZero() {
		return ErrInvalidListImageCreatedAt
	}

	if li.CreatedBy == "" {
		return ErrInvalidListImageCreatedBy
	}

	if li.UpdatedAt != nil && (li.UpdatedAt.IsZero() || li.UpdatedAt.Before(li.CreatedAt)) {
		return ErrInvalidListImageUpdatedAt
	}

	if li.UpdatedBy != nil && *li.UpdatedBy == "" {
		return ErrInvalidListImageUpdatedBy
	}

	return nil
}

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
		if v.ModelID == "" {
			continue
		}

		if !priceAllowed(v.Price) {
			continue
		}

		if _, ok := seen[v.ModelID]; ok {
			continue
		}

		seen[v.ModelID] = struct{}{}
		out = append(out, ListPriceRow{
			ModelID: v.ModelID,
			Price:   v.Price,
		})
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func validatePriceRows(rows []ListPriceRow) error {
	if rows == nil {
		return nil
	}

	for _, r := range rows {
		if r.ModelID == "" {
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

var readableIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func isValidReadableID(s string) bool {
	if s == "" {
		return false
	}

	if len(s) > MaxReadableIDLength {
		return false
	}

	return readableIDRe.MatchString(s)
}

var imageIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func isValidImageID(id string) bool {
	if id == "" {
		return false
	}

	if len(id) > MaxImageIDLength {
		return false
	}

	if strings.Contains(id, "/") || strings.Contains(id, "://") {
		return false
	}

	return imageIDRe.MatchString(id)
}

func validateURL(u string) error {
	if u == "" {
		return ErrInvalidListImageURL
	}

	pu, err := url.ParseRequestURI(u)
	if err != nil {
		return ErrInvalidListImageURL
	}

	if pu.Scheme == "" || pu.Host == "" {
		return ErrInvalidListImageURL
	}

	return nil
}
