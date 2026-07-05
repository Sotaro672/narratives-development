// backend/internal/domain/resale/entity.go
package resale

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ResaleStatus mirrors listing status used by the frontend.
type ResaleStatus string

const (
	StatusListing   ResaleStatus = "listing"
	StatusSuspended ResaleStatus = "suspended"
	StatusSold      ResaleStatus = "sold"
)

func IsValidStatus(s ResaleStatus) bool {
	switch s {
	case StatusListing, StatusSuspended, StatusSold:
		return true
	default:
		return false
	}
}

// ResaleCondition mirrors the frontend condition select values.
// Values are escaped to keep this source ASCII while preserving API payloads.
type ResaleCondition string

const (
	ConditionNewUnused ResaleCondition = "\u65b0\u54c1\u30fb\u672a\u4f7f\u7528"
	ConditionLikeNew   ResaleCondition = "\u672a\u4f7f\u7528\u306b\u8fd1\u3044"
	ConditionGood      ResaleCondition = "\u76ee\u7acb\u3063\u305f\u50b7\u3084\u6c5a\u308c\u306a\u3057"
	ConditionFair      ResaleCondition = "\u3084\u3084\u50b7\u3084\u6c5a\u308c\u3042\u308a"
	ConditionPoor      ResaleCondition = "\u50b7\u3084\u6c5a\u308c\u3042\u308a"
)

