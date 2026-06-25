// backend/internal/domain/inquiry/entity.go
package inquiry

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Types
type InquiryStatus string
type InquiryType string

const (
	InquiryStatusOpen     InquiryStatus = "open"
	InquiryStatusResolved InquiryStatus = "resolved"
	InquiryStatusClosed   InquiryStatus = "closed"
)

// FirebaseStorageDeleteOp represents a delete operation target in Firebase Storage.
// ObjectPath is the Firebase Storage object path, for example:
// inquiry-images/{inquiryId}/{imageId}/{fileName}
type FirebaseStorageDeleteOp struct {
	ObjectPath string
}

// ImageFile represents an image file attached to an inquiry.
//
// Firebase Storage policy:
// - frontend uploads the binary file directly to Firebase Storage.
// - backend stores only the Firebase Storage download URL and objectPath.
// - no GCS bucket/object metadata should be stored here.
type ImageFile struct {
	InquiryID  string     `json:"inquiryId"`
	FileName   string     `json:"fileName"`
	FileURL    string     `json:"fileUrl"`
	ObjectPath *string    `json:"objectPath,omitempty"`
	FileSize   int64      `json:"fileSize"`
	MimeType   string     `json:"mimeType"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  string     `json:"createdBy"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy  *string    `json:"updatedBy,omitempty"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
	DeletedBy  *string    `json:"deletedBy,omitempty"`
}

// Inquiry is the root aggregate.
//
// Inquiry is identified in the mall context by productId + avatarId.
// Images are part of the inquiry aggregate.
// The old inquiryImage domain and ImageID reference are no longer needed.
//
// Lifecycle:
// - avatar creates an inquiry as open.
// - company member can mark it as resolved.
// - company member or owner avatar can close the inquiry.
type Inquiry struct {
	ID          string        `json:"id"`
	ProductID   string        `json:"productId"`
	AvatarID    string        `json:"avatarId"`
	Subject     string        `json:"subject"`
	Content     string        `json:"content"`
	Status      InquiryStatus `json:"status"`
	InquiryType InquiryType   `json:"inquiryType"`
	IsRead      bool          `json:"isRead"`
	Images      []ImageFile   `json:"images,omitempty"`

	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy *string    `json:"resolvedBy,omitempty"`
	ClosedAt   *time.Time `json:"closedAt,omitempty"`
	ClosedBy   *string    `json:"closedBy,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// Policy
var (
	MaxImages               = 10
	MinFileSizeBytes  int64 = 1
	MaxFileSizeBytes  int64 = 20 * 1024 * 1024
	MaxFileNameLength       = 255

	AllowedMimeTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
		"image/gif":  {},
	}

	// Empty means all URL hosts are allowed.
	// If you want to restrict to Firebase Storage hosts, configure this in infrastructure/bootstrap.
	AllowedURLHosts = map[string]struct{}{}

	mimeRe = regexp.MustCompile(`^[a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+$`)
)

