// backend\internal\domain\campaignImage\entity.go
package campaignImage

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// Default GCS bucket for CampaignImage objects.
const DefaultBucket = "narratives_development_campaign_image"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// Entity (mirror web-app/src/shared/types/campaignImage.ts)
type CampaignImage struct {
	ID         string  `json:"id"`
	CampaignID string  `json:"campaignId"`
	ImageURL   string  `json:"imageUrl"`
	Width      *int    `json:"width,omitempty"`
	Height     *int    `json:"height,omitempty"`
	FileSize   *int64  `json:"fileSize,omitempty"`
	MimeType   *string `json:"mimeType,omitempty"`
}

// Errors
var (
	ErrInvalidID         = errors.New("campaignImage: invalid id")
	ErrInvalidCampaignID = errors.New("campaignImage: invalid campaignId")
	ErrInvalidImageURL   = errors.New("campaignImage: invalid imageUrl")
	ErrInvalidDimensions = errors.New("campaignImage: invalid width/height")
	ErrInvalidFileSize   = errors.New("campaignImage: invalid fileSize")
	ErrInvalidMimeType   = errors.New("campaignImage: invalid mimeType")
)

// Policy
var (
	// Dimensions: positive integers if present
	RequirePositiveDimensions = true

	// File size bounds (0 disables upper check)
	MinFileSizeBytes int64 = 1
	MaxFileSizeBytes int64 = 20 * 1024 * 1024 // 20MB

	// Allowed mime types (empty map = allow all that match mimeRe)
	AllowedMimeTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
		"image/gif":  {},
	}
	mimeRe = regexp.MustCompile(`^[a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+$`)

	// Optional allow-list for URL hosts (empty = allow all)
	AllowedURLHosts = map[string]struct{}{}
)

// Constructors

func New(
	id, campaignID, imageURL string,
	width, height *int,
	fileSize *int64,
	mimeType *string,
) (CampaignImage, error) {
	c := CampaignImage{
		ID:         strings.TrimSpace(id),
		CampaignID: strings.TrimSpace(campaignID),
		ImageURL:   strings.TrimSpace(imageURL),
		Width:      normalizeIntPtr(width),
		Height:     normalizeIntPtr(height),
		FileSize:   normalizeInt64Ptr(fileSize),
		MimeType:   normalizeStrPtr(mimeType),
	}
	if err := c.validate(); err != nil {
		return CampaignImage{}, err
	}
	return c, nil
}

// Behavior

func (c *CampaignImage) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if !urlOK(u) {
		return ErrInvalidImageURL
	}
	c.ImageURL = u
	return nil
}

func (c *CampaignImage) SetDimensions(width, height *int) error {
	w := normalizeIntPtr(width)
	h := normalizeIntPtr(height)
	if RequirePositiveDimensions {
		if w != nil && *w <= 0 {
			return ErrInvalidDimensions
		}
		if h != nil && *h <= 0 {
			return ErrInvalidDimensions
		}
	}
	c.Width = w
	c.Height = h
	return nil
}

func (c *CampaignImage) SetFileMeta(fileSize *int64, mimeType *string) error {
	fs := normalizeInt64Ptr(fileSize)
	mt := normalizeStrPtr(mimeType)
	if fs != nil && !fileSizeOK(*fs) {
		return ErrInvalidFileSize
	}
	if mt != nil && !mimeOK(*mt) {
		return ErrInvalidMimeType
	}
	c.FileSize = fs
	c.MimeType = mt
	return nil
}

// Validation

func (c CampaignImage) validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(c.CampaignID) == "" {
		return ErrInvalidCampaignID
	}
	if !urlOK(c.ImageURL) {
		return ErrInvalidImageURL
	}
	if c.Width != nil && *c.Width <= 0 {
		return ErrInvalidDimensions
	}
	if c.Height != nil && *c.Height <= 0 {
		return ErrInvalidDimensions
	}
	if c.FileSize != nil && !fileSizeOK(*c.FileSize) {
		return ErrInvalidFileSize
	}
	if c.MimeType != nil && !mimeOK(*c.MimeType) {
		return ErrInvalidMimeType
	}
	return nil
}

// Helpers

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func normalizeInt64Ptr(p *int64) *int64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func fileSizeOK(v int64) bool {
	if v < MinFileSizeBytes {
		return false
	}
	if MaxFileSizeBytes > 0 && v > MaxFileSizeBytes {
		return false
	}
	return true
}

func mimeOK(mt string) bool {
	mt = strings.TrimSpace(mt)
	if mt == "" || (mimeRe != nil && !mimeRe.MatchString(mt)) {
		return false
	}
	if len(AllowedMimeTypes) > 0 {
		if _, ok := AllowedMimeTypes[mt]; !ok {
			return false
		}
	}
	return true
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
