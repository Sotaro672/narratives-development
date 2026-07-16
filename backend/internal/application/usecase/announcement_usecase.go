package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	ann "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// AnnouncementUsecase coordinates Announcement, avatar read state,
// and attachment metadata.
type AnnouncementUsecase struct {
	annRepo    ann.Repository
	avatarRepo ann.AvatarRepository
	attRepo    ann.AttachmentRepository

	now func() time.Time
}

func NewAnnouncementUsecase(
	annRepo ann.Repository,
	avatarRepo ann.AvatarRepository,
	attRepo ann.AttachmentRepository,
) *AnnouncementUsecase {
	return &AnnouncementUsecase{
		annRepo:    annRepo,
		avatarRepo: avatarRepo,
		attRepo:    attRepo,
		now:        time.Now,
	}
}

func (u *AnnouncementUsecase) WithNow(now func() time.Time) *AnnouncementUsecase {
	u.now = now
	return u
}

// =======================
// Announcement command
// =======================

type AnnouncementAttachmentInput struct {
	ID         string
	FileName   string
	FileURL    string
	FileSize   int64
	MimeType   string
	ObjectPath string
}

type CreateAnnouncementInput struct {
	ID            string
	Title         string
	Content       string
	TargetToken   *string
	TargetAvatars []string
	Attachments   []AnnouncementAttachmentInput
	Published     bool
	PublishedAt   *time.Time
	CreatedBy     string
}

