// backend\internal\domain\tokenContents\entity.go
package tokenContents

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Default GCS bucket for TokenContents files.
const DefaultBucket = "narratives_development_token_contents"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ContentType mirrors TS: 'image' | 'video' | 'pdf' | 'document'
type ContentType string

const (
	TypeImage    ContentType = "image"
	TypeVideo    ContentType = "video"
	TypePDF      ContentType = "pdf"
	TypeDocument ContentType = "document"
)

func IsValidType(t ContentType) bool {
	switch t {
	case TypeImage, TypeVideo, TypePDF, TypeDocument:
		return true
	default:
		return false
	}
}

func AllTypes() []ContentType {
	return []ContentType{TypeImage, TypeVideo, TypePDF, TypeDocument}
}

// GCSTokenContent mirrors web-app/src/shared/types/tokenContent.ts (metadata in GCS-compatible store)
type GCSTokenContent struct {
	ID   string
	Name string
	Type ContentType
	URL  string
	Size int64 // bytes
}

// Errors
var (
	ErrInvalidID   = errors.New("tokenContent: invalid id")
	ErrInvalidName = errors.New("tokenContent: invalid name")
	ErrInvalidType = errors.New("tokenContent: invalid type")
	ErrInvalidURL  = errors.New("tokenContent: invalid url")
	ErrInvalidSize = errors.New("tokenContent: invalid size")
	ErrNotFound    = errors.New("tokenContents: not found")
	ErrConflict    = errors.New("tokenContents: conflict")
	ErrInvalid     = errors.New("tokenContents: invalid")
)

// 判定ヘルパー
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool  { return errors.Is(err, ErrInvalid) }

// ラップヘルパー（原因を保持）
func WrapInvalid(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalid, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrInvalid, msg, err)
}

func WrapConflict(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrConflict, msg, err)
}

func WrapNotFound(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrNotFound, msg, err)
}

// Policy (sync with tokenContentConstants.ts if defined)
var (
	// 0 disables the upper limit check
	MaxFileSize int64 = 50 * 1024 * 1024 // 50MB (adjust as needed)
)

// New creates a GCSTokenContent with validation.
func New(
	id, name string,
	ctype ContentType,
	u string,
	size int64,

) (GCSTokenContent, error) {
	tc := GCSTokenContent{
		ID:   strings.TrimSpace(id),
		Name: strings.TrimSpace(name),
		Type: ctype,
		URL:  strings.TrimSpace(u),
		Size: size,
	}
	if err := tc.validate(); err != nil {
		return GCSTokenContent{}, err
	}
	return tc, nil
}

// NewFromStrings parses audit timestamps from ISO8601/RFC3339 strings.
// Pass "" for deletedAt to keep it nil.
func NewFromStrings(
	id, name string,
	ctype ContentType,
	u string,
	size int64,
) (GCSTokenContent, error) {
	// createdAt/createdBy/updatedAt/updatedBy/deletedAt/deletedBy の処理は削除
	return New(id, name, ctype, u, size)
}

// NewFromGCSObject builds a public URL from the GCS bucket/object and constructs GCSTokenContent.
// If bucket is empty, DefaultBucket (narratives_development_token_contents) is used.
func NewFromGCSObject(
	id string,
	name string,
	ctype ContentType,
	size int64,
	bucket string,
	objectPath string,
) (GCSTokenContent, error) {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return GCSTokenContent{}, fmt.Errorf("tokenContents: empty objectPath")
	}
	publicURL := PublicURL(b, obj)
	// createdAt/createdBy/updatedAt/updatedBy/deletedAt/deletedBy の引数は削除
	return New(id, name, ctype, publicURL, size)
}

// Mutators

func (t *GCSTokenContent) UpdateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	t.Name = name
	return nil
}

func (t *GCSTokenContent) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if _, err := url.ParseRequestURI(u); err != nil {
		return ErrInvalidURL
	}
	t.URL = u
	return nil
}

func (t *GCSTokenContent) UpdateSize(size int64) error {
	if size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && size > MaxFileSize {
		return ErrInvalidSize
	}
	t.Size = size
	return nil
}

// Validation

func (t GCSTokenContent) validate() error {
	if t.ID == "" {
		return ErrInvalidID
	}
	if t.Name == "" {
		return ErrInvalidName
	}
	if !IsValidType(t.Type) {
		return ErrInvalidType
	}
	if _, err := url.ParseRequestURI(t.URL); err != nil {
		return ErrInvalidURL
	}
	if t.Size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && t.Size > MaxFileSize {
		return fmt.Errorf("%w: exceeds MaxFileSize", ErrInvalidSize)
	}
	return nil
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

// ToGCSDeleteOp tries to resolve the GCS delete target from this GCSTokenContent.
// Priority:
// 1) Parse from URL if it points to storage.googleapis.com/cloud.google.com
// 2) Fallback to DefaultBucket + Name (assuming Name equals stored object name)
func (t GCSTokenContent) ToGCSDeleteOp() GCSDeleteOp {
	if b, obj, ok := ParseGCSURL(t.URL); ok {
		return GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	return GCSDeleteOp{
		Bucket:     DefaultBucket,
		ObjectPath: strings.TrimLeft(strings.TrimSpace(t.Name), "/"),
	}
}
