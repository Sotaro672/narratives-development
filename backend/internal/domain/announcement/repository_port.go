// backend/internal/domain/announcement/repository_port.go
package announcement

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// AnnouncementPatch is a partial update input for announcement records.
// nil fields are not updated.
type AnnouncementPatch struct {
	Title       *string
	Content     *string
	TargetToken *string
	Published   *bool
	PublishedAt *time.Time
	Attachments *[]string

	UpdatedAt *time.Time
	UpdatedBy *string
}

// AnnouncementAvatarPatch is a partial update input for announcement avatar records.
// nil fields are not updated.
type AnnouncementAvatarPatch struct {
	IsRead *bool
	ReadAt *time.Time

	UpdatedAt *time.Time
}

// AttachmentFilePatch is a partial update input for attachment file records.
// nil fields are not updated.
type AttachmentFilePatch struct {
	FileURL    *string
	FileSize   *int64
	MimeType   *string
	ObjectPath *string

	UpdatedAt *time.Time
}

// Filter is the query condition for Announcement repository.
type Filter struct {
	TargetToken *string

	// Published status.
	Published *bool

	// Date range.
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	UpdatedFrom   *time.Time
	UpdatedTo     *time.Time
	PublishedFrom *time.Time
	PublishedTo   *time.Time
}

// AnnouncementAvatarFilter is the query condition for AnnouncementAvatar repository.
type AnnouncementAvatarFilter struct {
	AnnouncementID  *string
	AnnouncementIDs []string

	AvatarIDs []string
	IsRead    *bool
}

// AttachmentFilter is the query condition for AttachmentFile repository.
//
// NOTE:
// - Attachment file records are scoped by announcementId.
// - fileName alone should not be used as a global lookup key.
type AttachmentFilter struct {
	AnnouncementID  *string
	AnnouncementIDs []string

	FileNames []string
	MimeTypes []string

	SizeMin *int64
	SizeMax *int64
}

// Common type aliases.
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// Contract errors.
var (
	ErrNotFound = errors.New("announcement: not found")
	ErrConflict = errors.New("announcement: conflict")
)

// Repository is the repository port for Announcement aggregate root.
type Repository interface {
	// ListByTargetToken returns announcements whose targetToken equals tokenBlueprintID.
	//
	// Expected implementation policy:
	// - tokenBlueprintID is compared with Announcement.TargetToken.
	// - tokenBlueprintID should be treated as tokenBlueprintId from tokenBlueprint domain.
	// - Empty tokenBlueprintID should return ErrInvalidID or another validation error.
	// - Result should be paginated by page.
	ListByTargetToken(ctx context.Context, tokenBlueprintID string, page Page) (PageResult[Announcement], error)

	// ListByTargetAvatar returns announcements whose targetAvatars contains avatarID.
	//
	// Expected implementation policy:
	// - avatarID is the logged-in/current avatar id.
	// - avatarID is compared with Announcement.TargetAvatars.
	// - Empty avatarID should return ErrInvalidAvatarID or another validation error.
	// - Result should be paginated by page.
	// - In Firestore, this is usually implemented with:
	//   Where("targetAvatars", "array-contains", avatarID)
	ListByTargetAvatar(ctx context.Context, avatarID string, page Page) (PageResult[Announcement], error)

	// Read.
	GetByID(ctx context.Context, id string) (Announcement, error)

	// Write.
	Create(ctx context.Context, a Announcement) (Announcement, error)

	// Update replaces/upserts the mutable fields of a persisted Announcement.
	//
	// Expected implementation policy:
	// - id is the target document id.
	// - a.ID may be empty or equal to id.
	// - immutable fields such as CreatedAt and CreatedBy should not be overwritten
	//   unless the implementation intentionally treats Update as full replacement.
	Update(ctx context.Context, id string, a Announcement) (Announcement, error)

	// MarkPublished marks an announcement as published.
	//
	// Expected implementation policy:
	// - id is the target announcement document id.
	// - It should set published=true.
	// - It should set publishedAt=publishedAt.
	// - It should set updatedAt=publishedAt.
	// - It should set updatedBy=updatedBy when updatedBy is not nil.
	// - It should return the updated Announcement.
	// - Empty id should return ErrInvalidID or another validation error.
	// - Zero publishedAt should return ErrInvalidPublishedAt or another validation error.
	MarkPublished(ctx context.Context, id string, publishedAt time.Time, updatedBy *string) (Announcement, error)

	// Delete physically deletes an announcement document.
	// Implementations may also delete child avatar and attachment records
	// if the storage supports subcollections.
	Delete(ctx context.Context, id string) error
}

// AvatarRepository is the repository port for announcement avatar records.
//
// Avatar policy:
// - Avatar record is scoped by announcementId.
// - avatarID alone should not be used as a global lookup key.
// - Avatar read state is managed by MarkRead / Update.
// - MarkRead should be idempotent.
// - Update may create the avatar record if it does not exist, depending on implementation policy.
type AvatarRepository interface {
	// Query.
	ListByAnnouncementID(ctx context.Context, announcementID string, filter AnnouncementAvatarFilter) ([]AnnouncementAvatar, error)

	// Write.
	Update(ctx context.Context, announcementID string, avatarID string, patch AnnouncementAvatarPatch) (AnnouncementAvatar, error)

	// MarkRead marks announcements/{announcementId}/avatars/{avatarId} as read.
	//
	// Expected implementation policy:
	// - announcementID is the parent announcement document id.
	// - avatarID is the avatar subcollection document id.
	// - It should set isRead=true.
	// - It should set readAt=readAt.
	// - It should set updatedAt=readAt.
	// - It should be idempotent; calling it multiple times should not create an invalid state.
	// - It may create the avatar record if it does not exist, depending on implementation policy.
	MarkRead(ctx context.Context, announcementID string, avatarID string, readAt time.Time) (AnnouncementAvatar, error)
}

// AttachmentRepository is the repository port for announcement attachment file metadata.
//
// Attachment policy:
// - Frontend manages actual Firebase Storage objects.
// - Backend stores only attachment metadata.
// - Attachment record is scoped by announcementId.
// - fileName alone should not be used as a global lookup key.
type AttachmentRepository interface {
	// Query.
	ListByAnnouncementID(ctx context.Context, announcementID string) ([]AttachmentFile, error)

	// Write.
	Create(ctx context.Context, f AttachmentFile) (AttachmentFile, error)
	Update(ctx context.Context, announcementID string, fileName string, patch AttachmentFilePatch) (AttachmentFile, error)

	// Delete physically deletes an attachment metadata record.
	Delete(ctx context.Context, announcementID string, fileName string) error
}
