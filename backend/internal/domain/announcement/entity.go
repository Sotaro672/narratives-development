// backend\internal\domain\announcement\entity.go
package announcement

import (
	"errors"
	annatt "narratives/internal/domain/announcementAttachment"
	"time"
)

// Domain errors
var (
	ErrInvalidID        = errors.New("announcement: invalid id")
	ErrInvalidTitle     = errors.New("announcement: invalid title")
	ErrInvalidContent   = errors.New("announcement: invalid content")
	ErrInvalidCreatedBy = errors.New("announcement: invalid createdBy")
	ErrInvalidCreatedAt = errors.New("announcement: invalid createdAt")
	ErrInvalidUpdatedAt = errors.New("announcement: invalid updatedAt")
)

// avatars サブコレクション用
type AnnouncementAvatar struct {
	AvatarID string `json:"avatarId"`
	IsRead   bool   `json:"isRead"`
}

// Entity
type Announcement struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	TargetToken *string    `json:"targetToken,omitempty"`
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	Attachments []string   `json:"attachments,omitempty"` // IDs of announcementAttachment
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   *string    `json:"updatedBy,omitempty"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
	DeletedBy   *string    `json:"deletedBy,omitempty"`
}

// Constructors
func New(
	id, title, content string,
	targetToken *string,
	attachments []string,
	published bool,
	createdAt time.Time,
	createdBy string,
	publishedAt, updatedAt, deletedAt *time.Time,
	updatedBy, deletedBy *string,
) (Announcement, error) {
	a := Announcement{
		ID:          id,
		Title:       title,
		Content:     content,
		TargetToken: targetToken,
		Published:   published,
		PublishedAt: publishedAt,
		Attachments: attachments,
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		DeletedAt:   deletedAt,
		DeletedBy:   deletedBy,
	}
	if err := a.validate(); err != nil {
		return Announcement{}, err
	}
	return a, nil
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
	if a.DeletedAt != nil && a.DeletedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.PublishedAt != nil && a.PublishedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// 添付ファイル群から Announcement.Attachments 用の ID 配列を作るヘルパ
//
// 例:
//
//	a.Attachments = AttachmentIDsFromFiles(files)
func AttachmentIDsFromFiles(files []annatt.AttachmentFile) []string {
	ids := make([]string, 0, len(files))
	for _, f := range files {
		id := f.ID
		if id == "" {
			id = annatt.MakeAttachmentID(f.AnnouncementID, f.FileName)
		}
		ids = append(ids, id)
	}
	return ids
}
