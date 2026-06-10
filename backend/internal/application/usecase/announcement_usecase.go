// backend/internal/application/usecase/announcement_usecase.go
package usecase

import (
	"context"
	"time"

	ann "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// AnnouncementUsecase coordinates Announcement and its attachment metadata.
type AnnouncementUsecase struct {
	annRepo ann.Repository           // announcement repository
	attRepo ann.AttachmentRepository // announcement attachment metadata repository

	now func() time.Time
}

func NewAnnouncementUsecase(
	annRepo ann.Repository,
	attRepo ann.AttachmentRepository,
) *AnnouncementUsecase {
	return &AnnouncementUsecase{
		annRepo: annRepo,
		attRepo: attRepo,
		now:     time.Now,
	}
}

func (u *AnnouncementUsecase) WithNow(now func() time.Time) *AnnouncementUsecase {
	u.now = now
	return u
}

// =======================
// Announcement CRUD
// =======================

type CreateAnnouncementInput struct {
	ID          string
	Title       string
	Content     string
	TargetToken *string
	Attachments []string
	Published   bool
	PublishedAt *time.Time
	CreatedBy   string
}

func (u *AnnouncementUsecase) CreateAnnouncement(
	ctx context.Context,
	input CreateAnnouncementInput,
) (ann.Announcement, error) {
	now := u.now()

	entity, err := ann.New(
		input.ID,
		input.Title,
		input.Content,
		input.TargetToken,
		input.Attachments,
		input.Published,
		now,
		input.CreatedBy,
		input.PublishedAt,
		nil,
		nil,
	)
	if err != nil {
		return ann.Announcement{}, err
	}

	return u.annRepo.Create(ctx, entity)
}

func (u *AnnouncementUsecase) GetAnnouncement(
	ctx context.Context,
	id string,
) (ann.Announcement, error) {
	return u.annRepo.GetByID(ctx, id)
}

func (u *AnnouncementUsecase) ListAnnouncements(
	ctx context.Context,
	filter ann.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[ann.Announcement], error) {
	return u.annRepo.List(ctx, filter, sort, page)
}

func (u *AnnouncementUsecase) ListAnnouncementsByCursor(
	ctx context.Context,
	filter ann.Filter,
	sort common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[ann.Announcement], error) {
	return u.annRepo.ListByCursor(ctx, filter, sort, cpage)
}

type UpdateAnnouncementInput struct {
	Title       *string
	Content     *string
	TargetToken *string
	Published   *bool
	PublishedAt *time.Time
	Attachments *[]string
	UpdatedBy   *string
}

func (u *AnnouncementUsecase) UpdateAnnouncement(
	ctx context.Context,
	id string,
	input UpdateAnnouncementInput,
) (ann.Announcement, error) {
	patch := ann.AnnouncementPatch{
		Title:       input.Title,
		Content:     input.Content,
		TargetToken: input.TargetToken,
		Published:   input.Published,
		PublishedAt: input.PublishedAt,
		Attachments: input.Attachments,
		UpdatedBy:   input.UpdatedBy,
	}
	return u.annRepo.Update(ctx, id, patch)
}

func (u *AnnouncementUsecase) DeleteAnnouncement(
	ctx context.Context,
	id string,
) error {
	return u.annRepo.Delete(ctx, id)
}

// =======================
// Announcement avatars CRUD
// =======================

type UpsertAnnouncementAvatarInput struct {
	AvatarID string
	IsRead   bool
}

type UpdateAnnouncementAvatarInput struct {
	IsRead *bool
}

func (u *AnnouncementUsecase) ListAnnouncementAvatars(
	ctx context.Context,
	announcementID string,
	filter ann.AnnouncementAvatarFilter,
) ([]ann.AnnouncementAvatar, error) {
	return u.annRepo.ListAvatars(ctx, announcementID, filter)
}

func (u *AnnouncementUsecase) GetAnnouncementAvatar(
	ctx context.Context,
	announcementID, avatarID string,
) (ann.AnnouncementAvatar, error) {
	return u.annRepo.GetAvatar(ctx, announcementID, avatarID)
}

func (u *AnnouncementUsecase) UpsertAnnouncementAvatar(
	ctx context.Context,
	announcementID string,
	input UpsertAnnouncementAvatarInput,
) (ann.AnnouncementAvatar, error) {
	avatar := ann.AnnouncementAvatar{
		AvatarID: input.AvatarID,
		IsRead:   input.IsRead,
	}
	return u.annRepo.UpsertAvatar(ctx, announcementID, avatar)
}

func (u *AnnouncementUsecase) UpdateAnnouncementAvatar(
	ctx context.Context,
	announcementID, avatarID string,
	input UpdateAnnouncementAvatarInput,
) (ann.AnnouncementAvatar, error) {
	patch := ann.AnnouncementAvatarPatch{
		IsRead: input.IsRead,
	}
	return u.annRepo.UpdateAvatar(ctx, announcementID, avatarID, patch)
}

func (u *AnnouncementUsecase) DeleteAnnouncementAvatar(
	ctx context.Context,
	announcementID, avatarID string,
) error {
	return u.annRepo.DeleteAvatar(ctx, announcementID, avatarID)
}

// =======================
// Attachments metadata
// =======================

type NewAttachmentInput struct {
	ID         string
	FileName   string
	FileURL    string
	FileSize   int64
	MimeType   string
	ObjectPath string
}

// ReplaceAttachments replaces all attachment metadata of the announcement with the provided inputs.
// Firebase Storage upload/delete is handled by the frontend.
// This usecase only persists metadata and returns both the saved records and IDs to set into Announcement.Attachments.
func (u *AnnouncementUsecase) ReplaceAttachments(
	ctx context.Context,
	announcementID string,
	inputs []NewAttachmentInput,
) ([]ann.AttachmentFile, []string, error) {
	if announcementID == "" {
		return nil, nil, ann.ErrInvalidAnnouncementID
	}
	if u.attRepo == nil {
		return nil, nil, ann.ErrNotFound
	}

	files := make([]ann.AttachmentFile, 0, len(inputs))
	ids := make([]string, 0, len(inputs))

	for _, in := range inputs {
		f, err := ann.NewAttachmentFileWithObjectPath(
			announcementID,
			in.ID,
			in.FileName,
			in.FileURL,
			in.FileSize,
			in.MimeType,
			in.ObjectPath,
		)
		if err != nil {
			return nil, nil, err
		}

		files = append(files, f)
		ids = append(ids, f.ID)
	}

	if err := u.attRepo.DeleteAllAttachmentsByAnnouncementID(ctx, announcementID); err != nil {
		return nil, nil, err
	}

	saved := make([]ann.AttachmentFile, 0, len(files))
	for _, f := range files {
		out, err := u.attRepo.CreateAttachment(ctx, f)
		if err != nil {
			return nil, nil, err
		}
		saved = append(saved, out)
	}

	return saved, ids, nil
}

// ReplaceAttachmentsAndSyncAnnouncement replaces attachment metadata and also updates Announcement.Attachments.
func (u *AnnouncementUsecase) ReplaceAttachmentsAndSyncAnnouncement(
	ctx context.Context,
	announcementID string,
	inputs []NewAttachmentInput,
	updatedBy *string,
) (ann.Announcement, []ann.AttachmentFile, error) {
	saved, ids, err := u.ReplaceAttachments(ctx, announcementID, inputs)
	if err != nil {
		return ann.Announcement{}, nil, err
	}

	updated, err := u.annRepo.Update(ctx, announcementID, ann.AnnouncementPatch{
		Attachments: &ids,
		UpdatedBy:   updatedBy,
	})
	if err != nil {
		return ann.Announcement{}, nil, err
	}

	return updated, saved, nil
}

// =======================
// Delete with cascade (Announcement -> Attachment metadata)
// =======================

// DeleteAnnouncementCascade deletes related attachment metadata and then deletes the announcement.
// Firebase Storage objects are not deleted here because file storage is managed by the frontend.
func (u *AnnouncementUsecase) DeleteAnnouncementCascade(ctx context.Context, announcementID string) error {
	if announcementID == "" {
		return ann.ErrInvalidAnnouncementID
	}

	if u.attRepo != nil {
		if err := u.attRepo.DeleteAllAttachmentsByAnnouncementID(ctx, announcementID); err != nil {
			return err
		}
	}

	if err := u.annRepo.Delete(ctx, announcementID); err != nil {
		return err
	}

	return nil
}
