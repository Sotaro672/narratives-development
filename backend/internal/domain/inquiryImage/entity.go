// backend\internal\domain\inquiryImage\entity.go
package inquiryimage

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// FirebaseStorageDeleteOp represents a delete operation target in Firebase Storage.
// ObjectPath is the Firebase Storage object path, for example:
// inquiry-images/{inquiryId}/{imageId}/{fileName}
type FirebaseStorageDeleteOp struct {
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
//	  objectPath?: string;
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
	InquiryID  string     `json:"inquiryId"`
	FileName   string     `json:"fileName"`
	FileURL    string     `json:"fileUrl"`
	ObjectPath *string    `json:"objectPath,omitempty"`
	FileSize   int64      `json:"fileSize"`
	MimeType   string     `json:"mimeType"`
	Width      *int       `json:"width,omitempty"`
	Height     *int       `json:"height,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  string     `json:"createdBy"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy  *string    `json:"updatedBy,omitempty"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
	DeletedBy  *string    `json:"deletedBy,omitempty"`
}

// 集約（問い合わせIDと画像一覧）
type InquiryImage struct {
	ID     string      `json:"id"`
	Images []ImageFile `json:"images"`
}

// InquiryImagePatch: 部分更新用（nilは未変更）
type InquiryImagePatch struct {
	// 任意: 必要に応じて他の更新可能フィールドを追加
	// FileURL    *string
	// ObjectPath *string
	// Caption    *string
	// MimeType   *string

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
	ErrInvalidObjectPath   = errors.New("invalid objectPath")
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

// NewImageFile creates a new ImageFile with full set of fields.
// fileURL is expected to be the Firebase Storage download URL.
// objectPath is expected to be the Firebase Storage object path.
func NewImageFile(
	inquiryID, fileName, fileURL string,
	objectPath *string,
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
		InquiryID:  inquiryID,
		FileName:   fileName,
		FileURL:    fileURL,
		ObjectPath: objectPath,
		FileSize:   fileSize,
		MimeType:   mimeType,
		Width:      width,
		Height:     height,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
		UpdatedAt:  updatedAt,
		UpdatedBy:  updatedBy,
		DeletedAt:  deletedAt,
		DeletedBy:  deletedBy,
	}
	if err := validateImageFile(img); err != nil {
		return ImageFile{}, err
	}
	return img, nil
}

// NewImageFileMinimal creates ImageFile with required fields only.
func NewImageFileMinimal(
	inquiryID, fileName, fileURL string,
	objectPath *string,
	fileSize int64,
	mimeType string,
	width, height *int,
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
		width,
		height,
		createdAt,
		createdBy,
		nil,
		nil,
		nil,
		nil,
	)
}

func NewInquiryImage(id string, images []ImageFile) (InquiryImage, error) {
	ii := InquiryImage{
		ID:     id,
		Images: images,
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
	if i.ID != "" && img.InquiryID != i.ID {
		return ErrInconsistentInquiry
	}
	if containsURL(i.Images, img.FileURL) {
		return ErrDuplicateImage
	}
	if MaxImages > 0 && len(i.Images) >= MaxImages {
		return ErrTooManyImages
	}
	i.Images = append(i.Images, img)
	return nil
}

func (i *InquiryImage) ReplaceImages(images []ImageFile) error {
	out := make([]ImageFile, 0, len(images))
	seen := map[string]struct{}{}

	for _, im := range images {
		if err := validateImageFile(im); err != nil {
			return err
		}
		if i.ID != "" && im.InquiryID != i.ID {
			return ErrInconsistentInquiry
		}

		u := normURL(im.FileURL)
		if _, ok := seen[u]; ok {
			return ErrDuplicateImage
		}
		seen[u] = struct{}{}
		out = append(out, im)
	}

	if MaxImages > 0 && len(out) > MaxImages {
		return ErrTooManyImages
	}
	i.Images = out
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

func (i InquiryImage) FirebaseStorageDeleteOps() []FirebaseStorageDeleteOp {
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

// ========================================
// Validation
// ========================================

func validateInquiryImage(i InquiryImage) error {
	if i.ID == "" {
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
	if im.InquiryID == "" {
		return ErrInvalidInquiryID
	}

	if im.FileName == "" || (MaxFileNameLength > 0 && len([]rune(im.FileName)) > MaxFileNameLength) {
		return ErrInvalidFileName
	}

	if !urlOK(im.FileURL) {
		return ErrInvalidFileURL
	}

	if im.ObjectPath != nil && *im.ObjectPath == "" {
		return ErrInvalidObjectPath
	}

	if im.FileSize < MinFileSizeBytes || (MaxFileSizeBytes > 0 && im.FileSize > MaxFileSizeBytes) {
		return ErrInvalidFileSize
	}

	if im.MimeType == "" || (mimeRe != nil && !mimeRe.MatchString(im.MimeType)) {
		return ErrInvalidMIMEType
	}
	if len(AllowedMimeTypes) > 0 {
		if _, ok := AllowedMimeTypes[im.MimeType]; !ok {
			return ErrInvalidMIMEType
		}
	}

	if im.Width != nil && *im.Width <= 0 {
		return ErrInvalidDimensions
	}
	if im.Height != nil && *im.Height <= 0 {
		return ErrInvalidDimensions
	}

	if im.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if im.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if im.UpdatedAt != nil {
		if im.UpdatedAt.IsZero() || im.UpdatedAt.Before(im.CreatedAt) {
			return ErrInvalidUpdatedAt
		}
	}
	if im.UpdatedBy != nil && *im.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	if im.DeletedAt != nil {
		if im.DeletedAt.IsZero() || im.DeletedAt.Before(im.CreatedAt) {
			return ErrInvalidDeletedAt
		}
	}
	if im.DeletedBy != nil && *im.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}

	return nil
}

// ========================================
// Helpers
// ========================================

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

func containsURL(xs []ImageFile, u string) bool {
	u = normURL(u)

	for _, x := range xs {
		if normURL(x.FileURL) == u {
			return true
		}
	}

	return false
}
