// backend\internal\domain\announcement\entity.go
package announcement

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// Domain errors
var (
	ErrInvalidID          = errors.New("announcement: invalid id")
	ErrInvalidTitle       = errors.New("announcement: invalid title")
	ErrInvalidContent     = errors.New("announcement: invalid content")
	ErrInvalidCreatedBy   = errors.New("announcement: invalid createdBy")
	ErrInvalidCreatedAt   = errors.New("announcement: invalid createdAt")
	ErrInvalidUpdatedAt   = errors.New("announcement: invalid updatedAt")
	ErrInvalidPublishedAt = errors.New("announcement: invalid publishedAt")

	ErrInvalidAvatarID = errors.New("announcement avatar: invalid avatarId")
	ErrInvalidReadAt   = errors.New("announcement avatar: invalid readAt")

	ErrInvalidAnnouncementID = errors.New("announcement attachment: invalid announcementId")
	ErrInvalidFileName       = errors.New("announcement attachment: invalid fileName")
	ErrInvalidFileURL        = errors.New("announcement attachment: invalid fileUrl")
	ErrInvalidFileSize       = errors.New("announcement attachment: invalid fileSize")
	ErrInvalidMimeType       = errors.New("announcement attachment: invalid mimeType")
	ErrInvalidObjectPath     = errors.New("announcement attachment: invalid objectPath")
)

