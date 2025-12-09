// backend\internal\domain\inquiryImage\entity.go
package inquiryimage

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Default GCS bucket for InquiryImage files.
const DefaultBucket = "narratives_development_inquiry_image"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ========================================
// Types (mirror TS)
// ========================================

// ImageFile - 問い合わせに紐づく画像ファイル
// TS source of truth:
//
//	export interface ImageFile {
//	  inquiryId: string;
//	  fileName: string;
//	  fileUrl: string;
//	  fileSize: number;
//	  mimeType: string;
//	  width?: number;
//	  height?: number;
//	  createdAt: string | Date;
//	  createdBy: string;
//	  updatedAt?: string | Date;
//	  updatedBy?: string;
//	  deletedAt?: string | Date;
//	  deletedBy?: string;
//	}
type ImageFile struct {
	InquiryID string     `json:"inquiryId"`
	FileName  string     `json:"fileName"`
	FileURL   string     `json:"fileUrl"`
	FileSize  int64      `json:"fileSize"`
	MimeType  string     `json:"mimeType"`
	Width     *int       `json:"width,omitempty"`
	Height    *int       `json:"height,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

// 集約（問い合わせIDと画像一覧）
type InquiryImage struct {
	ID     string      `json:"id"`
	Images []ImageFile `json:"images"`
}

// InquiryImagePatch: 部分更新用（nilは未変更）
type InquiryImagePatch struct {
	// 任意: 必要に応じて他の更新可能フィールドを追加
	// FileURL  *string
	// Caption  *string
	// MimeType *string

	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

var (
	// Entity-level errors
	ErrInvalidID           = errors.New("invalid inquiry id")
	ErrInvalidInquiryID    = errors.New("invalid image inquiryId")
	ErrInvalidFileName     = errors.New("invalid fileName")
	ErrInvalidFileURL      = errors.New("invalid fileUrl")
	ErrInvalidFileSize     = errors.New("invalid fileSize")
	ErrInvalidMIMEType     = errors.New("invalid mimeType")
	ErrInvalidDimensions   = errors.New("invalid dimensions")
	ErrInvalidCreatedAt    = errors.New("invalid createdAt")
	ErrInvalidCreatedBy    = errors.New("invalid createdBy")
	ErrInvalidUpdatedAt    = errors.New("invalid updatedAt")
	ErrInvalidUpdatedBy    = errors.New("invalid updatedBy")
	ErrInvalidDeletedAt    = errors.New("invalid deletedAt")
	ErrInvalidDeletedBy    = errors.New("invalid deletedBy")
	ErrDuplicateImage      = errors.New("duplicate image")
	ErrTooManyImages       = errors.New("画像の最大枚数を超えています")
	ErrInconsistentInquiry = errors.New("image inquiryId must match aggregate id")
)

// ========================================
// Policy (align with inquiryImageConstants.ts)
// ========================================

var (
	// Limits (0 disables upper-bound checks)
	MaxImages               = 10
	MinFileSizeBytes  int64 = 1
	MaxFileSizeBytes  int64 = 20 * 1024 * 1024 // 20 MB
	MaxFileNameLength       = 255
	AllowedMimeTypes        = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
		"image/gif":  {},
	}
	// Optional allow-list for host names (empty = allow all)
	AllowedURLHosts = map[string]struct{}{}

	// Optional stricter mime validation (nil disables)
	mimeRe = regexp.MustCompile(`^[a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+$`)
)

// ========================================
// Constructors
// ========================================

// NewImageFile creates a new ImageFile with full set of fields (optional fields can be nil).
func NewImageFile(
	inquiryID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	createdBy string,
	updatedAt *time.Time,
	updatedBy *string,
	deletedAt *time.Time,
	deletedBy *string,
) (ImageFile, error) {
	img := ImageFile{
		InquiryID: strings.TrimSpace(inquiryID),
		FileName:  strings.TrimSpace(fileName),
		FileURL:   strings.TrimSpace(fileURL),
		FileSize:  fileSize,
		MimeType:  strings.TrimSpace(mimeType),
		Width:     normalizeIntPtr(width),
		Height:    normalizeIntPtr(height),

		CreatedAt: createdAt.UTC(),
		CreatedBy: strings.TrimSpace(createdBy),

		UpdatedAt: normalizeTimePtr(updatedAt),
		UpdatedBy: normalizeStrPtr(updatedBy),
		DeletedAt: normalizeTimePtr(deletedAt),
		DeletedBy: normalizeStrPtr(deletedBy),
	}
	if err := validateImageFile(img); err != nil {
		return ImageFile{}, err
	}
	return img, nil
}

// NewImageFileMinimal creates ImageFile with required fields only.
func NewImageFileMinimal(
	inquiryID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	createdBy string,
) (ImageFile, error) {
	return NewImageFile(inquiryID, fileName, fileURL, fileSize, mimeType, width, height, createdAt, createdBy, nil, nil, nil, nil)
}

// NewImageFileFromStringTimes parses createdAt/updatedAt/deletedAt from string (RFC3339 preferred).
func NewImageFileFromStringTimes(
	inquiryID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAtStr, createdBy string,
	updatedAtStr, deletedAtStr *string,
	updatedBy, deletedBy *string,
) (ImageFile, error) {
	ct, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return ImageFile{}, err
	}
	var ut *time.Time
	if updatedAtStr != nil {
		if t, err := parseTime(*updatedAtStr, ErrInvalidUpdatedAt); err == nil {
			ut = &t
		} else {
			return ImageFile{}, err
		}
	}
	var dt *time.Time
	if deletedAtStr != nil {
		if t, err := parseTime(*deletedAtStr, ErrInvalidDeletedAt); err == nil {
			dt = &t
		} else {
			return ImageFile{}, err
		}
	}
	return NewImageFile(inquiryID, fileName, fileURL, fileSize, mimeType, width, height, ct, createdBy, ut, updatedBy, dt, deletedBy)
}

func NewInquiryImage(id string, images []ImageFile) (InquiryImage, error) {
	ii := InquiryImage{
		ID:     strings.TrimSpace(id),
		Images: dedupByURL(images),
	}
	if err := validateInquiryImage(ii); err != nil {
		return InquiryImage{}, err
	}
	return ii, nil
}

// ========================================
// Behavior
// ========================================

func (i *InquiryImage) AddImage(img ImageFile) error {
	if err := validateImageFile(img); err != nil {
		return err
	}
	// enforce inquiry id consistency if aggregate has an id set
	if strings.TrimSpace(i.ID) != "" && img.InquiryID != i.ID {
		return ErrInconsistentInquiry
	}
	// duplicate check by URL
	if containsURL(i.Images, img.FileURL) {
		return ErrDuplicateImage
	}
	// capacity check
	if MaxImages > 0 && len(i.Images) >= MaxImages {
		return ErrTooManyImages
	}
	i.Images = append(i.Images, img)
	return nil
}

func (i *InquiryImage) ReplaceImages(images []ImageFile) error {
	// validate each, dedup by URL, and enforce inquiry id consistency
	dedup := make([]ImageFile, 0, len(images))
	seen := map[string]struct{}{}
	for _, im := range images {
		if err := validateImageFile(im); err != nil {
			return err
		}
		if strings.TrimSpace(i.ID) != "" && im.InquiryID != i.ID {
			return ErrInconsistentInquiry
		}
		u := normURL(im.FileURL)
		if _, ok := seen[u]; ok {
			return ErrDuplicateImage
		}
		seen[u] = struct{}{}
		dedup = append(dedup, im)
	}
	if MaxImages > 0 && len(dedup) > MaxImages {
		return ErrTooManyImages
	}
	i.Images = dedup
	return nil
}

func (i *InquiryImage) RemoveImageByURL(u string) bool {
	u = normURL(u)
	out := i.Images[:0]
	removed := false
	for _, im := range i.Images {
		if normURL(im.FileURL) == u {
			removed = true
			continue
		}
		out = append(out, im)
	}
	i.Images = out
	return removed
}

// ========================================
// Validation
// ========================================

func validateInquiryImage(i InquiryImage) error {
	if strings.TrimSpace(i.ID) == "" {
		return ErrInvalidID
	}
	if MaxImages > 0 && len(i.Images) > MaxImages {
		return ErrTooManyImages
	}
	seen := map[string]struct{}{}
	for _, im := range i.Images {
		if err := validateImageFile(im); err != nil {
			return err
		}
		// enforce consistency
		if im.InquiryID != i.ID {
			return ErrInconsistentInquiry
		}
		u := normURL(im.FileURL)
		if _, ok := seen[u]; ok {
			return ErrDuplicateImage
		}
		seen[u] = struct{}{}
	}
	return nil
}

func validateImageFile(im ImageFile) error {
	// inquiryId
	if strings.TrimSpace(im.InquiryID) == "" {
		return ErrInvalidInquiryID
	}
	// fileName
	if im.FileName == "" || (MaxFileNameLength > 0 && len([]rune(im.FileName)) > MaxFileNameLength) {
		return ErrInvalidFileName
	}
	// URL
	if !urlOK(im.FileURL) {
		return ErrInvalidFileURL
	}
	// fileSize
	if im.FileSize < MinFileSizeBytes || (MaxFileSizeBytes > 0 && im.FileSize > MaxFileSizeBytes) {
		return ErrInvalidFileSize
	}
	// mime
	if im.MimeType == "" || (mimeRe != nil && !mimeRe.MatchString(im.MimeType)) {
		return ErrInvalidMIMEType
	}
	if len(AllowedMimeTypes) > 0 {
		if _, ok := AllowedMimeTypes[im.MimeType]; !ok {
			return ErrInvalidMIMEType
		}
	}
	// dimensions
	if im.Width != nil && *im.Width <= 0 {
		return ErrInvalidDimensions
	}
	if im.Height != nil && *im.Height <= 0 {
		return ErrInvalidDimensions
	}
	// createdAt/createdBy
	if im.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(im.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	// updated
	if im.UpdatedAt != nil {
		if im.UpdatedAt.IsZero() || im.UpdatedAt.Before(im.CreatedAt) {
			return ErrInvalidUpdatedAt
		}
	}
	if im.UpdatedBy != nil && strings.TrimSpace(*im.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	// deleted
	if im.DeletedAt != nil && im.DeletedAt.Before(im.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if im.DeletedBy != nil && strings.TrimSpace(*im.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// ========================================
// Helpers
// ========================================

func normURL(u string) string {
	return strings.TrimSpace(u)
}

func urlOK(raw string) bool {
	raw = strings.TrimSpace(raw)
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

func normalizeIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	if *p <= 0 {
		return nil
	}
	v := *p
	return &v
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

func containsURL(xs []ImageFile, u string) bool {
	u = normURL(u)
	for _, x := range xs {
		if normURL(x.FileURL) == u {
			return true
		}
	}
	return false
}

func dedupByURL(xs []ImageFile) []ImageFile {
	seen := make(map[string]struct{}, len(xs))
	out := make([]ImageFile, 0, len(xs))
	for _, x := range xs {
		u := normURL(x.FileURL)
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, x)
	}
	return out
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
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
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}
