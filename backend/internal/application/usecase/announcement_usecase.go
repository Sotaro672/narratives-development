// backend\internal\application\usecase\announcement_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	ann "narratives/internal/domain/announcement"
	aa "narratives/internal/domain/announcementAttachment"
)

// AnnouncementUsecase coordinates Announcement and AnnouncementAttachment domains.
type AnnouncementUsecase struct {
	annRepo  ann.Repository       // announcement repository (source of truth)
	attRepo  aa.Repository        // attachment metadata repository
	objStore aa.ObjectStoragePort // GCS (or compatible) object storage adapter

	now func() time.Time
}

func NewAnnouncementUsecase(
	annRepo ann.Repository,
	attRepo aa.Repository,
	objStore aa.ObjectStoragePort,
) *AnnouncementUsecase {
	return &AnnouncementUsecase{
		annRepo:  annRepo,
		attRepo:  attRepo,
		objStore: objStore,
		now:      time.Now,
	}
}

func (u *AnnouncementUsecase) WithNow(now func() time.Time) *AnnouncementUsecase {
	u.now = now
	return u
}

// =======================
// Attachments (create/replace)
// =======================

type NewAttachmentInput struct {
	FileName string
	FileURL  string
	FileSize int64
	MimeType string
}

// ReplaceAttachments replaces all attachments of the announcement with the provided inputs.
// It persists attachment metadata and returns both the saved records and the IDs to set into Announcement.Attachments.
// Note: The caller should update the Announcement entity to use the returned IDs (e.g., via annRepo).
func (u *AnnouncementUsecase) ReplaceAttachments(
	ctx context.Context,
	announcementID string,
	inputs []NewAttachmentInput,
) ([]aa.AttachmentFile, []string, error) {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return nil, nil, aa.ErrInvalidAnnouncementID
	}

	// Build metadata following the domain convention (bucket + objectPath).
	files := make([]aa.AttachmentFile, 0, len(inputs))
	ids := make([]string, 0, len(inputs))
	for _, in := range inputs {
		f, err := aa.NewAttachmentFileWithBucket(
			aa.DefaultBucket,
			announcementID,
			in.FileName,
			in.FileURL,
			in.FileSize,
			in.MimeType,
		)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
		ids = append(ids, f.ID)
	}

	// Replace: delete existing, then create new metadata records.
	if err := u.attRepo.DeleteAllByAnnouncementID(ctx, announcementID); err != nil {
		return nil, nil, err
	}
	saved := make([]aa.AttachmentFile, 0, len(files))
	for _, f := range files {
		out, err := u.attRepo.Create(ctx, f)
		if err != nil {
			return nil, nil, err
		}
		saved = append(saved, out)
	}

	return saved, ids, nil
}

// =======================
// Delete with cascade (Announcement -> Attachments in GCS)
// =======================

// DeleteAnnouncementCascade deletes the announcement and also removes related attachments:
// - delete GCS objects via ObjectStoragePort (if provided)
// - delete attachment metadata via Repository (attRepo)
// - finally delete the announcement via annRepo
func (u *AnnouncementUsecase) DeleteAnnouncementCascade(ctx context.Context, announcementID string) error {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return aa.ErrInvalidAnnouncementID
	}

	// Load attachment metadata to determine delete targets in GCS.
	files, err := u.attRepo.GetByAnnouncementID(ctx, announcementID)
	if err != nil {
		return err
	}

	// Build GCS delete ops and execute.
	if u.objStore != nil && len(files) > 0 {
		ops := aa.BuildGCSDeleteOps(files)
		if len(ops) > 0 {
			if err := u.objStore.DeleteObjects(ctx, ops); err != nil {
				return err
			}
		}
	}

	// Delete metadata.
	if err := u.attRepo.DeleteAllByAnnouncementID(ctx, announcementID); err != nil {
		return err
	}

	// Delete the announcement itself (source of truth).
	if err := u.annRepo.Delete(ctx, announcementID); err != nil {
		return err
	}
	return nil
}
