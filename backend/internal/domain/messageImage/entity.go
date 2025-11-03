package messageImage

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	msgdom "narratives/internal/domain/message"
)

// GCS bucket (env で切り替える場合はアプリ層で差し込んでください)
const DefaultBucket = "narratives_development_message_image"

// BuildObjectPath は GCS 上の標準オブジェクトキーを組み立てます。
// 例: messages/{messageId}/{fileName}
func BuildObjectPath(messageID, fileName string) (string, error) {
	messageID = strings.TrimSpace(messageID)
	fileName = strings.TrimSpace(fileName)
	if messageID == "" {
		return "", ErrInvalidMessageID
	}
	if !isSafeFileName(fileName) {
		return "", ErrInvalidFileName
	}
	// フォルダ風の prefix を固定（将来変更する場合は一箇所で差し替え）
	return path.Join("messages", messageID, fileName), nil
}

// ImageFile mirrors web-app/src/shared/types/messageImage.ts (TS source of truth):
// export interface ImageFile {
//   messageId: string;
//   fileName: string;
//   fileUrl: string;
//   fileSize: number;
//   mimeType: string;
//   width?: number;
//   height?: number;
//   createdAt: Date | string;
//   updatedAt?: Date | string;
//   deletedAt?: Date | string;
// }
type ImageFile struct {
	MessageID  string
	FileName   string
	// FileURL は任意（署名付き URL 等）。未設定の場合は PublicURL() で生成可能。
	FileURL    string
	FileSize   int64
	MimeType   string
	Width      *int
	Height     *int

	// GCS 配置情報
	Bucket     string // 例: narratives_development_message_image
	ObjectPath string // 例: messages/{messageId}/{fileName}

	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

var (
	// Validation errors
	ErrInvalidMessageID  = errors.New("messageImage: invalid messageId")
	ErrInvalidFileName   = errors.New("messageImage: invalid fileName")
	ErrInvalidFileURL    = errors.New("messageImage: invalid fileUrl")
	ErrInvalidFileSize   = errors.New("messageImage: invalid fileSize")
	ErrInvalidMimeType   = errors.New("messageImage: invalid mimeType")
	ErrInvalidCreatedAt  = errors.New("messageImage: invalid createdAt")
	ErrInvalidUpdatedAt  = errors.New("messageImage: invalid updatedAt")
	ErrInvalidDeletedAt  = errors.New("messageImage: invalid deletedAt")
	ErrInvalidDimensions = errors.New("messageImage: invalid image dimensions")
	ErrInvalidBucket     = errors.New("messageImage: invalid bucket")
	ErrInvalidObjectPath = errors.New("messageImage: invalid objectPath")
)

// Policy (align with messageImageConstants.ts if exists)
var (
	AllowedMIMEs = map[string]struct{}{
		"image/png":  {},
		"image/jpeg": {},
		"image/webp": {},
		"image/gif":  {},
	}
	MaxFileSize int64 = 10 * 1024 * 1024 // 10MB; set 0 to disable check
	MinWidth          = 1
	MinHeight         = 1
	MaxWidth          = 10000
	MaxHeight         = 10000
)

// ========================================
// Constructors
// ========================================

// 既存の呼び出し互換: デフォルトバケット + 規定の ObjectPath を使用
func NewImageFile(
	messageID string,
	fileName string,
	fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	updatedAt, deletedAt *time.Time,
) (ImageFile, error) {
	objPath, err := BuildObjectPath(messageID, fileName)
	if err != nil {
		return ImageFile{}, err
	}
	img := ImageFile{
		MessageID:  strings.TrimSpace(messageID),
		FileName:   strings.TrimSpace(fileName),
		FileURL:    strings.TrimSpace(fileURL),
		FileSize:   fileSize,
		MimeType:   strings.TrimSpace(mimeType),
		Width:      width,
		Height:     height,
		Bucket:     DefaultBucket,
		ObjectPath: objPath,

		CreatedAt: createdAt.UTC(),
		UpdatedAt: normalizeTimePtr(updatedAt),
		DeletedAt: normalizeTimePtr(deletedAt),
	}
	if err := img.validate(); err != nil {
		return ImageFile{}, err
	}
	return img, nil
}

// バケットを明示するコンストラクタ
func NewImageFileWithBucket(
	bucket string,
	messageID string,
	fileName string,
	fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	updatedAt, deletedAt *time.Time,
) (ImageFile, error) {
	if strings.TrimSpace(bucket) == "" {
		bucket = DefaultBucket
	}
	objPath, err := BuildObjectPath(messageID, fileName)
	if err != nil {
		return ImageFile{}, err
	}
	img := ImageFile{
		MessageID:  strings.TrimSpace(messageID),
		FileName:   strings.TrimSpace(fileName),
		FileURL:    strings.TrimSpace(fileURL),
		FileSize:   fileSize,
		MimeType:   strings.TrimSpace(mimeType),
		Width:      width,
		Height:     height,
		Bucket:     strings.TrimSpace(bucket),
		ObjectPath: objPath,

		CreatedAt: createdAt.UTC(),
		UpdatedAt: normalizeTimePtr(updatedAt),
		DeletedAt: normalizeTimePtr(deletedAt),
	}
	if err := img.validate(); err != nil {
		return ImageFile{}, err
	}
	return img, nil
}

func NewImageFileFromStringTimes(
	messageID string,
	fileName string,
	fileURL string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt string,
	updatedAt, deletedAt *string,
) (ImageFile, error) {
	ca, err := parseTime(createdAt)
	if err != nil {
		return ImageFile{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	var ua, da *time.Time
	if updatedAt != nil && strings.TrimSpace(*updatedAt) != "" {
		t, e := parseTime(*updatedAt)
		if e != nil {
			return ImageFile{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, e)
		}
		ua = &t
	}
	if deletedAt != nil && strings.TrimSpace(*deletedAt) != "" {
		t, e := parseTime(*deletedAt)
		if e != nil {
			return ImageFile{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, e)
		}
		da = &t
	}
	return NewImageFile(messageID, fileName, fileURL, fileSize, mimeType, width, height, ca, ua, da)
}

// ========================================
// Mutators
// ========================================

func (i *ImageFile) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if u == "" {
		// FileURL は任意。空にすることも許容。
		i.FileURL = ""
		return nil
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		return ErrInvalidFileURL
	}
	// 署名付き URL/公開 URL いずれも bucket 名を含んでいる前提（厳密チェックはアプリ層で）
	i.FileURL = u
	return nil
}

func (i *ImageFile) UpdateDimensions(width, height *int) error {
	if width != nil {
		if *width < MinWidth || *width > MaxWidth {
			return ErrInvalidDimensions
		}
	}
	if height != nil {
		if *height < MinHeight || *height > MaxHeight {
			return ErrInvalidDimensions
		}
	}
	i.Width, i.Height = width, height
	return nil
}

func (i *ImageFile) TouchUpdated(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	t := now.UTC()
	i.UpdatedAt = &t
	return nil
}

func (i *ImageFile) MarkDeleted(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidDeletedAt
	}
	t := now.UTC()
	i.DeletedAt = &t
	return nil
}

func (i *ImageFile) ClearDeleted() {
	i.DeletedAt = nil
}

// GCSURI は gs://{bucket}/{objectPath} を返します。
func (i ImageFile) GCSURI() string {
	b := strings.TrimSpace(i.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return fmt.Sprintf("gs://%s/%s", b, strings.TrimLeft(i.ObjectPath, "/"))
}

// PublicURL は https://storage.googleapis.com/{bucket}/{objectPath} を返します。
// 既に FileURL が設定されていればそちらを優先します（署名付き URL 等）。
func (i ImageFile) PublicURL() string {
	if strings.TrimSpace(i.FileURL) != "" {
		return i.FileURL
	}
	b := strings.TrimSpace(i.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, strings.TrimLeft(i.ObjectPath, "/"))
}

// ========================================
// Validation
// ========================================

func (i ImageFile) validate() error {
	if strings.TrimSpace(i.MessageID) == "" {
		return ErrInvalidMessageID
	}
	if !isSafeFileName(i.FileName) {
		return ErrInvalidFileName
	}
	// FileURL は任意（署名付きの場合があるため）。設定されている場合のみ検証。
	if strings.TrimSpace(i.FileURL) != "" {
		if _, err := url.ParseRequestURI(i.FileURL); err != nil {
			return ErrInvalidFileURL
		}
	}
	if i.FileSize < 0 {
		return ErrInvalidFileSize
	}
	if MaxFileSize > 0 && i.FileSize > MaxFileSize {
		return ErrInvalidFileSize
	}
	if !isAllowedMIME(i.MimeType) {
		return ErrInvalidMimeType
	}
	if i.Width != nil {
		if *i.Width < MinWidth || *i.Width > MaxWidth {
			return ErrInvalidDimensions
		}
	}
	if i.Height != nil {
		if *i.Height < MinHeight || *i.Height > MaxHeight {
			return ErrInvalidDimensions
		}
	}
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if i.UpdatedAt != nil && i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if i.DeletedAt != nil && i.DeletedAt.Before(i.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	// GCS 関連の検証
	if strings.TrimSpace(i.Bucket) == "" {
		return ErrInvalidBucket
	}
	if strings.TrimSpace(i.ObjectPath) == "" {
		return ErrInvalidObjectPath
	}
	// 期待フォーマット: messages/{messageId}/{fileName}
	expect, _ := BuildObjectPath(i.MessageID, i.FileName)
	if strings.TrimLeft(i.ObjectPath, "/") != expect {
		return ErrInvalidObjectPath
	}
	return nil
}

func isAllowedMIME(mt string) bool {
	if mt == "" {
		return false
	}
	if len(AllowedMIMEs) == 0 {
		// accept typical "type/subtype" if policy not configured
		return strings.Count(mt, "/") == 1
	}
	_, ok := AllowedMIMEs[mt]
	return ok
}

func isSafeFileName(fn string) bool {
	if fn == "" {
		return false
	}
	// パス要素やディレクトリトラバーサルは禁止
	if strings.Contains(fn, "/") || strings.Contains(fn, "\\") || strings.Contains(fn, "..") {
		return false
	}
	// 先頭・末尾の空白は除外（上位で Trim される想定）
	return strings.TrimSpace(fn) == fn
}

// ========================================
// Helpers
// ========================================

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrInvalidCreatedAt
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

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

// ========================================
// SQL DDL (任意: 参照情報のみを RDB に保持する場合の雛形)
// ========================================
const MessageImagesTableDDL = `
-- Migration: Initialize message_images table

BEGIN;

CREATE TABLE IF NOT EXISTS message_images (
  message_id     UUID        NOT NULL,
  file_name      TEXT        NOT NULL,
  file_url       TEXT        NOT NULL,
  file_size      BIGINT      NOT NULL CHECK (file_size >= 0),
  mime_type      TEXT        NOT NULL,
  width          INT         NULL CHECK (width IS NULL OR (width >= 1 AND width <= 10000)),
  height         INT         NULL CHECK (height IS NULL OR (height IS NULL) OR (height >= 1 AND height <= 10000)),
  created_at     TIMESTAMPTZ NOT NULL,
  updated_at     TIMESTAMPTZ NULL,
  deleted_at     TIMESTAMPTZ NULL,

  -- 参照情報（GCS）
  bucket         TEXT        NOT NULL,
  object_path    TEXT        NOT NULL,

  CONSTRAINT pk_message_images PRIMARY KEY (message_id, file_name),

  CONSTRAINT chk_message_images_non_empty CHECK (
    char_length(trim(file_name)) > 0
    AND char_length(trim(file_url)) > 0
    AND char_length(trim(mime_type)) > 0
    AND char_length(trim(bucket)) > 0
    AND char_length(trim(object_path)) > 0
  ),

  CONSTRAINT chk_message_images_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

CREATE INDEX IF NOT EXISTS idx_message_images_message_id  ON message_images (message_id);
CREATE INDEX IF NOT EXISTS idx_message_images_created_at  ON message_images (created_at);
CREATE INDEX IF NOT EXISTS idx_message_images_deleted_at  ON message_images (deleted_at);

COMMIT;
`

// ========================================
// Cascade delete helpers (Message -> MessageImage)
// ========================================

// GCS 上の削除対象を表現するドメイン意図
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ToGCSDeleteOp はこの画像に対応する GCS 削除オペレーションを返します。
func (i ImageFile) ToGCSDeleteOp() GCSDeleteOp {
	b := strings.TrimSpace(i.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return GCSDeleteOp{
		Bucket:     b,
		ObjectPath: strings.TrimLeft(i.ObjectPath, "/"),
	}
}

// BuildGCSDeleteOps はメッセージに紐づく全画像の削除オペレーションを生成します。
func BuildGCSDeleteOps(images []ImageFile) []GCSDeleteOp {
	ops := make([]GCSDeleteOp, 0, len(images))
	for _, img := range images {
		ops = append(ops, img.ToGCSDeleteOp())
	}
	return ops
}

// PrepareCascadeDelete は連携削除前にソフトデリートの印を付与します（任意）。
func PrepareCascadeDelete(images []ImageFile, now time.Time) []ImageFile {
	out := make([]ImageFile, len(images))
	copy(out, images)
	for idx := range out {
		_ = out[idx].MarkDeleted(now) // 時刻検証のみ、エラーは上位で扱う
	}
	return out
}

// BuildGCSDeleteOpsFromMessage は Message から ImageRef を読み取り、
// GCS の削除オペレーションを生成します（ObjectPath が空なら規約で補完）。
// アプリケーション層で Message.MarkDeleted の後に呼び出して、GCS 削除バッチへ渡してください。
func BuildGCSDeleteOpsFromMessage(m msgdom.Message) []GCSDeleteOp {
	return BuildGCSDeleteOpsFromRefs(m.ID, m.Images)
}

// BuildGCSDeleteOpsFromRefs は messageID と ImageRef 群から GCS 削除オペレーションを生成します。
func BuildGCSDeleteOpsFromRefs(messageID string, refs []msgdom.ImageRef) []GCSDeleteOp {
	ops := make([]GCSDeleteOp, 0, len(refs))
	for _, ref := range refs {
		op := GCSDeleteOp{Bucket: DefaultBucket}
		obj := strings.TrimLeft(strings.TrimSpace(ref.ObjectPath), "/")
		if obj == "" {
			// 参照に ObjectPath が無い場合は規約パスで補完
			if p, err := BuildObjectPath(messageID, ref.FileName); err == nil {
				obj = p
			} else {
				// 不正な参照はスキップ（必要ならログ/エラー収集は上位層で）
				continue
			}
		}
		op.ObjectPath = obj
		ops = append(ops, op)
	}
	return ops
}