// Errors
var (
	ErrInvalidID          = errors.New("inquiry: invalid id")
	ErrInvalidProductID   = errors.New("inquiry: invalid productId")
	ErrInvalidAvatarID    = errors.New("inquiry: invalid avatarId")
	ErrInvalidSubject     = errors.New("inquiry: invalid subject")
	ErrInvalidContent     = errors.New("inquiry: invalid content")
	ErrInvalidStatus      = errors.New("inquiry: invalid status")
	ErrInvalidInquiryType = errors.New("inquiry: invalid inquiryType")
	ErrInvalidCreatedAt   = errors.New("inquiry: invalid createdAt")
	ErrInvalidUpdatedAt   = errors.New("inquiry: invalid updatedAt")
	ErrInvalidUpdatedBy   = errors.New("inquiry: invalid updatedBy")
	ErrInvalidDeletedAt   = errors.New("inquiry: invalid deletedAt")
	ErrInvalidDeletedBy   = errors.New("inquiry: invalid deletedBy")
	ErrInvalidResolvedAt  = errors.New("inquiry: invalid resolvedAt")
	ErrInvalidResolvedBy  = errors.New("inquiry: invalid resolvedBy")
	ErrInvalidClosedAt    = errors.New("inquiry: invalid closedAt")
	ErrInvalidClosedBy    = errors.New("inquiry: invalid closedBy")

	ErrInvalidImageInquiryID  = errors.New("inquiry: invalid image inquiryId")
	ErrInvalidImageFileName   = errors.New("inquiry: invalid image fileName")
	ErrInvalidImageFileURL    = errors.New("inquiry: invalid image fileUrl")
	ErrInvalidImageObjectPath = errors.New("inquiry: invalid image objectPath")
	ErrInvalidImageFileSize   = errors.New("inquiry: invalid image fileSize")
	ErrInvalidImageMIMEType   = errors.New("inquiry: invalid image mimeType")
	ErrInvalidImageCreatedAt  = errors.New("inquiry: invalid image createdAt")
	ErrInvalidImageCreatedBy  = errors.New("inquiry: invalid image createdBy")
	ErrInvalidImageUpdatedAt  = errors.New("inquiry: invalid image updatedAt")
	ErrInvalidImageUpdatedBy  = errors.New("inquiry: invalid image updatedBy")
	ErrInvalidImageDeletedAt  = errors.New("inquiry: invalid image deletedAt")
	ErrInvalidImageDeletedBy  = errors.New("inquiry: invalid image deletedBy")

	ErrDuplicateImage         = errors.New("inquiry: duplicate image")
	ErrTooManyImages          = errors.New("inquiry: too many images")
	ErrInconsistentInquiry    = errors.New("inquiry: image inquiryId must match inquiry id")
	ErrInquiryAlreadyClosed   = errors.New("inquiry: already closed")
	ErrInquiryForbidden       = errors.New("inquiry: forbidden")
	ErrInquiryInvalidWorkflow = errors.New("inquiry: invalid workflow")
)

// Constructors

func New(
	id, productID, avatarID, subject, content string,
	status InquiryStatus,
	inquiryType InquiryType,
	createdAt, updatedAt time.Time,
) (Inquiry, error) {
	in := Inquiry{
		ID:          id,
		ProductID:   productID,
		AvatarID:    avatarID,
		Subject:     subject,
		Content:     content,
		Status:      status,
		InquiryType: inquiryType,
		IsRead:      false,
		Images:      []ImageFile{},
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
	}

	if in.Status == "" {
		in.Status = InquiryStatusOpen
	}

	if err := in.Validate(); err != nil {
		return Inquiry{}, err
	}
	return in, nil
}

func NewWithOptional(
	id, productID, avatarID, subject, content string,
	status InquiryStatus,
	inquiryType InquiryType,
	createdAt, updatedAt time.Time,
	resolvedAt *time.Time,
	resolvedBy *string,
	closedAt *time.Time,
	closedBy *string,
	updatedBy, deletedBy *string,
	deletedAt *time.Time,
	images []ImageFile,
) (Inquiry, error) {
	in := Inquiry{
		ID:          id,
		ProductID:   productID,
		AvatarID:    avatarID,
		Subject:     subject,
		Content:     content,
		Status:      status,
		InquiryType: inquiryType,
		IsRead:      false,
		Images:      images,
		ResolvedAt:  normalizeOptionalTime(resolvedAt),
		ResolvedBy:  resolvedBy,
		ClosedAt:    normalizeOptionalTime(closedAt),
		ClosedBy:    closedBy,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
		UpdatedBy:   updatedBy,
		DeletedAt:   normalizeOptionalTime(deletedAt),
		DeletedBy:   deletedBy,
	}

	if in.Status == "" {
		in.Status = InquiryStatusOpen
	}
	if in.Images == nil {
		in.Images = []ImageFile{}
	}
	if err := in.Validate(); err != nil {
		return Inquiry{}, err
	}
	return in, nil
}

