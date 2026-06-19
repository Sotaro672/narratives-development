// backend/internal/adapters/out/firestore/announcement_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	announcement "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// Firestore implementation of announcement.Repository.
type AnnouncementRepositoryFS struct {
	Client *firestore.Client
}

// Firestore implementation of announcement.AvatarRepository.
type AnnouncementAvatarRepositoryFS struct {
	Client *firestore.Client
}

// Firestore implementation of announcement.AttachmentRepository.
type AnnouncementAttachmentRepositoryFS struct {
	Client *firestore.Client
}

func NewAnnouncementRepositoryFS(client *firestore.Client) *AnnouncementRepositoryFS {
	return &AnnouncementRepositoryFS{Client: client}
}

func NewAnnouncementAvatarRepositoryFS(client *firestore.Client) *AnnouncementAvatarRepositoryFS {
	return &AnnouncementAvatarRepositoryFS{Client: client}
}

func NewAnnouncementAttachmentRepositoryFS(client *firestore.Client) *AnnouncementAttachmentRepositoryFS {
	return &AnnouncementAttachmentRepositoryFS{Client: client}
}

// Compile-time checks.
var _ announcement.Repository = (*AnnouncementRepositoryFS)(nil)
var _ announcement.AvatarRepository = (*AnnouncementAvatarRepositoryFS)(nil)
var _ announcement.AttachmentRepository = (*AnnouncementAttachmentRepositoryFS)(nil)

func announcementCollection(client *firestore.Client) *firestore.CollectionRef {
	return client.Collection("announcements")
}

func avatarCollection(client *firestore.Client, announcementID string) *firestore.CollectionRef {
	return announcementCollection(client).Doc(announcementID).Collection("avatars")
}

func attachmentCollection(client *firestore.Client, announcementID string) *firestore.CollectionRef {
	return announcementCollection(client).Doc(announcementID).Collection("attachments")
}

func attachmentDoc(
	client *firestore.Client,
	announcementID string,
	fileName string,
) *firestore.DocumentRef {
	id := announcement.MakeAttachmentID(announcementID, fileName)
	return attachmentCollection(client, announcementID).Doc(id)
}

// GetByID retrieves an announcement by ID from Firestore.
func (r *AnnouncementRepositoryFS) GetByID(ctx context.Context, id string) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}

	doc, err := announcementCollection(r.Client).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	var a announcement.Announcement
	if err := doc.DataTo(&a); err != nil {
		return announcement.Announcement{}, err
	}

	if a.ID == "" {
		a.ID = doc.Ref.ID
	}

	return a, nil
}

// Create inserts a new announcement.
func (r *AnnouncementRepositoryFS) Create(ctx context.Context, a announcement.Announcement) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}

	ref := announcementCollection(r.Client).Doc(a.ID)
	if a.ID == "" {
		ref = announcementCollection(r.Client).NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = &now

	_, err := ref.Set(ctx, a)
	if err != nil {
		return announcement.Announcement{}, err
	}

	return a, nil
}

// Update replaces/upserts the mutable fields of a persisted announcement.
//
// Expected policy:
// - id is the target document id.
// - a.ID may be empty or equal to id.
// - immutable fields such as CreatedAt and CreatedBy are not overwritten here.
func (r *AnnouncementRepositoryFS) Update(
	ctx context.Context,
	id string,
	a announcement.Announcement,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}

	ref := announcementCollection(r.Client).Doc(id)

	updatedAt := time.Now().UTC()
	if a.UpdatedAt != nil {
		updatedAt = *a.UpdatedAt
	}

	updates := []firestore.Update{
		{Path: "Title", Value: a.Title},
		{Path: "Content", Value: a.Content},
		{Path: "TargetToken", Value: a.TargetToken},
		{Path: "TargetAvatars", Value: a.TargetAvatars},
		{Path: "Published", Value: a.Published},
		{Path: "PublishedAt", Value: a.PublishedAt},
		{Path: "Attachments", Value: a.Attachments},
		{Path: "UpdatedAt", Value: updatedAt},
	}

	if a.UpdatedBy != nil {
		updates = append(updates, firestore.Update{Path: "UpdatedBy", Value: *a.UpdatedBy})
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	return r.GetByID(ctx, id)
}