func (u *AnnouncementUsecase) CreateAnnouncement(
	ctx context.Context,
	input CreateAnnouncementInput,
) (ann.Announcement, error) {
	if u.annRepo == nil {
		return ann.Announcement{}, ann.ErrNotFound
	}

	now := u.now()

	id := input.ID
	if id == "" {
		generatedID, err := newAnnouncementID()
		if err != nil {
			return ann.Announcement{}, err
		}

		id = generatedID
	}

	attachmentFiles, attachmentIDs, err := buildAttachmentFiles(
		id,
		toNewAttachmentInputs(input.Attachments),
	)
	if err != nil {
		return ann.Announcement{}, err
	}

	entity, err := ann.New(
		id,
		input.Title,
		input.Content,
		input.TargetToken,
		input.TargetAvatars,
		attachmentIDs,
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

	created, err := u.annRepo.Create(ctx, entity)
	if err != nil {
		return ann.Announcement{}, err
	}

	if len(attachmentFiles) == 0 {
		return created, nil
	}

	if u.attRepo == nil {
		return ann.Announcement{}, ann.ErrNotFound
	}

	for _, file := range attachmentFiles {
		if _, err := u.attRepo.Create(ctx, file); err != nil {
			return ann.Announcement{}, err
		}
	}

	refreshed, err := u.annRepo.GetByID(ctx, created.ID)
	if err != nil {
		return ann.Announcement{}, err
	}

	return refreshed, nil
}

func (u *AnnouncementUsecase) ListAnnouncementsByTargetAvatar(
	ctx context.Context,
	avatarID string,
	page common.Page,
) (common.PageResult[ann.Announcement], error) {
	if u.annRepo == nil {
		return common.PageResult[ann.Announcement]{}, ann.ErrNotFound
	}
	if avatarID == "" {
		return common.PageResult[ann.Announcement]{}, ann.ErrInvalidAvatarID
	}

	return u.annRepo.ListByTargetAvatar(ctx, avatarID, page)
}

type UpdateAnnouncementInput struct {
	Title         *string
	Content       *string
	TargetToken   *string
	TargetAvatars *[]string
	Published     *bool
	PublishedAt   *time.Time
	Attachments   *[]AnnouncementAttachmentInput
	UpdatedBy     *string
}

func (u *AnnouncementUsecase) UpdateAnnouncement(
	ctx context.Context,
	id string,
	input UpdateAnnouncementInput,
) (ann.Announcement, error) {
	if u.annRepo == nil {
		return ann.Announcement{}, ann.ErrNotFound
	}
	if id == "" {
		return ann.Announcement{}, ann.ErrInvalidID
	}

	entity, err := u.annRepo.GetByID(ctx, id)
	if err != nil {
		return ann.Announcement{}, err
	}

	if input.Title != nil {
		entity.Title = *input.Title
	}
	if input.Content != nil {
		entity.Content = *input.Content
	}
	if input.TargetToken != nil {
		entity.TargetToken = input.TargetToken
	}
	if input.TargetAvatars != nil {
		entity.TargetAvatars = *input.TargetAvatars
	}
	if input.Published != nil {
		entity.Published = *input.Published
	}
	if input.PublishedAt != nil {
		entity.PublishedAt = input.PublishedAt
	}
	if input.UpdatedBy != nil {
		entity.UpdatedBy = input.UpdatedBy
	}

	now := u.now()
	entity.UpdatedAt = &now

	updated, err := u.annRepo.Update(ctx, id, entity)
	if err != nil {
		return ann.Announcement{}, err
	}

	if input.Attachments == nil {
		return updated, nil
	}

	returned, _, err := u.ReplaceAttachmentsAndSyncAnnouncement(
		ctx,
		id,
		toNewAttachmentInputs(*input.Attachments),
		input.UpdatedBy,
	)
	if err != nil {
		return ann.Announcement{}, err
	}

	return returned, nil
}

func (u *AnnouncementUsecase) MarkPublished(
	ctx context.Context,
	announcementID string,
	updatedBy *string,
) (ann.Announcement, error) {
	if u.annRepo == nil {
		return ann.Announcement{}, ann.ErrNotFound
	}
	if announcementID == "" {
		return ann.Announcement{}, ann.ErrInvalidAnnouncementID
	}

	now := u.now()

	return u.annRepo.MarkPublished(ctx, announcementID, now, updatedBy)
}

func (u *AnnouncementUsecase) DeleteAnnouncement(
	ctx context.Context,
	id string,
) error {
	if u.annRepo == nil {
		return ann.ErrNotFound
	}
	if id == "" {
		return ann.ErrInvalidID
	}

	return u.annRepo.Delete(ctx, id)
}

// =======================
// Announcement avatars
// =======================

type UpsertAnnouncementAvatarInput struct {
	AvatarID string
	IsRead   bool
}

func (u *AnnouncementUsecase) ListAnnouncementAvatars(
	ctx context.Context,
	announcementID string,
	filter ann.AnnouncementAvatarFilter,
) ([]ann.AnnouncementAvatar, error) {
	if u.avatarRepo == nil {
		return nil, ann.ErrNotFound
	}

	return u.avatarRepo.ListByAnnouncementID(ctx, announcementID, filter)
}

// UpsertAnnouncementAvatar creates or updates an announcement avatar record.
func (u *AnnouncementUsecase) UpsertAnnouncementAvatar(
	ctx context.Context,
	announcementID string,
	input UpsertAnnouncementAvatarInput,
) (ann.AnnouncementAvatar, error) {
	if u.avatarRepo == nil {
		return ann.AnnouncementAvatar{}, ann.ErrNotFound
	}
	if input.AvatarID == "" {
		return ann.AnnouncementAvatar{}, ann.ErrInvalidAvatarID
	}

	isRead := input.IsRead
	now := u.now()

	patch := ann.AnnouncementAvatarPatch{
		IsRead:    &isRead,
		UpdatedAt: &now,
	}

	return u.avatarRepo.Upsert(
		ctx,
		announcementID,
		input.AvatarID,
		patch,
	)
}

func (u *AnnouncementUsecase) MarkRead(
	ctx context.Context,
	announcementID string,
	avatarID string,
) (ann.AnnouncementAvatar, error) {
	if u.avatarRepo == nil {
		return ann.AnnouncementAvatar{}, ann.ErrNotFound
	}
	if announcementID == "" {
		return ann.AnnouncementAvatar{}, ann.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return ann.AnnouncementAvatar{}, ann.ErrInvalidAvatarID
	}

	now := u.now()

	return u.avatarRepo.MarkRead(
		ctx,
		announcementID,
		avatarID,
		now,
	)
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

// ReplaceAttachments replaces all attachment metadata of the announcement
// with the provided inputs.
//
// Firebase Storage upload/delete is handled by the frontend.
// This usecase only persists metadata and returns both the saved records and IDs
// to set into Announcement.Attachments.
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

	files, ids, err := buildAttachmentFiles(announcementID, inputs)
	if err != nil {
		return nil, nil, err
	}

	current, err := u.attRepo.ListByAnnouncementID(ctx, announcementID)
	if err != nil {
		return nil, nil, err
	}

	for _, f := range current {
		if err := u.attRepo.Delete(
			ctx,
			announcementID,
			f.FileName,
		); err != nil {
			return nil, nil, err
		}
	}

	saved := make([]ann.AttachmentFile, 0, len(files))
	for _, f := range files {
		out, err := u.attRepo.Create(ctx, f)
		if err != nil {
			return nil, nil, err
		}

		saved = append(saved, out)
	}

	return saved, ids, nil
}

// ReplaceAttachmentsAndSyncAnnouncement replaces attachment metadata and also
// updates Announcement.Attachments.
func (u *AnnouncementUsecase) ReplaceAttachmentsAndSyncAnnouncement(
	ctx context.Context,
	announcementID string,
	inputs []NewAttachmentInput,
	updatedBy *string,
) (ann.Announcement, []ann.AttachmentFile, error) {
	if u.annRepo == nil {
		return ann.Announcement{}, nil, ann.ErrNotFound
	}

	saved, ids, err := u.ReplaceAttachments(
		ctx,
		announcementID,
		inputs,
	)
	if err != nil {
		return ann.Announcement{}, nil, err
	}

	entity, err := u.annRepo.GetByID(ctx, announcementID)
	if err != nil {
		return ann.Announcement{}, nil, err
	}

	entity.Attachments = ids
	if updatedBy != nil {
		entity.UpdatedBy = updatedBy
	}

	now := u.now()
	entity.UpdatedAt = &now

	updated, err := u.annRepo.Update(
		ctx,
		announcementID,
		entity,
	)
	if err != nil {
		return ann.Announcement{}, nil, err
	}

	return updated, saved, nil
}

// =======================
// Delete with cascade
// Announcement -> Attachment metadata
// =======================

// DeleteAnnouncementCascade deletes related attachment metadata and then
// deletes the announcement.
//
// Firebase Storage objects are not deleted here because file storage is managed
// by the frontend.
func (u *AnnouncementUsecase) DeleteAnnouncementCascade(
	ctx context.Context,
	announcementID string,
) error {
	if announcementID == "" {
		return ann.ErrInvalidAnnouncementID
	}

	if u.attRepo != nil {
		files, err := u.attRepo.ListByAnnouncementID(
			ctx,
			announcementID,
		)
		if err != nil {
			return err
		}

		for _, f := range files {
			if err := u.attRepo.Delete(
				ctx,
				announcementID,
				f.FileName,
			); err != nil {
				return err
			}
		}
	}

	if u.annRepo == nil {
		return ann.ErrNotFound
	}

	if err := u.annRepo.Delete(ctx, announcementID); err != nil {
		return err
	}

	return nil
}

// =======================
// Helpers
// =======================

func toNewAttachmentInputs(
	values []AnnouncementAttachmentInput,
) []NewAttachmentInput {
	if len(values) == 0 {
		return nil
	}

	result := make([]NewAttachmentInput, 0, len(values))

	for _, value := range values {
		result = append(result, NewAttachmentInput{
			ID:         value.ID,
			FileName:   value.FileName,
			FileURL:    value.FileURL,
			FileSize:   value.FileSize,
			MimeType:   value.MimeType,
			ObjectPath: value.ObjectPath,
		})
	}

	return result
}

func buildAttachmentFiles(
	announcementID string,
	inputs []NewAttachmentInput,
) ([]ann.AttachmentFile, []string, error) {
	if len(inputs) == 0 {
		return nil, nil, nil
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

	return files, ids, nil
}

func newAnnouncementID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(b[:]), nil
}
