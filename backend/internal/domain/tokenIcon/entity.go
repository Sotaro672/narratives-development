package tokenIcon

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Default GCS bucket for TokenIcon files.
const DefaultBucket = "narratives_development_token_icon"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// TokenIcon mirrors shared/types/tokenIcon.ts
type TokenIcon struct {
	ID       string
	URL      string
	FileName string
	Size     int64
}

// Policy (sync with tokenIconConstants.ts if needed)
var (
	// 0 disables max size check
	MaxFileSize int64 = 10 * 1024 * 1024 // 10MB default
	// Allowed file extensions (optional; empty = no restriction)
	AllowedExtensions = map[string]struct{}{
		".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	}
)

// Errors
var (
	ErrInvalidID       = errors.New("tokenIcon: invalid id")
	ErrInvalidFileName = errors.New("tokenIcon: invalid fileName")
	ErrInvalidSize     = errors.New("tokenIcon: invalid size")
	ErrInvalidURL      = errors.New("tokenIcon: invalid url")
)

// Validation
func (ti TokenIcon) validate() error {
	if strings.TrimSpace(ti.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(ti.URL) == "" {
		return ErrInvalidURL
	}
	if _, err := url.ParseRequestURI(ti.URL); err != nil {
		return ErrInvalidURL
	}
	if strings.TrimSpace(ti.FileName) == "" {
		return ErrInvalidFileName
	}
	if ti.Size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && ti.Size > MaxFileSize {
		return ErrInvalidSize
	}
	return nil
}

// New constructs a TokenIcon with validation.
func New(
	id, u, fileName string,
	size int64,
) (TokenIcon, error) {
	ti := TokenIcon{
		ID:       strings.TrimSpace(id),
		URL:      strings.TrimSpace(u),
		FileName: strings.TrimSpace(fileName),
		Size:     size,
	}
	if err := ti.validate(); err != nil {
		return TokenIcon{}, err
	}
	return ti, nil
}

// NewFromStrings parses simple fields and constructs TokenIcon.
func NewFromStrings(
	id, u, fileName string,
	size int64,
) (TokenIcon, error) {
	return New(id, u, fileName, size)
}

// NewFromGCSObject builds public URL from GCS bucket/object and constructs TokenIcon.
// If bucket is empty, DefaultBucket (narratives_development_token_icon) is used.
func NewFromGCSObject(
	id string,
	fileName string,
	size int64,
	bucket string,
	objectPath string,
) (TokenIcon, error) {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return TokenIcon{}, fmt.Errorf("tokenIcon: empty objectPath")
	}
	publicURL := PublicURL(b, obj)
	return New(id, publicURL, fileName, size)
}

// Mutators

func (t *TokenIcon) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if err := validateURL(u); err != nil {
		return err
	}
	t.URL = u
	return nil
}

func (t *TokenIcon) UpdateFileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidFileName
	}
	if !extAllowed(name) {
		return ErrInvalidFileName
	}
	t.FileName = name
	return nil
}

func (t *TokenIcon) UpdateSize(size int64) error {
	if size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && size > MaxFileSize {
		return ErrInvalidSize
	}
	t.Size = size
	return nil
}

// Helpers

func validateURL(u string) error {
	if u == "" {
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(u)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func extAllowed(name string) bool {
	if len(AllowedExtensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := AllowedExtensions[ext]
	return ok
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

// ToGCSDeleteOp tries to resolve the GCS delete target from this TokenIcon.
// Priority:
// 1) Parse from URL if it points to storage.googleapis.com/cloud.google.com
// 2) Fallback to DefaultBucket + "token_icons/{fileName}"
func (t TokenIcon) ToGCSDeleteOp() GCSDeleteOp {
	if b, obj, ok := ParseGCSURL(t.URL); ok {
		return GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	return GCSDeleteOp{
		Bucket:     DefaultBucket,
		ObjectPath: "token_icons/" + strings.TrimSpace(t.FileName),
	}
}

// TokenIconsTableDDL defines the SQL for the token_icons table migration.
const TokenIconsTableDDL = `
-- Migration: Initialize TokenIcon domain
-- Mirrors backend/internal/domain/tokenIcon/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS token_icons (
  id          TEXT        PRIMARY KEY,
  url         TEXT        NOT NULL,
  file_name   TEXT        NOT NULL,
  size        BIGINT      NOT NULL CHECK (size >= 0),

  -- Non-empty checks
  CONSTRAINT chk_ti_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(url)) > 0
    AND char_length(trim(file_name)) > 0
    AND char_length(trim(created_by)) > 0
    AND char_length(trim(updated_by)) > 0
  ),

  -- simple URL format
  CONSTRAINT chk_ti_url_format CHECK (url ~* '^(https?)://'),
COMMIT;
`