func NewImageFile(
	inquiryID, fileName, fileURL string,
	objectPath *string,
	fileSize int64,
	mimeType string,
	createdAt time.Time,
	createdBy string,
	updatedAt *time.Time,
	updatedBy *string,
	deletedAt *time.Time,
	deletedBy *string,
) (ImageFile, error) {
	img := ImageFile{
		InquiryID:  inquiryID,
		FileName:   fileName,
		FileURL:    fileURL,
		ObjectPath: objectPath,
		FileSize:   fileSize,
		MimeType:   mimeType,
		CreatedAt:  createdAt.UTC(),
		CreatedBy:  createdBy,
		UpdatedAt:  normalizeOptionalTime(updatedAt),
		UpdatedBy:  updatedBy,
		DeletedAt:  normalizeOptionalTime(deletedAt),
		DeletedBy:  deletedBy,
	}
	if err := validateImageFile(img); err != nil {
		return ImageFile{}, err
	}
	return img, nil
}

func NewImageFileMinimal(
	inquiryID, fileName, fileURL string,
	objectPath *string,
	fileSize int64,
	mimeType string,
	createdAt time.Time,
	createdBy string,
) (ImageFile, error) {
	return NewImageFile(
		inquiryID,
		fileName,
		fileURL,
		objectPath,
		fileSize,
		mimeType,
		createdAt,
		createdBy,
		nil,
		nil,
		nil,
		nil,
	)
}

// Behavior

func (i *Inquiry) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Inquiry) MarkAsRead(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	i.IsRead = true
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Inquiry) MarkAsUnread(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	i.IsRead = false
	i.UpdatedAt = now.UTC()
	return nil
}

// ResolveByMember marks the inquiry as resolved by a company member.
func (i *Inquiry) ResolveByMember(memberID string, now time.Time) error {
	if memberID == "" {
		return ErrInvalidResolvedBy
	}
	if now.IsZero() {
		return ErrInvalidResolvedAt
	}
	if i.Status == InquiryStatusClosed {
		return ErrInquiryAlreadyClosed
	}

	resolvedAt := now.UTC()
	resolvedBy := memberID

	i.Status = InquiryStatusResolved
	i.ResolvedAt = &resolvedAt
	i.ResolvedBy = &resolvedBy
	i.UpdatedAt = resolvedAt
	i.UpdatedBy = &resolvedBy

	return nil
}

// CloseByMember closes the inquiry by a company member.
func (i *Inquiry) CloseByMember(memberID string, now time.Time) error {
	if memberID == "" {
		return ErrInvalidClosedBy
	}
	if now.IsZero() {
		return ErrInvalidClosedAt
	}
	if i.Status == InquiryStatusClosed {
		return ErrInquiryAlreadyClosed
	}

	closedAt := now.UTC()
	closedBy := memberID

	i.Status = InquiryStatusClosed
	i.ClosedAt = &closedAt
	i.ClosedBy = &closedBy
	i.UpdatedAt = closedAt
	i.UpdatedBy = &closedBy

	return nil
}

// CloseByAvatar closes the inquiry by the owner avatar.
//
// Only the avatar that created the inquiry can close it.
func (i *Inquiry) CloseByAvatar(avatarID string, now time.Time) error {
	if avatarID == "" {
		return ErrInvalidAvatarID
	}
	if now.IsZero() {
		return ErrInvalidClosedAt
	}
	if i.AvatarID != avatarID {
		return ErrInquiryForbidden
	}
	if i.Status == InquiryStatusClosed {
		return ErrInquiryAlreadyClosed
	}

	closedAt := now.UTC()
	closedBy := avatarID

	i.Status = InquiryStatusClosed
	i.ClosedAt = &closedAt
	i.ClosedBy = &closedBy
	i.UpdatedAt = closedAt
	i.UpdatedBy = &closedBy

	return nil
}

