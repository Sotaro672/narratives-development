// backend/internal/adapters/out/firestore/announcement_attachment_repository_fs.go
package firestore

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	announcement "narratives/internal/domain/announcement"
)

// Firestore implementation of announcement.AttachmentRepository.
type AnnouncementAttachmentRepositoryFS struct {
	Client *firestore.Client
}

func NewAnnouncementAttachmentRepositoryFS(client *firestore.Client) *AnnouncementAttachmentRepositoryFS {
	return &AnnouncementAttachmentRepositoryFS{Client: client}
}

// Compile-time check.
var _ announcement.AttachmentRepository = (*AnnouncementAttachmentRepositoryFS)(nil)

func attachmentCollection(client *firestore.Client, announcementID string) *firestore.CollectionRef {
	return announcementDoc(client, announcementID).Collection("attachments")
}

func attachmentDoc(client *firestore.Client, announcementID string, fileName string) *firestore.DocumentRef {
	return attachmentCollection(client, announcementID).
		Doc(announcement.MakeAttachmentID(announcementID, fileName))
}

// ListByAnnouncementID retrieves all attachment metadata documents for one announcement.
func (r *AnnouncementAttachmentRepositoryFS) ListByAnnouncementID(
	ctx context.Context,
	announcementID string,
) ([]announcement.AttachmentFile, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return nil, announcement.ErrInvalidAnnouncementID
	}

	iter := attachmentCollection(r.Client, announcementID).Documents(ctx)
	defer iter.Stop()

	results := []announcement.AttachmentFile{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		f, err := attachmentFromDoc(doc, announcementID)
		if err != nil {
			return nil, err
		}

		results = append(results, f)
	}

	return results, nil
}

// Create inserts attachment metadata.
func (r *AnnouncementAttachmentRepositoryFS) Create(
	ctx context.Context,
	f announcement.AttachmentFile,
) (announcement.AttachmentFile, error) {
	if r.Client == nil {
		return announcement.AttachmentFile{}, errors.New("firestore client is nil")
	}
	if f.AnnouncementID == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidAnnouncementID
	}
	if f.ID == "" {
		f.ID = announcement.MakeAttachmentID(f.AnnouncementID, f.FileName)
	}
	if f.ID == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidID
	}

	if _, err := attachmentCollection(r.Client, f.AnnouncementID).Doc(f.ID).Set(ctx, f); err != nil {
		return announcement.AttachmentFile{}, err
	}

	return f, nil
}

// Update applies a patch to attachment metadata.
func (r *AnnouncementAttachmentRepositoryFS) Update(
	ctx context.Context,
	announcementID string,
	fileName string,
	patch announcement.AttachmentFilePatch,
) (announcement.AttachmentFile, error) {
	if r.Client == nil {
		return announcement.AttachmentFile{}, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidAnnouncementID
	}
	if fileName == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidFileName
	}

	ref := attachmentDoc(r.Client, announcementID, fileName)

	updates := []firestore.Update{}

	if patch.FileURL != nil {
		updates = append(updates, firestore.Update{Path: "fileUrl", Value: *patch.FileURL})
	}
	if patch.FileSize != nil {
		updates = append(updates, firestore.Update{Path: "fileSize", Value: *patch.FileSize})
	}
	if patch.MimeType != nil {
		updates = append(updates, firestore.Update{Path: "mimeType", Value: *patch.MimeType})
	}
	if patch.ObjectPath != nil {
		updates = append(updates, firestore.Update{Path: "objectPath", Value: *patch.ObjectPath})
	}
	if patch.UpdatedAt != nil {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: *patch.UpdatedAt})
	}

	if len(updates) > 0 {
		_, err := ref.Update(ctx, updates)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return announcement.AttachmentFile{}, announcement.ErrNotFound
			}
			return announcement.AttachmentFile{}, err
		}
	}

	return getAttachment(ctx, r.Client, announcementID, fileName)
}

// Delete removes one attachment metadata document.
func (r *AnnouncementAttachmentRepositoryFS) Delete(
	ctx context.Context,
	announcementID string,
	fileName string,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.ErrInvalidAnnouncementID
	}
	if fileName == "" {
		return announcement.ErrInvalidFileName
	}

	ref := attachmentDoc(r.Client, announcementID, fileName)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.ErrNotFound
		}
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

func getAttachment(
	ctx context.Context,
	client *firestore.Client,
	announcementID string,
	fileName string,
) (announcement.AttachmentFile, error) {
	doc, err := attachmentDoc(client, announcementID, fileName).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AttachmentFile{}, announcement.ErrNotFound
		}
		return announcement.AttachmentFile{}, err
	}

	return attachmentFromDoc(doc, announcementID)
}

func attachmentFromDoc(
	doc *firestore.DocumentSnapshot,
	announcementID string,
) (announcement.AttachmentFile, error) {
	if doc == nil {
		return announcement.AttachmentFile{}, announcement.ErrNotFound
	}

	var f announcement.AttachmentFile
	if err := doc.DataTo(&f); err != nil {
		return announcement.AttachmentFile{}, err
	}

	if f.ID == "" {
		f.ID = doc.Ref.ID
	}
	if f.AnnouncementID == "" {
		f.AnnouncementID = announcementID
	}

	return f, nil
}