// AnnouncementAvatar は announcements/{announcementId}/avatars/{avatarId} subcollection 用の entity。
// Announcement 本体ではなく、avatar ごとの既読状態を表す。
type AnnouncementAvatar struct {
	AnnouncementID string     `json:"announcementId"`
	AvatarID       string     `json:"avatarId"`
	IsRead         bool       `json:"isRead"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

// Entity
type Announcement struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	TargetToken   *string    `json:"targetToken,omitempty"`
	TargetAvatars []string   `json:"targetAvatars,omitempty"`
	Published     bool       `json:"published"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	Attachments   []string   `json:"attachments,omitempty"` // IDs of AnnouncementAttachment
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     string     `json:"createdBy"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
}

// AttachmentFile は Announcement 添付ファイルのメタデータ。
// Firebase Storage の実体操作は frontend 側で行い、backend は metadata のみ保持する。
type AttachmentFile struct {
	AnnouncementID string `json:"announcementId"`
	ID             string `json:"id"`
	FileName       string `json:"fileName"`
	FileURL        string `json:"fileUrl"`
	FileSize       int64  `json:"fileSize"`
	MimeType       string `json:"mimeType"`
	ObjectPath     string `json:"objectPath"`
}

// Constructors
func New(
	id, title, content string,
	targetToken *string,
	targetAvatars []string,
	attachments []string,
	published bool,
	createdAt time.Time,
	createdBy string,
	publishedAt, updatedAt *time.Time,
	updatedBy *string,
) (Announcement, error) {
	a := Announcement{
		ID:            id,
		Title:         title,
		Content:       content,
		TargetToken:   targetToken,
		TargetAvatars: targetAvatars,
		Published:     published,
		PublishedAt:   publishedAt,
		Attachments:   attachments,
		CreatedAt:     createdAt,
		CreatedBy:     createdBy,
		UpdatedAt:     updatedAt,
		UpdatedBy:     updatedBy,
	}
	if err := a.validate(); err != nil {
		return Announcement{}, err
	}
	return a, nil
}

// NewAnnouncementAvatar creates an unread avatar state for
// announcements/{announcementId}/avatars/{avatarId}.
func NewAnnouncementAvatar(
	announcementID string,
	avatarID string,
	createdAt time.Time,
) (AnnouncementAvatar, error) {
	av := AnnouncementAvatar{
		AnnouncementID: announcementID,
		AvatarID:       avatarID,
		IsRead:         false,
		ReadAt:         nil,
		CreatedAt:      createdAt,
		UpdatedAt:      nil,
	}
	if err := av.validate(); err != nil {
		return AnnouncementAvatar{}, err
	}
	return av, nil
}

// NewAnnouncementAvatarWithState creates an avatar state with explicit read state.
// Firestore repository などで永続化済み document から domain entity に戻す場合に使う。
func NewAnnouncementAvatarWithState(
	announcementID string,
	avatarID string,
	isRead bool,
	readAt *time.Time,
	createdAt time.Time,
	updatedAt *time.Time,
) (AnnouncementAvatar, error) {
	av := AnnouncementAvatar{
		AnnouncementID: announcementID,
		AvatarID:       avatarID,
		IsRead:         isRead,
		ReadAt:         readAt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	if err := av.validate(); err != nil {
		return AnnouncementAvatar{}, err
	}
	return av, nil
}

// Validation
func (a Announcement) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.Title == "" {
		return ErrInvalidTitle
	}
	if a.Content == "" {
		return ErrInvalidContent
	}
	if a.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt != nil && a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.PublishedAt != nil && a.PublishedAt.Before(a.CreatedAt) {
		return ErrInvalidPublishedAt
	}
	return nil
}

func (av AnnouncementAvatar) validate() error {
	if av.AnnouncementID == "" {
		return ErrInvalidAnnouncementID
	}
	if av.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if av.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if av.UpdatedAt != nil && av.UpdatedAt.Before(av.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if av.ReadAt != nil && av.ReadAt.Before(av.CreatedAt) {
		return ErrInvalidReadAt
	}
	if av.IsRead && av.ReadAt == nil {
		return ErrInvalidReadAt
	}
	if !av.IsRead && av.ReadAt != nil {
		return ErrInvalidReadAt
	}
	return nil
}

// MarkRead marks this announcement avatar state as read.
// 既読化は冪等に扱う。すでに既読の場合は状態を壊さず、UpdatedAt のみ更新する。
func (av AnnouncementAvatar) MarkRead(now time.Time) (AnnouncementAvatar, error) {
	if now.IsZero() {
		return AnnouncementAvatar{}, ErrInvalidUpdatedAt
	}

	if !av.IsRead {
		av.IsRead = true
		av.ReadAt = &now
	}

	av.UpdatedAt = &now

	if err := av.validate(); err != nil {
		return AnnouncementAvatar{}, err
	}
	return av, nil
}

// MarkUnread marks this announcement avatar state as unread.
// 管理者操作やテスト用途などで未読へ戻す場合に使う。
func (av AnnouncementAvatar) MarkUnread(now time.Time) (AnnouncementAvatar, error) {
	if now.IsZero() {
		return AnnouncementAvatar{}, ErrInvalidUpdatedAt
	}

	av.IsRead = false
	av.ReadAt = nil
	av.UpdatedAt = &now

	if err := av.validate(); err != nil {
		return AnnouncementAvatar{}, err
	}
	return av, nil
}

// IsTargetAvatar returns whether the announcement targets the given avatar directly.
// TargetAvatars が空の場合の全体配信扱いは usecase 側の policy に寄せる。
func (a Announcement) IsTargetAvatar(avatarID string) bool {
	if avatarID == "" {
		return false
	}

	for _, targetAvatarID := range a.TargetAvatars {
		if targetAvatarID == avatarID {
			return true
		}
	}

	return false
}

// ========================================
// Attachment policy
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

// BuildAttachmentObjectPath builds the standard Firebase Storage object path.
// e.g. announcements/{announcementId}/attachments/{fileName}
//
// attachmentID is kept in the signature for backward compatibility,
// but it is not used in the Firebase Storage object path.
// The Firestore attachment document ID is still AttachmentFile.ID.
func BuildAttachmentObjectPath(announcementID, attachmentID, fileName string) (string, error) {
	if announcementID == "" {
		return "", ErrInvalidAnnouncementID
	}
	if fileName == "" {
		return "", ErrInvalidFileName
	}

	return path.Join("announcements", announcementID, "attachments", fileName), nil
}

// MakeAttachmentID は announcementId と fileName から安定IDを生成します。
// 形式: hex(sha1(lower(trim(announcementId))+":"+trim(fileName)))
func MakeAttachmentID(announcementID, fileName string) string {
	aid := strings.ToLower(strings.TrimSpace(announcementID))
	fn := strings.TrimSpace(fileName)

	h := sha1.Sum([]byte(aid + ":" + fn))
	return hex.EncodeToString(h[:])
}

// NewAttachmentFile creates AttachmentFile metadata.
// Firebase Storage upload itself is handled by frontend Firebase Storage SDK.
func NewAttachmentFile(
	announcementID, fileName, fileURL string,
	fileSize int64,
	mimeType string,
) (AttachmentFile, error) {
	id := MakeAttachmentID(announcementID, fileName)

	objectPath, err := BuildAttachmentObjectPath(announcementID, id, fileName)
	if err != nil {
		return AttachmentFile{}, err
	}

	f := AttachmentFile{
		AnnouncementID: announcementID,
		ID:             id,
		FileName:       fileName,
		FileURL:        fileURL,
		FileSize:       fileSize,
		MimeType:       mimeType,
		ObjectPath:     objectPath,
	}
	if err := validateAttachmentFile(f); err != nil {
		return AttachmentFile{}, err
	}

	return f, nil
}

// NewAttachmentFileWithObjectPath creates AttachmentFile metadata using frontend-generated Firebase Storage metadata.
func NewAttachmentFileWithObjectPath(
	announcementID, id, fileName, fileURL string,
	fileSize int64,
	mimeType, objectPath string,
) (AttachmentFile, error) {
	if id == "" {
		id = MakeAttachmentID(announcementID, fileName)
	}
	if objectPath == "" {
		p, err := BuildAttachmentObjectPath(announcementID, id, fileName)
		if err != nil {
			return AttachmentFile{}, err
		}
		objectPath = p
	}

	f := AttachmentFile{
		AnnouncementID: announcementID,
		ID:             id,
		FileName:       fileName,
		FileURL:        fileURL,
		FileSize:       fileSize,
		MimeType:       mimeType,
		ObjectPath:     objectPath,
	}
	if err := validateAttachmentFile(f); err != nil {
		return AttachmentFile{}, err
	}

	return f, nil
}

func validateAttachmentFile(f AttachmentFile) error {
	if f.AnnouncementID == "" {
		return ErrInvalidAnnouncementID
	}
	if f.FileName == "" || (MaxFileNameLength > 0 && len([]rune(f.FileName)) > MaxFileNameLength) {
		return ErrInvalidFileName
	}
	if f.ID == "" {
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
	if f.ObjectPath == "" {
		return ErrInvalidObjectPath
	}

	expected, err := BuildAttachmentObjectPath(f.AnnouncementID, f.ID, f.FileName)
	if err != nil {
		return err
	}
	if strings.TrimLeft(f.ObjectPath, "/") != expected {
		return ErrInvalidObjectPath
	}

	return nil
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

// 添付ファイル群から Announcement.Attachments 用の ID 配列を作るヘルパ
//
// 例:
//
//	a.Attachments = AttachmentIDsFromFiles(files)
func AttachmentIDsFromFiles(files []AttachmentFile) []string {
	ids := make([]string, 0, len(files))
	for _, f := range files {
		if f.ID == "" {
			continue
		}
		ids = append(ids, f.ID)
	}
	return ids
}