// MarkPublished marks an announcement as published.
func (r *AnnouncementRepositoryFS) MarkPublished(
	ctx context.Context,
	id string,
	publishedAt time.Time,
	updatedBy *string,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}
	if publishedAt.IsZero() {
		return announcement.Announcement{}, announcement.ErrInvalidPublishedAt
	}

	ref := announcementCollection(r.Client).Doc(id)

	updates := []firestore.Update{
		{Path: "Published", Value: true},
		{Path: "PublishedAt", Value: publishedAt},
		{Path: "UpdatedAt", Value: publishedAt},
	}

	if updatedBy != nil {
		updates = append(updates, firestore.Update{Path: "UpdatedBy", Value: *updatedBy})
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	return r.GetByID(ctx, id)
}

// Delete removes an announcement by ID.
func (r *AnnouncementRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.ErrInvalidID
	}

	ref := announcementCollection(r.Client).Doc(id)

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.ErrNotFound
		}
		return err
	}

	_, err = ref.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

// ListByTargetToken returns announcements whose TargetToken equals tokenBlueprintID.
func (r *AnnouncementRepositoryFS) ListByTargetToken(
	ctx context.Context,
	tokenBlueprintID string,
	page common.Page,
) (common.PageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.PageResult[announcement.Announcement]{}, errors.New("firestore client is nil")
	}
	if tokenBlueprintID == "" {
		return common.PageResult[announcement.Announcement]{}, announcement.ErrInvalidID
	}

	iter := announcementCollection(r.Client).
		Where("TargetToken", "==", tokenBlueprintID).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var items []announcement.Announcement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		var a announcement.Announcement
		if err := doc.DataTo(&a); err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		if a.ID == "" {
			a.ID = doc.Ref.ID
		}

		items = append(items, a)
	}

	return paginateAnnouncements(items, page), nil
}

// ListByTargetAvatar returns announcements whose TargetAvatars contains avatarID.
func (r *AnnouncementRepositoryFS) ListByTargetAvatar(
	ctx context.Context,
	avatarID string,
	page common.Page,
) (common.PageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.PageResult[announcement.Announcement]{}, errors.New("firestore client is nil")
	}
	if avatarID == "" {
		return common.PageResult[announcement.Announcement]{}, announcement.ErrInvalidAvatarID
	}

	iter := announcementCollection(r.Client).
		Where("TargetAvatars", "array-contains", avatarID).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var items []announcement.Announcement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		var a announcement.Announcement
		if err := doc.DataTo(&a); err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		if a.ID == "" {
			a.ID = doc.Ref.ID
		}

		items = append(items, a)
	}

	return paginateAnnouncements(items, page), nil
}

// ListByAnnouncementID returns avatar subcollection documents.
func (r *AnnouncementAvatarRepositoryFS) ListByAnnouncementID(
	ctx context.Context,
	announcementID string,
	filter announcement.AnnouncementAvatarFilter,
) ([]announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return nil, announcement.ErrInvalidAnnouncementID
	}

	iter := avatarCollection(r.Client, announcementID).Documents(ctx)
	defer iter.Stop()

	var results []announcement.AnnouncementAvatar
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var avatar announcement.AnnouncementAvatar
		if err := doc.DataTo(&avatar); err != nil {
			return nil, err
		}

		normalizeAnnouncementAvatarFromDoc(doc, &avatar)

		if len(filter.AvatarIDs) > 0 && !containsString(filter.AvatarIDs, avatar.AvatarID) {
			continue
		}
		if filter.IsRead != nil && avatar.IsRead != *filter.IsRead {
			continue
		}

		results = append(results, avatar)
	}

	return results, nil
}

// Update manages the avatar read state.
//
// This method intentionally works as an upsert so the port does not need
// separate Upsert/Delete methods for announcement avatar records.
func (r *AnnouncementAvatarRepositoryFS) Update(
	ctx context.Context,
	announcementID string,
	avatarID string,
	patch announcement.AnnouncementAvatarPatch,
) (announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return announcement.AnnouncementAvatar{}, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidAvatarID
	}

	now := time.Now().UTC()

	data := map[string]any{
		"announcementId": announcementID,
		"avatarId":       avatarID,
		"updatedAt":      now,
	}

	doc, err := avatarCollection(r.Client, announcementID).Doc(avatarID).Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return announcement.AnnouncementAvatar{}, err
		}
		data["createdAt"] = now
	}

	if doc != nil && doc.Exists() {
		var current announcement.AnnouncementAvatar
		if err := doc.DataTo(&current); err != nil {
			return announcement.AnnouncementAvatar{}, err
		}
		if current.CreatedAt.IsZero() {
			data["createdAt"] = now
		}
	}

	if patch.IsRead != nil {
		data["isRead"] = *patch.IsRead
	}
	if patch.ReadAt != nil {
		data["readAt"] = *patch.ReadAt
	}
	if patch.UpdatedAt != nil {
		data["updatedAt"] = *patch.UpdatedAt
	}

	ref := avatarCollection(r.Client, announcementID).Doc(avatarID)

	_, err = ref.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	return getAnnouncementAvatar(ctx, r.Client, announcementID, avatarID)
}