func IsValidCondition(c ResaleCondition) bool {
	switch c {
	case ConditionNewUnused, ConditionLikeNew, ConditionGood, ConditionFair, ConditionPoor:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidID                 = errors.New("resale: invalid id")
	ErrInvalidStatus             = errors.New("resale: invalid status")
	ErrInvalidMintAddress        = errors.New("resale: invalid mintAddress")
	ErrInvalidTokenBlueprintID   = errors.New("resale: invalid tokenBlueprintId")
	ErrInvalidProductID          = errors.New("resale: invalid productId")
	ErrInvalidBrandID            = errors.New("resale: invalid brandId")
	ErrInvalidProductBlueprintID = errors.New("resale: invalid productBlueprintId")
	ErrInvalidAvatarID           = errors.New("resale: invalid avatarId")
	ErrInvalidPrice              = errors.New("resale: invalid price")
	ErrInvalidCondition          = errors.New("resale: invalid condition")
	ErrInvalidDescription        = errors.New("resale: invalid description")
	ErrInvalidCreatedBy          = errors.New("resale: invalid createdBy")
	ErrInvalidCreatedAt          = errors.New("resale: invalid createdAt")
	ErrInvalidUpdatedAt          = errors.New("resale: invalid updatedAt")
	ErrInvalidUpdatedBy          = errors.New("resale: invalid updatedBy")

	ErrEmptyImageID   = errors.New("resale: imageId must not be empty")
	ErrInvalidImageID = errors.New("resale: invalid imageId")

	ErrInvalidConditionImageID           = errors.New("resale: invalid image id")
	ErrInvalidConditionImageResaleID     = errors.New("resale: invalid image resaleId")
	ErrInvalidConditionImageURL          = errors.New("resale: invalid image url")
	ErrInvalidConditionImageDisplayOrder = errors.New("resale: invalid image displayOrder")
	ErrInvalidConditionImageCreatedAt    = errors.New("resale: invalid image createdAt")
	ErrInvalidConditionImageCreatedBy    = errors.New("resale: invalid image createdBy")
	ErrInvalidConditionImageUpdatedAt    = errors.New("resale: invalid image updatedAt")
	ErrInvalidConditionImageUpdatedBy    = errors.New("resale: invalid image updatedBy")
	ErrConditionImageNotFound            = errors.New("resale: image not found")
	ErrConditionImageConflict            = errors.New("resale: image conflict")
)

var (
	MaxReferenceIDLength = 128
	MaxMintAddressLength = 128
	MaxDescriptionLength = 1000
	MinPrice             = 0
	MaxPrice             = 10_000_000
	MaxImageIDLength     = 128
)

// ResaleColor is a display-only color value resolved from model variation.
type ResaleColor struct {
	Name string `json:"name,omitempty"`
	RGB  int    `json:"rgb,omitempty"`
}

// ResaleVolume is a display-only volume value resolved from model variation.
type ResaleVolume struct {
	Amount int    `json:"amount,omitempty"`
	Unit   string `json:"unit,omitempty"`
}

// Resale is the resale listing aggregate root.
//
// Delete policy:
// - Resale deletion is physical deletion.
// - Backend does not keep DeletedAt / DeletedBy.
//
// Image policy:
// - Frontend uploads resale images before or around listing creation.
// - Backend persists only the primary image record id on Resale.ImageID.
// - Display URL is resolved from ResaleImage.URL in query / mapper / handler layer.
type Resale struct {
	ID     string       `json:"id,omitempty"`
	Status ResaleStatus `json:"status,omitempty"`

	MintAddress        string `json:"mintAddress,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
	ProductID          string `json:"productId,omitempty"`
	BrandID            string `json:"brandId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	AvatarID           string `json:"avatarId,omitempty"`

	Price       int             `json:"price,omitempty"`
	Condition   ResaleCondition `json:"condition,omitempty"`
	Description string          `json:"description,omitempty"`

	// Primary resale image record id. This is not a URL.
	ImageID string `json:"imageId,omitempty"`

	// Display-only fields resolved by mall resale query.
	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`
	TokenIcon   string `json:"tokenIcon,omitempty"`
	BrandName   string `json:"brandName,omitempty"`
	AvatarName  string `json:"avatarName,omitempty"`
	AvatarIcon  string `json:"avatarIcon,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`

	// Display-only model fields resolved from product.modelId -> model variation.
	ModelID      string         `json:"modelId,omitempty"`
	Kind         string         `json:"kind,omitempty"`
	ModelNumber  string         `json:"modelNumber,omitempty"`
	Size         string         `json:"size,omitempty"`
	Color        *ResaleColor   `json:"color,omitempty"`
	Measurements map[string]int `json:"measurements,omitempty"`
	Volume       *ResaleVolume  `json:"volume,omitempty"`

	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`

	UpdatedBy *string    `json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

func (r Resale) GetID() string {
	return r.ID
}

// NewForCreate creates a Resale for create flow.
// - ID can be empty because repository generates it.
// - CreatedAt can be zero because repository fills it.
// - ImageID can be empty because images can be attached later.
func NewForCreate(
	status ResaleStatus,
	mintAddress string,
	tokenBlueprintID string,
	productID string,
	brandID string,
	productBlueprintID string,
	avatarID string,
	price int,
	condition ResaleCondition,
	description string,
	createdBy string,
) (Resale, error) {
	if status == "" {
		status = StatusListing
	}

	r := Resale{
		ID:                 "",
		Status:             status,
		MintAddress:        mintAddress,
		TokenBlueprintID:   tokenBlueprintID,
		ProductID:          productID,
		BrandID:            brandID,
		ProductBlueprintID: productBlueprintID,
		AvatarID:           avatarID,
		Price:              price,
		Condition:          condition,
		Description:        description,
		ImageID:            "",
		ProductName:        "",
		TokenName:          "",
		TokenIcon:          "",
		BrandName:          "",
		AvatarName:         "",
		AvatarIcon:         "",
		ImageURL:           "",
		ModelID:            "",
		Kind:               "",
		ModelNumber:        "",
		Size:               "",
		Color:              nil,
		Measurements:       nil,
		Volume:             nil,
		CreatedBy:          createdBy,
		CreatedAt:          time.Time{},
	}

	if err := r.ValidateForCreate(); err != nil {
		return Resale{}, err
	}

	return r, nil
}

func (r *Resale) UpdatePrice(price int, now time.Time) error {
	if !priceAllowed(price) {
		return ErrInvalidPrice
	}

	r.Price = price
	r.touch(now)
	return nil
}

func (r *Resale) UpdateCondition(condition ResaleCondition, now time.Time) error {
	if !IsValidCondition(condition) {
		return ErrInvalidCondition
	}

	r.Condition = condition
	r.touch(now)
	return nil
}

func (r *Resale) UpdateDescription(description string, now time.Time) error {
	if len(description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}

	r.Description = description
	r.touch(now)
	return nil
}

func (r *Resale) AssignAvatar(avatarID string, now time.Time) error {
	if !isValidReferenceID(avatarID) {
		return ErrInvalidAvatarID
	}

	r.AvatarID = avatarID
	r.touch(now)
	return nil
}

func (r *Resale) Suspend(now time.Time) error {
	r.Status = StatusSuspended
	r.touch(now)
	return nil
}

func (r *Resale) Resume(now time.Time) error {
	r.Status = StatusListing
	r.touch(now)
	return nil
}

func (r *Resale) MarkSold(now time.Time) error {
	r.Status = StatusSold
	r.touch(now)
	return nil
}

// SetPrimaryImageID sets Resale.ImageID as primary resale image record id.
// Empty value is not allowed here. Use ClearPrimaryImageID to unset.
func (r *Resale) SetPrimaryImageID(imageID string, now time.Time) error {
	if r == nil {
		return nil
	}

	if imageID == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(imageID) {
		return ErrInvalidImageID
	}

	if r.ID == "" {
		return ErrInvalidID
	}

	r.ImageID = imageID
	r.touch(now)
	return nil
}

func (r *Resale) ClearPrimaryImageID(now time.Time) error {
	if r == nil {
		return nil
	}

	if r.ID == "" {
		return ErrInvalidID
	}

	r.ImageID = ""
	r.touch(now)
	return nil
}

// ValidateImageLink checks only if ImageID is set and valid.
func (r Resale) ValidateImageLink() error {
	if r.ImageID == "" {
		return ErrEmptyImageID
	}

	if !isValidImageID(r.ImageID) {
		return ErrInvalidImageID
	}

	return nil
}

// ValidateForCreate validates fields required at create time.
//   - ID can be empty.
//   - CreatedAt can be zero.
//   - ImageID can be empty.
//   - BrandID and ProductBlueprintID are optional because the frontend can submit
//     a resale listing from token context where productId and tokenBlueprintId are
//     the required listing target.
//   - Model display fields are query-only and are not validated here.
func (r Resale) ValidateForCreate() error {
	if r.Status != "" && !IsValidStatus(r.Status) {
		return ErrInvalidStatus
	}

	if !isValidMintAddress(r.MintAddress) {
		return ErrInvalidMintAddress
	}

	if !isValidReferenceID(r.TokenBlueprintID) {
		return ErrInvalidTokenBlueprintID
	}

	if !isValidReferenceID(r.ProductID) {
		return ErrInvalidProductID
	}

	if r.BrandID != "" && !isValidReferenceID(r.BrandID) {
		return ErrInvalidBrandID
	}

	if r.ProductBlueprintID != "" && !isValidReferenceID(r.ProductBlueprintID) {
		return ErrInvalidProductBlueprintID
	}

	if !isValidReferenceID(r.AvatarID) {
		return ErrInvalidAvatarID
	}

	if !priceAllowed(r.Price) {
		return ErrInvalidPrice
	}

	if !IsValidCondition(r.Condition) {
		return ErrInvalidCondition
	}

	if len(r.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}

	if r.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if r.ImageID != "" && !isValidImageID(r.ImageID) {
		return ErrInvalidImageID
	}

	if r.UpdatedAt != nil && (r.UpdatedAt.IsZero() || (!r.CreatedAt.IsZero() && r.UpdatedAt.Before(r.CreatedAt))) {
		return ErrInvalidUpdatedAt
	}

	if r.UpdatedBy != nil && *r.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	return nil
}

// ValidateForPersist validates a fully persisted Resale.
//   - Model display fields are query-only and are not validated here.
func (r Resale) ValidateForPersist() error {
	if r.ID == "" {
		return ErrInvalidID
	}

	if !IsValidStatus(r.Status) {
		return ErrInvalidStatus
	}

	if !isValidMintAddress(r.MintAddress) {
		return ErrInvalidMintAddress
	}

	if !isValidReferenceID(r.TokenBlueprintID) {
		return ErrInvalidTokenBlueprintID
	}

	if !isValidReferenceID(r.ProductID) {
		return ErrInvalidProductID
	}

	if r.BrandID != "" && !isValidReferenceID(r.BrandID) {
		return ErrInvalidBrandID
	}

	if r.ProductBlueprintID != "" && !isValidReferenceID(r.ProductBlueprintID) {
		return ErrInvalidProductBlueprintID
	}

	if !isValidReferenceID(r.AvatarID) {
		return ErrInvalidAvatarID
	}

	if !priceAllowed(r.Price) {
		return ErrInvalidPrice
	}

	if !IsValidCondition(r.Condition) {
		return ErrInvalidCondition
	}

	if len(r.Description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}

	if r.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if r.ImageID != "" && !isValidImageID(r.ImageID) {
		return ErrInvalidImageID
	}

	if r.UpdatedAt != nil && (r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt)) {
		return ErrInvalidUpdatedAt
	}

	if r.UpdatedBy != nil && *r.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	return nil
}

// ResaleImage is an image record under:
//
// /resales/{resaleId}/conditionImages/{imageId}
//
// Image policy:
// - URL is Firebase Storage getDownloadURL().
// - Backend persists only the display URL and ordering metadata.
// - Backend does not manage objectPath, fileName, contentType, or size.
type ResaleImage struct {
	ID           string     `json:"id"`
	ResaleID     string     `json:"resaleId"`
	URL          string     `json:"url"`
	DisplayOrder int        `json:"displayOrder"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    *string    `json:"updatedBy,omitempty"`
}

func NewResaleImage(
	id string,
	resaleID string,
	u string,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
) (ResaleImage, error) {
	image := ResaleImage{
		ID:           id,
		ResaleID:     resaleID,
		URL:          u,
		DisplayOrder: displayOrder,
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    createdBy,
	}

	if err := image.Validate(); err != nil {
		return ResaleImage{}, err
	}

	return image, nil
}

func (image *ResaleImage) UpdateURL(u string) error {
	if err := validateURL(u); err != nil {
		return err
	}

	image.URL = u
	return nil
}

func (image *ResaleImage) SetDisplayOrder(order int) error {
	if order < 0 {
		return ErrInvalidConditionImageDisplayOrder
	}

	image.DisplayOrder = order
	return nil
}

func (image *ResaleImage) Touch(now time.Time, actor string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	t := now.UTC()
	image.UpdatedAt = &t

	if actor != "" {
		image.UpdatedBy = &actor
	}
}

func (image ResaleImage) Validate() error {
	if image.ID == "" {
		return ErrInvalidConditionImageID
	}

	if !isValidImageID(image.ID) {
		return ErrInvalidConditionImageID
	}

	if image.ResaleID == "" {
		return ErrInvalidConditionImageResaleID
	}

	if err := validateURL(image.URL); err != nil {
		return err
	}

	if image.DisplayOrder < 0 {
		return ErrInvalidConditionImageDisplayOrder
	}

	if image.CreatedAt.IsZero() {
		return ErrInvalidConditionImageCreatedAt
	}

	if image.CreatedBy == "" {
		return ErrInvalidConditionImageCreatedBy
	}

	if image.UpdatedAt != nil && (image.UpdatedAt.IsZero() || image.UpdatedAt.Before(image.CreatedAt)) {
		return ErrInvalidConditionImageUpdatedAt
	}

	if image.UpdatedBy != nil && *image.UpdatedBy == "" {
		return ErrInvalidConditionImageUpdatedBy
	}

	return nil
}

func (r *Resale) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	t := now.UTC()
	r.UpdatedAt = &t
}

func priceAllowed(v int) bool {
	return v >= MinPrice && v <= MaxPrice
}

func isValidMintAddress(s string) bool {
	if s == "" {
		return false
	}

	if len(s) > MaxMintAddressLength {
		return false
	}

	if strings.ContainsAny(s, " \t\r\n") || strings.Contains(s, "/") || strings.Contains(s, "://") {
		return false
	}

	return true
}

func isValidReferenceID(id string) bool {
	if id == "" {
		return false
	}

	if len(id) > MaxReferenceIDLength {
		return false
	}

	if strings.Contains(id, "/") || strings.Contains(id, "://") {
		return false
	}

	return true
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
		return ErrInvalidConditionImageURL
	}

	pu, err := url.ParseRequestURI(u)
	if err != nil {
		return ErrInvalidConditionImageURL
	}

	if pu.Scheme == "" || pu.Host == "" {
		return ErrInvalidConditionImageURL
	}

	return nil
}
