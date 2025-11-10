// backend\internal\domain\listImage\entity.go
package listImage

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// Default GCS bucket for ListImage files.
const DefaultBucket = "narratives_development_list_image"

// GCSDeleteOp represents a delete operation target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ListImage mirrors web-app/src/shared/types/listImage.ts
// TS source of truth:
//
//	export interface ListImage {
//	  id: string;
//	  listId: string;
//	  url: string;
//	  fileName: string;
//	  size: number;
//	  displayOrder: number;
//	  createdAt: Date | string;
//	  createdBy: string;
//	  updatedAt?: Date | string;
//	  updatedBy?: string;
//	  deletedAt: Date | string | null;
//	  deletedBy?: string | null;
//	}
type ListImage struct {
	ID           string
	ListID       string
	URL          string
	FileName     string
	Size         int64
	DisplayOrder int

	CreatedAt time.Time
	CreatedBy string
	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
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
	ErrInvalidFileName     = errors.New("listImage: invalid fileName")
	ErrInvalidSize         = errors.New("listImage: invalid size")
	ErrInvalidDisplayOrder = errors.New("listImage: invalid displayOrder")
	ErrInvalidCreatedAt    = errors.New("listImage: invalid createdAt")
	ErrInvalidCreatedBy    = errors.New("listImage: invalid createdBy")
	ErrInvalidUpdatedAt    = errors.New("listImage: invalid updatedAt")
	ErrInvalidUpdatedBy    = errors.New("listImage: invalid updatedBy")
	ErrInvalidDeletedAt    = errors.New("listImage: invalid deletedAt")
	ErrInvalidDeletedBy    = errors.New("listImage: invalid deletedBy")
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
	AllowedExtensions = map[string]struct{}{
		".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	}
	// 0 disables the upper limit check
	MaxFileSize int64 = 20 * 1024 * 1024 // 20MB
)

// ========================================
// Constructors
// ========================================

// New creates a ListImage with validation.
// Optional fields updatedAt/updatedBy/deletedAt/deletedBy can be nil.
func New(
	id, listID, u, fileName string,
	size int64,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
	updatedAt *time.Time,
	updatedBy *string,
	deletedAt *time.Time,
	deletedBy *string,
) (ListImage, error) {
	li := ListImage{
		ID:           strings.TrimSpace(id),
		ListID:       strings.TrimSpace(listID),
		URL:          strings.TrimSpace(u),
		FileName:     strings.TrimSpace(fileName),
		Size:         size,
		DisplayOrder: displayOrder,

		CreatedAt: createdAt.UTC(),
		CreatedBy: strings.TrimSpace(createdBy),
		UpdatedAt: normalizeTimePtr(updatedAt),
		UpdatedBy: normalizeStrPtr(updatedBy),
		DeletedAt: normalizeTimePtr(deletedAt),
		DeletedBy: normalizeStrPtr(deletedBy),
	}
	if err := li.validate(); err != nil {
		return ListImage{}, err
	}
	return li, nil
}

// NewMinimal - 必須項目のみで作成
func NewMinimal(
	id, listID, u, fileName string,
	size int64,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
) (ListImage, error) {
	return New(id, listID, u, fileName, size, displayOrder, createdAt, createdBy, nil, nil, nil, nil)
}

// NewFromStringTimes parses createdAt/updatedAt/deletedAt from ISO8601 string (RFC3339優先)
func NewFromStringTimes(
	id, listID, u, fileName string,
	size int64,
	displayOrder int,
	createdAtStr, createdBy string,
	updatedAtStr, deletedAtStr *string,
	updatedBy, deletedBy *string,
) (ListImage, error) {
	ca, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return ListImage{}, err
	}
	var ua *time.Time
	if updatedAtStr != nil {
		t, e := parseTime(*updatedAtStr, ErrInvalidUpdatedAt)
		if e != nil {
			return ListImage{}, e
		}
		ua = &t
	}
	var da *time.Time
	if deletedAtStr != nil {
		t, e := parseTime(*deletedAtStr, ErrInvalidDeletedAt)
		if e != nil {
			return ListImage{}, e
		}
		da = &t
	}
	return New(id, listID, u, fileName, size, displayOrder, ca, createdBy, ua, updatedBy, da, deletedBy)
}

// NewFromGCSObject builds public URL from GCS bucket/object and constructs ListImage.
// If bucket is empty, DefaultBucket (narratives_development_list_image) is used.
func NewFromGCSObject(
	id, listID, fileName string,
	size int64,
	displayOrder int,
	bucket string,
	objectPath string,
	createdAt time.Time,
	createdBy string,
	updatedAt *time.Time,
	updatedBy *string,
	deletedAt *time.Time,
	deletedBy *string,
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
	return New(id, listID, publicURL, fileName, size, displayOrder, createdAt, createdBy, updatedAt, updatedBy, deletedAt, deletedBy)
}