func (i *Inquiry) AddImage(img ImageFile) error {
	if err := validateImageFile(img); err != nil {
		return err
	}
	if img.InquiryID != i.ID {
		return ErrInconsistentInquiry
	}
	if containsImageURL(i.Images, img.FileURL) {
		return ErrDuplicateImage
	}
	if MaxImages > 0 && len(i.Images) >= MaxImages {
		return ErrTooManyImages
	}

	i.Images = append(i.Images, img)
	return nil
}

func (i *Inquiry) ReplaceImages(images []ImageFile) error {
	out := make([]ImageFile, 0, len(images))
	seen := map[string]struct{}{}

	for _, img := range images {
		if err := validateImageFile(img); err != nil {
			return err
		}
		if img.InquiryID != i.ID {
			return ErrInconsistentInquiry
		}

		u := normURL(img.FileURL)
		if _, ok := seen[u]; ok {
			return ErrDuplicateImage
		}
		seen[u] = struct{}{}

		out = append(out, img)
	}

	if MaxImages > 0 && len(out) > MaxImages {
		return ErrTooManyImages
	}

	i.Images = out
	return nil
}

func (i *Inquiry) RemoveImageByURL(fileURL string) bool {
	fileURL = normURL(fileURL)

	out := i.Images[:0]
	removed := false

	for _, img := range i.Images {
		if normURL(img.FileURL) == fileURL {
			removed = true
			continue
		}
		out = append(out, img)
	}

	i.Images = out
	return removed
}

func (i *Inquiry) RemoveImageByFileName(fileName string) bool {
	out := i.Images[:0]
	removed := false

	for _, img := range i.Images {
		if img.FileName == fileName {
			removed = true
			continue
		}
		out = append(out, img)
	}

	i.Images = out
	return removed
}

func (i Inquiry) FirebaseStorageDeleteOps() []FirebaseStorageDeleteOp {
	out := make([]FirebaseStorageDeleteOp, 0, len(i.Images))

	for _, img := range i.Images {
		if img.ObjectPath == nil || *img.ObjectPath == "" {
			continue
		}
		out = append(out, FirebaseStorageDeleteOp{
			ObjectPath: *img.ObjectPath,
		})
	}

	return out
}

// Validation

func (i Inquiry) Validate() error {
	if i.ID == "" {
		return ErrInvalidID
	}
	if i.ProductID == "" {
		return ErrInvalidProductID
	}
	if i.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if i.Subject == "" {
		return ErrInvalidSubject
	}
	if i.Content == "" {
		return ErrInvalidContent
	}
	if !isValidInquiryStatus(i.Status) {
		return ErrInvalidStatus
	}
	if string(i.InquiryType) == "" {
		return ErrInvalidInquiryType
	}
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if i.UpdatedAt.IsZero() || i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if i.UpdatedBy != nil && *i.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	if i.DeletedAt != nil && i.DeletedAt.Before(i.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if i.DeletedBy != nil && *i.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}

	if err := validateResolvedFields(i); err != nil {
		return err
	}
	if err := validateClosedFields(i); err != nil {
		return err
	}

	if MaxImages > 0 && len(i.Images) > MaxImages {
		return ErrTooManyImages
	}

	seen := map[string]struct{}{}
	for _, img := range i.Images {
		if err := validateImageFile(img); err != nil {
			return err
		}
		if img.InquiryID != i.ID {
			return ErrInconsistentInquiry
		}

		u := normURL(img.FileURL)
		if _, ok := seen[u]; ok {
			return ErrDuplicateImage
		}
		seen[u] = struct{}{}
	}

	return nil
}

