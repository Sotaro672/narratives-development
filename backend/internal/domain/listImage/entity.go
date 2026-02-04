// backend/internal/domain/listImage/entity.go
package listImage

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Default GCS bucket for ListImage files.
// NOTE:
// - In production, bucket should be injected from application/httpout via env.
// - This constant is kept only as a fallback for development or legacy data.
const DefaultBucket = "narratives-development-list"

// DefaultObjectPathPrefix is the canonical prefix for list images in a single bucket.
//
// Expected layout (recommended):
//
//	gs://{bucket}/lists/{listId}/images/{imageId}
//
// (listId folder can contain multiple images)
const DefaultObjectPathPrefix = "lists"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ListImage mirrors web-app/src/shared/types/listImage.ts
// TS source of truth (updated):
//
//	export interface ListImage {
//	  id: string;
//	  listId: string;
//	  url: string;
//	  fileName: string;
//	  size: number;
//	  displayOrder: number;
//	}
//
// Domain additions:
//   - ObjectPath is the canonical GCS object path used for upload/update/delete.
//     This avoids relying on URL parsing or fileName-based inference.
type ListImage struct {
	ID           string
	ListID       string
	URL          string
	ObjectPath   string // ✅ canonical: lists/{listId}/images/{imageId}
	FileName     string
	Size         int64
	DisplayOrder int
}