// NewMinimalFromGCSObject - minimal constructor using GCS bucket/object.
func NewMinimalFromGCSObject(
	id, listID, fileName string,
	size int64,
	displayOrder int,
	bucket string,
	objectPath string,
	createdAt time.Time,
	createdBy string,
) (ListImage, error) {
	return NewFromGCSObject(id, listID, fileName, size, displayOrder, bucket, objectPath, createdAt, createdBy, nil, nil, nil, nil)
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

func (l *ListImage) TouchUpdated(now time.Time, by *string) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	t := now.UTC()
	l.UpdatedAt = &t
	if by != nil {
		b := strings.TrimSpace(*by)
		if b == "" {
			return ErrInvalidUpdatedBy
		}
		l.UpdatedBy = &b
	}
	return nil
}

func (l *ListImage) MarkDeleted(now time.Time, by *string) error {
	if now.IsZero() {
		return ErrInvalidDeletedAt
	}
	t := now.UTC()
	l.DeletedAt = &t
	if by != nil {
		b := strings.TrimSpace(*by)
		if b == "" {
			return ErrInvalidDeletedBy
		}
		l.DeletedBy = &b
	}
	return nil
}

func (l *ListImage) ClearDeleted() {
	l.DeletedAt = nil
	l.DeletedBy = nil
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
	if l.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(l.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if l.UpdatedAt != nil && (l.UpdatedAt.IsZero() || l.UpdatedAt.Before(l.CreatedAt)) {
		return ErrInvalidUpdatedAt
	}
	if l.UpdatedBy != nil && strings.TrimSpace(*l.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if l.DeletedAt != nil && l.DeletedAt.Before(l.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if l.DeletedBy != nil && strings.TrimSpace(*l.DeletedBy) == "" {
		return ErrInvalidDeletedBy
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

func extAllowed(name string) bool {
	if len(AllowedExtensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := AllowedExtensions[ext]
	return ok
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
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
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

// ToGCSDeleteOp tries to resolve the GCS delete target from this ListImage.
// Priority:
// 1) Parse from URL if it points to storage.googleapis.com/cloud.google.com
// 2) Fallback to DefaultBucket + "list_images/{listID}/{fileName}"
func (l ListImage) ToGCSDeleteOp() GCSDeleteOp {
	if b, obj, ok := ParseGCSURL(l.URL); ok {
		return GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	return GCSDeleteOp{
		Bucket:     DefaultBucket,
		ObjectPath: "list_images/" + strings.TrimSpace(l.ListID) + "/" + strings.TrimSpace(l.FileName),
	}
}

// ========================================
// SQL DDL
// ========================================
const ListImagesTableDDL = `
-- Migration: Initialize ListImage domain
-- Mirrors backend/internal/domain/listImage/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS list_images (
  id             TEXT        PRIMARY KEY,
  list_id        TEXT        NOT NULL,
  url            TEXT        NOT NULL,
  file_name      TEXT        NOT NULL,
  size           BIGINT      NOT NULL CHECK (size >= 0),
  display_order  INT         NOT NULL CHECK (display_order >= 0),

  created_at     TIMESTAMPTZ NOT NULL,
  created_by     TEXT        NOT NULL,
  updated_at     TIMESTAMPTZ NULL,
  updated_by     TEXT        NULL,
  deleted_at     TIMESTAMPTZ NULL,
  deleted_by     TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_list_images_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(list_id)) > 0
    AND char_length(trim(url)) > 0
    AND char_length(trim(file_name)) > 0
    AND char_length(trim(created_by)) > 0
  ),

  -- Time order
  CONSTRAINT chk_list_images_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Prevent duplicate file names per list (optional but useful)
CREATE UNIQUE INDEX IF NOT EXISTS ux_list_images_list_file
  ON list_images (list_id, file_name);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_list_images_list_id        ON list_images (list_id);
CREATE INDEX IF NOT EXISTS idx_list_images_display_order  ON list_images (list_id, display_order);
CREATE INDEX IF NOT EXISTS idx_list_images_created_at     ON list_images (created_at);
CREATE INDEX IF NOT EXISTS idx_list_images_updated_at     ON list_images (updated_at);
CREATE INDEX IF NOT EXISTS idx_list_images_deleted_at     ON list_images (deleted_at);

COMMIT;
`
