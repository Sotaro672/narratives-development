// backend\internal\domain\announcementAttachment\entity.go
package announcementAttachment

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// GCS bucket where announcement attachments are stored.
const DefaultBucket = "narratives_development_announcement_attachment"

// BuildObjectPath builds the standard GCS object path for an attachment.
// e.g. announcements/{announcementId}/{fileName}
func BuildObjectPath(announcementID, fileName string) (string, error) {
	aid := strings.TrimSpace(announcementID)
	fn := strings.TrimSpace(fileName)
	if aid == "" {
		return "", ErrInvalidAnnouncementID
	}
	if fn == "" {
		return "", ErrInvalidFileName
	}
	// Directory-like prefix for clarity and future-proofing.
	return path.Join("announcements", aid, fn), nil
}

// ========================================
// Types (mirror TS)
// export interface AttachmentFile {
//   announcementId: string;
//   id: string;          // 追加: 安定ID（announcementId + fileName から決定的に生成）
//   fileName: string;
//   fileUrl: string;
//   fileSize: number;
//   mimeType: string;
//   // GCS 参照情報（追加）
//   bucket: string;
//   objectPath: string;
// }
// ========================================

type AttachmentFile struct {
	AnnouncementID string `json:"announcementId"`
	ID             string `json:"id"` // 追加: 参照用ID（Announcement.Attachments はこのIDを列挙）
	FileName       string `json:"fileName"`
	FileURL        string `json:"fileUrl"`
	FileSize       int64  `json:"fileSize"`
	MimeType       string `json:"mimeType"`

	// GCS placement (bucket/object key)
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"objectPath"`
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidAnnouncementID = errors.New("announcementAttachment: invalid announcementId")
	ErrInvalidFileName       = errors.New("announcementAttachment: invalid fileName")
	ErrInvalidFileURL        = errors.New("announcementAttachment: invalid fileUrl")
	ErrInvalidFileSize       = errors.New("announcementAttachment: invalid fileSize")
	ErrInvalidMimeType       = errors.New("announcementAttachment: invalid mimeType")
	ErrInvalidID             = errors.New("announcementAttachment: invalid id")
	ErrInvalidBucket         = errors.New("announcementAttachment: invalid bucket")
	ErrInvalidObjectPath     = errors.New("announcementAttachment: invalid objectPath")
)

// ========================================
// Policy
// ========================================

var (
	// Limits (0 disables upper checks)
	MinFileSizeBytes  int64 = 1
	MaxFileSizeBytes  int64 = 50 * 1024 * 1024 // 50MB
	MaxFileNameLength       = 255

	// Allowed MIME types (empty map = allow all matching mimeRe)
	AllowedMimeTypes = map[string]struct{}{
		"application/pdf": {},
		"image/jpeg":      {},
		"image/png":       {},
		"image/webp":      {},
		"image/gif":       {},
		"text/plain":      {},
	}

	// Optional allow-list for URL hosts (empty = allow all)
	AllowedURLHosts = map[string]struct{}{}

	// MIME validation (nil disables)
	mimeRe = regexp.MustCompile(`^[a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+$`)
)

// ========================================
// ID helper
// ========================================

// MakeAttachmentID は announcementId と fileName から安定IDを生成します。
// 形式: hex(sha1(lower(trim(announcementId))+":"+trim(fileName)))
func MakeAttachmentID(announcementID, fileName string) string {
	aid := strings.ToLower(strings.TrimSpace(announcementID))
	fn := strings.TrimSpace(fileName)
	h := sha1.Sum([]byte(aid + ":" + fn))
	return hex.EncodeToString(h[:])
}

// ========================================
// Constructors
// ========================================

// NewAttachmentFile creates an AttachmentFile and sets its GCS bucket/objectPath
// to the default bucket and the standard object path.
func NewAttachmentFile(
	announcementID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
) (AttachmentFile, error) {
	announcementID = strings.TrimSpace(announcementID)
	fileName = strings.TrimSpace(fileName)

	objPath, err := BuildObjectPath(announcementID, fileName)
	if err != nil {
		return AttachmentFile{}, err
	}

	f := AttachmentFile{
		AnnouncementID: announcementID,
		ID:             MakeAttachmentID(announcementID, fileName),
		FileName:       fileName,
		FileURL:        strings.TrimSpace(fileURL),
		FileSize:       fileSize,
		MimeType:       strings.TrimSpace(mimeType),

		Bucket:     DefaultBucket,
		ObjectPath: objPath,
	}
	if err := validateAttachmentFile(f); err != nil {
		return AttachmentFile{}, err
	}
	return f, nil
}

// NewAttachmentFileWithBucket lets callers override the bucket (falls back to DefaultBucket if empty).
func NewAttachmentFileWithBucket(
	bucket string,
	announcementID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
) (AttachmentFile, error) {
	announcementID = strings.TrimSpace(announcementID)
	fileName = strings.TrimSpace(fileName)

	if strings.TrimSpace(bucket) == "" {
		bucket = DefaultBucket
	}
	objPath, err := BuildObjectPath(announcementID, fileName)
	if err != nil {
		return AttachmentFile{}, err
	}

	f := AttachmentFile{
		AnnouncementID: announcementID,
		ID:             MakeAttachmentID(announcementID, fileName),
		FileName:       fileName,
		FileURL:        strings.TrimSpace(fileURL),
		FileSize:       fileSize,
		MimeType:       strings.TrimSpace(mimeType),

		Bucket:     strings.TrimSpace(bucket),
		ObjectPath: objPath,
	}
	if err := validateAttachmentFile(f); err != nil {
		return AttachmentFile{}, err
	}
	return f, nil
}

// ========================================
// Validation
// ========================================

func validateAttachmentFile(f AttachmentFile) error {
	if f.AnnouncementID == "" {
		return ErrInvalidAnnouncementID
	}
	if f.FileName == "" || (MaxFileNameLength > 0 && len([]rune(f.FileName)) > MaxFileNameLength) {
		return ErrInvalidFileName
	}
	// ID は決定的に生成される前提。フィールドと整合するか確認。
	if f.ID == "" || f.ID != MakeAttachmentID(f.AnnouncementID, f.FileName) {
		return ErrInvalidID
	}
	if !urlOK(f.FileURL) {
		return ErrInvalidFileURL
	}
	if f.FileSize < MinFileSizeBytes || (MaxFileSizeBytes > 0 && f.FileSize > MaxFileSizeBytes) {
		return ErrInvalidFileSize
	}
	if f.MimeType == "" || (mimeRe != nil && !mimeRe.MatchString(f.MimeType)) {
		return ErrInvalidMimeType
	}
	if len(AllowedMimeTypes) > 0 {
		if _, ok := AllowedMimeTypes[f.MimeType]; !ok {
			return ErrInvalidMimeType
		}
	}
	// GCS 参照の検証
	if strings.TrimSpace(f.Bucket) == "" {
		return ErrInvalidBucket
	}
	if strings.TrimSpace(f.ObjectPath) == "" {
		return ErrInvalidObjectPath
	}
	expect, _ := BuildObjectPath(f.AnnouncementID, f.FileName)
	if strings.TrimLeft(f.ObjectPath, "/") != expect {
		return ErrInvalidObjectPath
	}
	return nil
}

// ========================================
// Helpers
// ========================================

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

// GCSURI returns gs://{bucket}/{objectPath}
func (f AttachmentFile) GCSURI() string {
	b := strings.TrimSpace(f.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return fmt.Sprintf("gs://%s/%s", b, strings.TrimLeft(f.ObjectPath, "/"))
}

// PublicURL returns https://storage.googleapis.com/{bucket}/{objectPath}.
// If FileURL is already set (e.g., signed URL), that should be preferred by callers.
func (f AttachmentFile) PublicURL() string {
	b := strings.TrimSpace(f.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, strings.TrimLeft(f.ObjectPath, "/"))
}

// ========================================
// Cascade delete helpers (Announcement -> AnnouncementAttachment)
// ========================================

// GCSDeleteOp represents a single delete operation in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ToGCSDeleteOp converts this attachment to a GCS delete operation.
func (f AttachmentFile) ToGCSDeleteOp() GCSDeleteOp {
	b := strings.TrimSpace(f.Bucket)
	if b == "" {
		b = DefaultBucket
	}
	return GCSDeleteOp{
		Bucket:     b,
		ObjectPath: strings.TrimLeft(strings.TrimSpace(f.ObjectPath), "/"),
	}
}

// BuildGCSDeleteOps builds delete operations for all provided attachments.
// Announcement 削除時は、まず添付メタデータを取得してから本関数で GCS 削除対象を組み立ててください。
func BuildGCSDeleteOps(files []AttachmentFile) []GCSDeleteOp {
	ops := make([]GCSDeleteOp, 0, len(files))
	for _, af := range files {
		op := af.ToGCSDeleteOp()
		if op.Bucket == "" || op.ObjectPath == "" {
			continue
		}
		ops = append(ops, op)
	}
	return ops
}

// BuildGCSDeleteOpsFromFileNames can be used when you only have file names and announcementID.
// Note: If your Announcement entity stores only attachment IDs,メタデータ解決後に使用してください。
func BuildGCSDeleteOpsFromFileNames(announcementID string, fileNames []string) []GCSDeleteOp {
	out := make([]GCSDeleteOp, 0, len(fileNames))
	aid := strings.TrimSpace(announcementID)
	if aid == "" {
		return out
	}
	for _, fn := range fileNames {
		fn = strings.TrimSpace(fn)
		if fn == "" {
			continue
		}
		if p, err := BuildObjectPath(aid, fn); err == nil {
			out = append(out, GCSDeleteOp{
				Bucket:     DefaultBucket,
				ObjectPath: p,
			})
		}
	}
	return out
}

// ========================================
// SQL DDL
// ========================================

// AnnouncementAttachmentsTableDDL defines the SQL for the announcement_attachments table.
// 既存スキーマ互換のため、DB 上は複合PKのままでも可（ID はアプリ側で決定的生成）。
// 新規に ID カラムを持たせる場合は PRIMARY KEY を id に、(announcement_id,file_name) は UNIQUE に変更してください。
const AnnouncementAttachmentsTableDDL = `
CREATE TABLE IF NOT EXISTS announcement_attachments (
  announcement_id TEXT NOT NULL,
  file_name TEXT NOT NULL,
  file_url TEXT NOT NULL,
  file_size BIGINT NOT NULL CHECK (file_size > 0 AND file_size <= 52428800),
  mime_type TEXT NOT NULL CHECK (
    mime_type IN ('application/pdf','image/jpeg','image/png','image/webp','image/gif','text/plain')
  ),
  -- GCS references
  bucket TEXT NOT NULL,
  object_path TEXT NOT NULL,

  -- Use a composite primary key since there is no standalone id in the persisted schema
  CONSTRAINT pk_announcement_attachments PRIMARY KEY (announcement_id, file_name),

  -- Basic URL sanity check (http/https)
  CHECK (file_url ~* '^https?://'),

  -- Basic non-empty checks for GCS references
  CHECK (char_length(trim(bucket)) > 0 AND char_length(trim(object_path)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_announcement_attachments_announcement_id
  ON announcement_attachments (announcement_id);

CREATE INDEX IF NOT EXISTS idx_announcement_attachments_mime_type
  ON announcement_attachments (mime_type);
`
