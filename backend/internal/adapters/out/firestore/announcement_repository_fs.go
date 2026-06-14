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

	ref := announcementCollection(r.Client).Doc(id)

	updatedAt := time.Now().UTC()
	if a.UpdatedAt != nil {
		updatedAt = *a.UpdatedAt
	}

	updates := []firestore.Update{
		{Path: "title", Value: a.Title},
		{Path: "content", Value: a.Content},
		{Path: "targetToken", Value: a.TargetToken},
		{Path: "published", Value: a.Published},
		{Path: "publishedAt", Value: a.PublishedAt},
		{Path: "attachments", Value: a.Attachments},
		{Path: "updatedAt", Value: updatedAt},
	}

	if a.UpdatedBy != nil {
		updates = append(updates, firestore.Update{Path: "updatedBy", Value: *a.UpdatedBy})
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

// List returns announcements with simple pagination.
func (r *AnnouncementRepositoryFS) List(
	ctx context.Context,
	_ announcement.Filter,
	_ common.Sort,
	page common.Page,
) (common.PageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.PageResult[announcement.Announcement]{}, errors.New("firestore client is nil")
	}

	iter := announcementCollection(r.Client).
		OrderBy("createdAt", firestore.Desc).
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

	total := len(items)
	if total == 0 {
		return common.PageResult[announcement.Announcement]{
			Items:      []announcement.Announcement{},
			TotalCount: 0,
			Page:       1,
			PerPage:    0,
		}, nil
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
	}, nil
}

// ListByCursor returns announcements using a simple ID-based cursor.
func (r *AnnouncementRepositoryFS) ListByCursor(
	ctx context.Context,
	_ announcement.Filter,
	_ common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.CursorPageResult[announcement.Announcement]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := announcementCollection(r.Client).
		OrderBy("createdAt", firestore.Desc).
		OrderBy("id", firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := cpage.After
	skipping := after != ""

	var items []announcement.Announcement

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.CursorPageResult[announcement.Announcement]{}, err
		}

		var a announcement.Announcement
		if err := doc.DataTo(&a); err != nil {
			return common.CursorPageResult[announcement.Announcement]{}, err
		}

		if a.ID == "" {
			a.ID = doc.Ref.ID
		}

		if skipping {
			if a.ID >= after {
				continue
			}
			skipping = false
		}

		items = append(items, a)

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		cursor := items[limit-1].ID
		items = items[:limit]
		next = &cursor
	}

	return common.CursorPageResult[announcement.Announcement]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
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

		if avatar.AvatarID == "" {
			avatar.AvatarID = doc.Ref.ID
		}

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
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidID
	}

	now := time.Now().UTC()

	data := map[string]any{
		"avatarId":  avatarID,
		"updatedAt": now,
	}

	if patch.IsRead != nil {
		data["isRead"] = *patch.IsRead
	}
	if patch.UpdatedAt != nil {
		data["updatedAt"] = *patch.UpdatedAt
	}

	ref := avatarCollection(r.Client, announcementID).Doc(avatarID)

	_, err := ref.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	doc, err := ref.Get(ctx)
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

	if avatar.AvatarID == "" {
		avatar.AvatarID = doc.Ref.ID
	}

	return avatar, nil
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