// MarkRead marks announcements/{announcementId}/avatars/{avatarId} as read.
//
// This method is idempotent. If the avatar record already exists, it updates
// isRead/readAt/updatedAt. If it does not exist, it creates the avatar record.
func (r *AnnouncementAvatarRepositoryFS) MarkRead(
	ctx context.Context,
	announcementID string,
	avatarID string,
	readAt time.Time,
) (announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return announcement.AnnouncementAvatar{}, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidAvatarID
	}
	if readAt.IsZero() {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidReadAt
	}

	ref := avatarCollection(r.Client, announcementID).Doc(avatarID)

	data := map[string]any{
		"announcementId": announcementID,
		"avatarId":       avatarID,
		"isRead":         true,
		"readAt":         readAt,
		"updatedAt":      readAt,
	}

	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return announcement.AnnouncementAvatar{}, err
		}
		data["createdAt"] = readAt
	}

	if doc != nil && doc.Exists() {
		var current announcement.AnnouncementAvatar
		if err := doc.DataTo(&current); err != nil {
			return announcement.AnnouncementAvatar{}, err
		}
		if current.CreatedAt.IsZero() {
			data["createdAt"] = readAt
		}
	}

	_, err = ref.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	return getAnnouncementAvatar(ctx, r.Client, announcementID, avatarID)
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

	var results []announcement.AttachmentFile
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var f announcement.AttachmentFile
		if err := doc.DataTo(&f); err != nil {
			return nil, err
		}

		normalizeAttachmentFromDoc(doc, &f)

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

	_, err := attachmentCollection(r.Client, f.AnnouncementID).Doc(f.ID).Set(ctx, f)
	if err != nil {
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

	if len(updates) == 0 {
		return getAttachment(ctx, r.Client, announcementID, fileName)
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AttachmentFile{}, announcement.ErrNotFound
		}
		return announcement.AttachmentFile{}, err
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

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.ErrNotFound
		}
		return err
	}

	_, err = ref.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

func getAnnouncementAvatar(
	ctx context.Context,
	client *firestore.Client,
	announcementID string,
	avatarID string,
) (announcement.AnnouncementAvatar, error) {
	doc, err := avatarCollection(client, announcementID).Doc(avatarID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AnnouncementAvatar{}, announcement.ErrNotFound
		}
		return announcement.AnnouncementAvatar{}, err
	}

	var avatar announcement.AnnouncementAvatar
	if err := doc.DataTo(&avatar); err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	normalizeAnnouncementAvatarFromDoc(doc, &avatar)

	return avatar, nil
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

	var f announcement.AttachmentFile
	if err := doc.DataTo(&f); err != nil {
		return announcement.AttachmentFile{}, err
	}

	normalizeAttachmentFromDoc(doc, &f)

	return f, nil
}

func normalizeAnnouncementAvatarFromDoc(
	doc *firestore.DocumentSnapshot,
	avatar *announcement.AnnouncementAvatar,
) {
	if avatar == nil || doc == nil {
		return
	}

	if avatar.AvatarID == "" {
		avatar.AvatarID = doc.Ref.ID
	}

	if avatar.AnnouncementID == "" && doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
		avatar.AnnouncementID = doc.Ref.Parent.Parent.ID
	}
}

func normalizeAttachmentFromDoc(
	doc *firestore.DocumentSnapshot,
	f *announcement.AttachmentFile,
) {
	if f == nil || doc == nil {
		return
	}

	if f.ID == "" {
		f.ID = doc.Ref.ID
	}

	if f.AnnouncementID == "" && doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
		f.AnnouncementID = doc.Ref.Parent.Parent.ID
	}
}

func paginateAnnouncements(
	items []announcement.Announcement,
	page common.Page,
) common.PageResult[announcement.Announcement] {
	total := len(items)
	if total == 0 {
		return common.PageResult[announcement.Announcement]{
			Items:      []announcement.Announcement{},
			TotalCount: 0,
			Page:       1,
			PerPage:    0,
		}
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}

	end := offset + perPage
	if end > total {
		end = total
	}

	return common.PageResult[announcement.Announcement]{
		Items:      items[offset:end],
		TotalCount: total,
		Page:       pageNum,
		PerPage:    perPage,
	}
}
