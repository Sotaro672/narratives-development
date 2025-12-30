// backend/internal/domain/avatarIcon/entity.go
package avatarIcon

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// Default GCS bucket for AvatarIcon objects.
//
// NOTE:
// - GCS bucket 名は DNS 準拠が推奨（underscore は基本NG）
// - 既存バケット名に合わせて必要なら変更してください
const DefaultBucket = "narratives-development-avatar-icon"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// Domain errors
var (
	ErrInvalidID       = errors.New("avatarIcon: invalid id")
	ErrInvalidURL      = errors.New("avatarIcon: invalid url")
	ErrInvalidFileName = errors.New("avatarIcon: invalid fileName")
	ErrInvalidSize     = errors.New("avatarIcon: invalid size")
)

// AvatarIcon mirrors web-app/src/shared/types (without audit fields)
type AvatarIcon struct {
	ID       string  `json:"id"`
	AvatarID *string `json:"avatarId,omitempty"`
	URL      string  `json:"url"`
	FileName *string `json:"fileName,omitempty"`
	Size     *int64  `json:"size,omitempty"`
}

// Policy
var (
	AllowedExtensions = map[string]struct{}{
		".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	}

	// ✅ 旧参照（avicon.DefaultMaxIconSizeBytes）を吸収するため const を用意
	DefaultMaxIconSizeBytes int64 = 10 * 1024 * 1024 // 10MB

	// MaxFileSize は実行時に調整したい場合のため var のまま
	MaxFileSize int64 = 10 * 1024 * 1024 // 10MB
)

/*
Constructors
*/

// New constructs an AvatarIcon with validation.
// Optional fields are nil if not provided.
func New(
	id string,
	urlStr string,
	avatarID, fileName *string,
	size *int64,
) (AvatarIcon, error) {
	a := AvatarIcon{
		ID:       strings.TrimSpace(id),
		AvatarID: normalizeStrPtr(avatarID),
		URL:      strings.TrimSpace(urlStr),
		FileName: normalizeStrPtr(fileName),
		Size:     size,
	}
	if err := a.validate(); err != nil {
		return AvatarIcon{}, err
	}
	return a, nil
}

// NewForCreate kept for compatibility (no audit fields anymore).
func NewForCreate(
	id string,
	input struct {
		URL      string
		AvatarID *string
		FileName *string
		Size     *int64
	},
) (AvatarIcon, error) {
	return New(
		id,
		input.URL,
		input.AvatarID,
		input.FileName,
		input.Size,
	)
}

// NewFromBucketObject constructs an AvatarIcon assuming the file is stored in GCS.
// It builds a public URL like https://storage.googleapis.com/{bucket}/{objectPath}.
// If bucket is empty, DefaultBucket is used.
// fileName is optional (derived from objectPath when nil/empty).
func NewFromBucketObject(
	id string,
	bucket string,
	objectPath string,
	fileName *string,
	size *int64,
) (AvatarIcon, error) {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return AvatarIcon{}, fmt.Errorf("avatarIcon: empty objectPath")
	}

	// Derive fileName from objectPath if not provided
	var fn *string
	if fileName != nil && strings.TrimSpace(*fileName) != "" {
		fn = normalizeStrPtr(fileName)
	} else {
		base := path.Base(obj)
		if base != "." && base != "/" && base != "" {
			fn = &base
		}
	}

	publicURL := PublicURL(b, obj)
	return New(
		id,
		publicURL,
		nil, // AvatarID is optional; caller may set via SetAvatarID
		fn,
		size,
	)
}

/*
Mutators
*/

func (a *AvatarIcon) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if err := validateURL(u); err != nil {
		return err
	}
	a.URL = u
	return nil
}

func (a *AvatarIcon) SetAvatarID(v *string) {
	a.AvatarID = normalizeStrPtr(v)
}

func (a *AvatarIcon) SetFileName(v *string) error {
	v = normalizeStrPtr(v)
	if v != nil && *v != "" && !extAllowed(*v) {
		return ErrInvalidFileName
	}
	a.FileName = v
	return nil
}

func (a *AvatarIcon) SetSize(v *int64) error {
	if v != nil {
		if *v < 0 {
			return ErrInvalidSize
		}
		if MaxFileSize > 0 && *v > MaxFileSize {
			return ErrInvalidSize
		}
	}
	a.Size = v
	return nil
}

/*
Validation
*/

func (a AvatarIcon) validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ErrInvalidID
	}
	if err := validateURL(a.URL); err != nil {
		return err
	}
	if a.FileName != nil && *a.FileName != "" && !extAllowed(*a.FileName) {
		return ErrInvalidFileName
	}
	if a.Size != nil {
		if *a.Size < 0 {
			return ErrInvalidSize
		}
		if MaxFileSize > 0 && *a.Size > MaxFileSize {
			return ErrInvalidSize
		}
	}
	return nil
}

/*
Helpers
*/

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

func extAllowed(name string) bool {
	if len(AllowedExtensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := AllowedExtensions[ext]
	return ok
}

// validateURL returns domain error on invalid URL.
// Adjusted to accept GCS HTTPS endpoints and encourage DefaultBucket.
func validateURL(u string) error {
	if u == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}

	// Allow generic http(s)
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		// If it's a GCS URL, minimally validate bucket/object presence
		host := strings.ToLower(parsed.Host)
		if host == "storage.googleapis.com" || host == "storage.cloud.google.com" {
			if b, obj, ok := ParseGCSURL(u); ok {
				if b == "" || strings.TrimLeft(obj, "/") == "" {
					return ErrInvalidURL
				}
			}
		}
		return nil
	}

	return ErrInvalidURL
}

// PublicURL returns the HTTPS public URL for a GCS object:
// https://storage.googleapis.com/{bucket}/{objectPath}
func PublicURL(bucket, objectPath string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)
}

// ParseGCSURL parses a URL of the form:
// - https://storage.googleapis.com/{bucket}/{objectPath}
// - https://storage.cloud.google.com/{bucket}/{objectPath}
// Returns bucket, objectPath, ok.
func ParseGCSURL(u string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(parsed.Host)
	if host != "storage.googleapis.com" && host != "storage.cloud.google.com" {
		return "", "", false
	}
	p := strings.TrimLeft(parsed.EscapedPath(), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.SplitN(p, "/", 2)
	if len(parts) < 2 {
		return "", "", false
	}
	bucket := parts[0]
	objectPath, _ := url.PathUnescape(parts[1])
	return bucket, objectPath, true
}