func validateResolvedFields(i Inquiry) error {
	if i.ResolvedAt != nil {
		if i.ResolvedAt.IsZero() || i.ResolvedAt.Before(i.CreatedAt) {
			return ErrInvalidResolvedAt
		}
	}
	if i.ResolvedBy != nil && *i.ResolvedBy == "" {
		return ErrInvalidResolvedBy
	}

	if i.Status == InquiryStatusResolved {
		if i.ResolvedAt == nil || i.ResolvedAt.IsZero() {
			return ErrInvalidResolvedAt
		}
		if i.ResolvedBy == nil || *i.ResolvedBy == "" {
			return ErrInvalidResolvedBy
		}
	}

	return nil
}

func validateClosedFields(i Inquiry) error {
	if i.ClosedAt != nil {
		if i.ClosedAt.IsZero() || i.ClosedAt.Before(i.CreatedAt) {
			return ErrInvalidClosedAt
		}
	}
	if i.ClosedBy != nil && *i.ClosedBy == "" {
		return ErrInvalidClosedBy
	}

	if i.Status == InquiryStatusClosed {
		if i.ClosedAt == nil || i.ClosedAt.IsZero() {
			return ErrInvalidClosedAt
		}
		if i.ClosedBy == nil || *i.ClosedBy == "" {
			return ErrInvalidClosedBy
		}
	}

	return nil
}

func validateImageFile(img ImageFile) error {
	if img.InquiryID == "" {
		return ErrInvalidImageInquiryID
	}

	if img.FileName == "" || (MaxFileNameLength > 0 && len([]rune(img.FileName)) > MaxFileNameLength) {
		return ErrInvalidImageFileName
	}

	if !urlOK(img.FileURL) {
		return ErrInvalidImageFileURL
	}

	if img.ObjectPath != nil && *img.ObjectPath == "" {
		return ErrInvalidImageObjectPath
	}

	if img.FileSize < MinFileSizeBytes || (MaxFileSizeBytes > 0 && img.FileSize > MaxFileSizeBytes) {
		return ErrInvalidImageFileSize
	}

	if img.MimeType == "" || (mimeRe != nil && !mimeRe.MatchString(img.MimeType)) {
		return ErrInvalidImageMIMEType
	}
	if len(AllowedMimeTypes) > 0 {
		if _, ok := AllowedMimeTypes[img.MimeType]; !ok {
			return ErrInvalidImageMIMEType
		}
	}

	if img.CreatedAt.IsZero() {
		return ErrInvalidImageCreatedAt
	}
	if img.CreatedBy == "" {
		return ErrInvalidImageCreatedBy
	}

	if img.UpdatedAt != nil {
		if img.UpdatedAt.IsZero() || img.UpdatedAt.Before(img.CreatedAt) {
			return ErrInvalidImageUpdatedAt
		}
	}
	if img.UpdatedBy != nil && *img.UpdatedBy == "" {
		return ErrInvalidImageUpdatedBy
	}

	if img.DeletedAt != nil {
		if img.DeletedAt.IsZero() || img.DeletedAt.Before(img.CreatedAt) {
			return ErrInvalidImageDeletedAt
		}
	}
	if img.DeletedBy != nil && *img.DeletedBy == "" {
		return ErrInvalidImageDeletedBy
	}

	return nil
}

// Helpers

func isValidInquiryStatus(status InquiryStatus) bool {
	switch status {
	case InquiryStatusOpen, InquiryStatusResolved, InquiryStatusClosed:
		return true
	default:
		return false
	}
}

func normalizeOptionalTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func normURL(u string) string {
	return u
}

func urlOK(raw string) bool {
	if raw == "" {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	if len(AllowedURLHosts) > 0 {
		host := strings.ToLower(u.Hostname())
		if _, ok := AllowedURLHosts[host]; !ok {
			return false
		}
	}

	return true
}

func containsImageURL(images []ImageFile, fileURL string) bool {
	fileURL = normURL(fileURL)

	for _, img := range images {
		if normURL(img.FileURL) == fileURL {
			return true
		}
	}

	return false
}
