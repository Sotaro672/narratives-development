// backend/internal/domain/avatarIcon/entity.go
package avatarIcon

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

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

	// 旧参照（avicon.DefaultMaxIconSizeBytes）を吸収するため const を用意
	DefaultMaxIconSizeBytes int64 = 10 * 1024 * 1024 // 10MB

	// MaxFileSize は実行時に調整したい場合のため var のまま
	MaxFileSize int64 = 10 * 1024 * 1024 // 10MB
)

/*
Constructors
*/

// New constructs an AvatarIcon with validation.
// Domain layer does not normalize or parse values; it only validates them.
func New(
	id string,
	urlStr string,
	avatarID, fileName *string,
	size *int64,
) (AvatarIcon, error) {
	a := AvatarIcon{
		ID:       id,
		AvatarID: avatarID,
		URL:      urlStr,
		FileName: fileName,
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

/*
Mutators
*/

func (a *AvatarIcon) UpdateURL(u string) error {
	if err := validateURL(u); err != nil {
		return err
	}
	a.URL = u
	return nil
}

func (a *AvatarIcon) SetAvatarID(v *string) {
	a.AvatarID = v
}

func (a *AvatarIcon) SetFileName(v *string) error {
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
	if a.ID == "" {
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

func extAllowed(name string) bool {
	if len(AllowedExtensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := AllowedExtensions[ext]
	return ok
}

// validateURL returns domain error on invalid URL.
// Domain layer validates only; it does not expose parse/build helpers.
func validateURL(u string) error {
	if u == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	return nil
}
