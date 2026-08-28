// backend/internal/application/port/announcement_attachment_storage.go
package port

import "context"

// AnnouncementAttachmentStorage manages storage objects associated with an announcement.
type AnnouncementAttachmentStorage interface {
	DeleteAll(
		ctx context.Context,
		announcementID string,
	) error
}
