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
	// List query.
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Announcement], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Announcement], error)

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
// - Avatar read state is managed by Update.
// - Update may create the avatar record if it does not exist, depending on implementation policy.
type AvatarRepository interface {
	// Query.
	ListByAnnouncementID(ctx context.Context, announcementID string, filter AnnouncementAvatarFilter) ([]AnnouncementAvatar, error)

	// Write.
	Update(ctx context.Context, announcementID string, avatarID string, patch AnnouncementAvatarPatch) (AnnouncementAvatar, error)
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