// ImageFileValidation - 画像ファイルのバリデーション結果
type ImageFileValidation struct {
	IsValid      bool   `json:"isValid"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// エラーメッセージ（UI向け）
const (
	ErrMsgInvalidFileType = "無効なファイル形式です"
	ErrMsgFileTooLarge    = "ファイルサイズが大きすぎます"
	ErrMsgUploadFailed    = "画像のアップロードに失敗しました"
)

// ドメインエラー
var (
	ErrInvalidFileType     = errors.New(ErrMsgInvalidFileType)
	ErrFileTooLarge        = errors.New(ErrMsgFileTooLarge)
	ErrUploadFailed        = errors.New(ErrMsgUploadFailed)
	ErrInvalidID           = errors.New("listImage: invalid id")
	ErrInvalidListID       = errors.New("listImage: invalid listId")
	ErrInvalidURL          = errors.New("listImage: invalid url")
	ErrInvalidObjectPath   = errors.New("listImage: invalid objectPath")
	ErrInvalidFileName     = errors.New("listImage: invalid fileName")
	ErrInvalidSize         = errors.New("listImage: invalid size")
	ErrInvalidDisplayOrder = errors.New("listImage: invalid displayOrder")
)

// NewImageFileValidation - エラーから検証結果を作成
func NewImageFileValidation(err error) ImageFileValidation {
	if err == nil {
		return ImageFileValidation{IsValid: true}
	}
	return ImageFileValidation{IsValid: false, ErrorMessage: err.Error()}
}

// ========================================
// エラーハンドリング/バリデーション（serviceから移譲）
// ========================================

const DefaultMaxImageSizeBytes = 5 * 1024 * 1024 // 5MB

var SupportedImageMIMEs = map[string]struct{}{
	"image/jpeg": {},
	"image/jpg":  {},
	"image/png":  {},
	"image/webp": {},
}

// RequireNonEmpty - 必須文字列チェック
func RequireNonEmpty(name, v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// ValidateDataURL - data URL形式（data:<mime>;base64,<payload>）を検証
// 返り値: mime, デコード済みバイト列（必要なら呼び出し側で利用可能）
func ValidateDataURL(data string, maxBytes int, supported map[string]struct{}) (mime string, payload []byte, err error) {
	if !strings.HasPrefix(data, "data:") {
		return "", nil, errors.New("invalid data URL: missing 'data:' prefix")
	}
	parts := strings.SplitN(data, ",", 2)
	if len(parts) != 2 {
		return "", nil, errors.New("invalid data URL: missing payload")
	}
	meta := parts[0] // e.g. data:image/png;base64
	raw := parts[1]

	if !strings.Contains(meta, ";base64") {
		return "", nil, errors.New("invalid data URL: not base64 encoded")
	}

	mime = strings.TrimPrefix(strings.SplitN(meta, ";", 2)[0], "data:")
	if mime == "" {
		return "", nil, errors.New("invalid data URL: missing mime type")
	}
	if _, ok := supported[mime]; !ok {
		return "", nil, fmt.Errorf("unsupported content type: %s", mime)
	}

	decoded, decErr := base64.StdEncoding.DecodeString(raw)
	if decErr != nil {
		return "", nil, fmt.Errorf("invalid base64 payload: %w", decErr)
	}
	if len(decoded) == 0 {
		return "", nil, errors.New("empty image payload")
	}
	if len(decoded) > maxBytes {
		return "", nil, fmt.Errorf("file too large: %d bytes (max %d)", len(decoded), maxBytes)
	}

	return mime, decoded, nil
}

// Policy (align with listImageConstants.ts as needed)
var (
	// Allowed file extensions for listing images (empty map disables the check)
	// NOTE: gif is NOT allowed unless SupportedImageMIMEs also supports image/gif.
	AllowedExtensions = map[string]struct{}{
		".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {},
	}
	// 0 disables the upper limit check
	MaxFileSize int64 = 20 * 1024 * 1024 // 20MB
)

// ========================================
// Constructors
// ========================================

// New creates a ListImage with validation.
//
// NOTE:
// - ObjectPath is required to make update/delete stable without relying on URL parsing.
// - URL is still required because frontend consumes it (public URL).
func New(
	id, listID, u, objectPath, fileName string,
	size int64,
	displayOrder int,
) (ListImage, error) {
	li := ListImage{
		ID:           strings.TrimSpace(id),
		ListID:       strings.TrimSpace(listID),
		URL:          strings.TrimSpace(u),
		ObjectPath:   strings.TrimLeft(strings.TrimSpace(objectPath), "/"),
		FileName:     strings.TrimSpace(fileName),
		Size:         size,
		DisplayOrder: displayOrder,
	}
	if err := li.validate(); err != nil {
		return ListImage{}, err
	}
	return li, nil
}

// NewMinimal - 必須項目のみで作成（New と同義）
func NewMinimal(
	id, listID, u, objectPath, fileName string,
	size int64,
	displayOrder int,
) (ListImage, error) {
	return New(id, listID, u, objectPath, fileName, size, displayOrder)
}

// NewFromGCSObject builds public URL from GCS bucket/object and constructs ListImage.
// If bucket is empty, DefaultBucket (narratives-development-list) is used.
func NewFromGCSObject(
	id, listID, fileName string,
	size int64,
	displayOrder int,
	bucket string,
	objectPath string,
) (ListImage, error) {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return ListImage{}, fmt.Errorf("listImage: empty objectPath")
	}
	publicURL := PublicURL(b, obj)
	return New(id, listID, publicURL, obj, fileName, size, displayOrder)
}

// NewMinimalFromGCSObject - minimal constructor using GCS bucket/object.
func NewMinimalFromGCSObject(
	id, listID, fileName string,
	size int64,
	displayOrder int,
	bucket string,
	objectPath string,
) (ListImage, error) {
	return NewFromGCSObject(id, listID, fileName, size, displayOrder, bucket, objectPath)
}

// NewWithCanonicalPath builds objectPath as:
//
//	lists/{listId}/images/{imageId}
//
// It then builds public URL from bucket+objectPath and constructs ListImage.
// If bucket is empty, DefaultBucket is used.
func NewWithCanonicalPath(
	id, listID, fileName string,
	size int64,
	displayOrder int,
	bucket string,
) (ListImage, error) {
	obj := CanonicalObjectPath(listID, id)
	return NewFromGCSObject(id, listID, fileName, size, displayOrder, bucket, obj)
}

// ========================================
// Mutators
// ========================================

func (l *ListImage) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if err := validateURL(u); err != nil {
		return err
	}
	l.URL = u
	return nil
}

func (l *ListImage) UpdateObjectPath(objectPath string) error {
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if err := validateObjectPath(obj); err != nil {
		return err
	}
	l.ObjectPath = obj
	return nil
}

func (l *ListImage) UpdateFileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidFileName
	}
	if !extAllowed(name) {
		return ErrInvalidFileName
	}
	l.FileName = name
	return nil
}

func (l *ListImage) UpdateSize(size int64) error {
	if size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && size > MaxFileSize {
		return ErrInvalidSize
	}
	l.Size = size
	return nil
}

func (l *ListImage) SetDisplayOrder(order int) error {
	if order < 0 {
		return ErrInvalidDisplayOrder
	}
	l.DisplayOrder = order
	return nil
}

// ========================================
// Validation
// ========================================

func (l ListImage) validate() error {
	if strings.TrimSpace(l.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(l.ListID) == "" {
		return ErrInvalidListID
	}
	if err := validateURL(l.URL); err != nil {
		return err
	}
	if err := validateObjectPath(l.ObjectPath); err != nil {
		return err
	}
	if l.FileName == "" || !extAllowed(l.FileName) {
		return ErrInvalidFileName
	}
	if l.Size < 0 {
		return ErrInvalidSize
	}
	if MaxFileSize > 0 && l.Size > MaxFileSize {
		return ErrInvalidSize
	}
	if l.DisplayOrder < 0 {
		return ErrInvalidDisplayOrder
	}
	return nil
}

// ========================================
// Helpers
// ========================================

func validateURL(u string) error {
	if u == "" {
		return ErrInvalidURL
	}
	pu, err := url.ParseRequestURI(u)
	if err != nil {
		return ErrInvalidURL
	}
	if pu.Scheme == "" || pu.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

// validateObjectPath validates the canonical object path stored for stable operations.
// We keep it permissive, but must be non-empty and not contain URL scheme.
func validateObjectPath(p string) error {
	p = strings.TrimLeft(strings.TrimSpace(p), "/")
	if p == "" {
		return ErrInvalidObjectPath
	}
	// prevent accidentally storing URL
	if strings.Contains(p, "://") {
		return ErrInvalidObjectPath
	}
	// guard: no backslashes
	if strings.Contains(p, `\`) {
		return ErrInvalidObjectPath
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

// CanonicalObjectPath returns canonical object path for list image:
// lists/{listId}/images/{imageId}
func CanonicalObjectPath(listID, imageID string) string {
	return strings.TrimLeft(
		fmt.Sprintf("%s/%s/images/%s", DefaultObjectPathPrefix, strings.TrimSpace(listID), strings.TrimSpace(imageID)),
		"/",
	)
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

// ToGCSDeleteOp resolves the GCS delete target from this ListImage.
// Priority:
// 1) Use explicit ObjectPath (recommended)
// 2) Parse from URL if it points to storage.googleapis.com/cloud.google.com (legacy)
// 3) Fallback to DefaultBucket + canonical path (lists/{listId}/images/{imageId})
func (l ListImage) ToGCSDeleteOp() GCSDeleteOp {
	// 1) explicit ObjectPath
	if obj := strings.TrimLeft(strings.TrimSpace(l.ObjectPath), "/"); obj != "" {
		return GCSDeleteOp{
			Bucket:     DefaultBucket,
			ObjectPath: obj,
		}
	}

	// 2) parse from URL (legacy)
	if b, obj, ok := ParseGCSURL(l.URL); ok {
		return GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}

	// 3) canonical fallback
	return GCSDeleteOp{
		Bucket:     DefaultBucket,
		ObjectPath: CanonicalObjectPath(l.ListID, l.ID),
	}
}
