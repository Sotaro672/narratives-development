// backend/internal/adapters/out/firestore/announcement_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	announcement "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// Firestore implementation of announcement.Repository
type AnnouncementRepositoryFS struct {
	Client *firestore.Client
}

func NewAnnouncementRepositoryFS(client *firestore.Client) *AnnouncementRepositoryFS {
	return &AnnouncementRepositoryFS{Client: client}
}

// Compile-time check: ensure this satisfies announcement.Repository.
var _ announcement.Repository = (*AnnouncementRepositoryFS)(nil)

// Compile-time check: ensure this satisfies announcement.AttachmentRepository.
var _ announcement.AttachmentRepository = (*AnnouncementRepositoryFS)(nil)

func (r *AnnouncementRepositoryFS) announcementCollection() *firestore.CollectionRef {
	return r.Client.Collection("announcements")
}

func (r *AnnouncementRepositoryFS) avatarCollection(announcementID string) *firestore.CollectionRef {
	return r.announcementCollection().Doc(announcementID).Collection("avatars")
}

func (r *AnnouncementRepositoryFS) attachmentCollection(announcementID string) *firestore.CollectionRef {
	return r.announcementCollection().Doc(announcementID).Collection("attachments")
}

func (r *AnnouncementRepositoryFS) attachmentDoc(
	announcementID string,
	fileName string,
) *firestore.DocumentRef {
	id := announcement.MakeAttachmentID(announcementID, fileName)
	return r.attachmentCollection(announcementID).Doc(id)
}

// GetByID retrieves an announcement by ID from Firestore.
func (r *AnnouncementRepositoryFS) GetByID(ctx context.Context, id string) (announcement.Announcement, error) {
	doc, err := r.announcementCollection().Doc(id).Get(ctx)
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
	ref := r.announcementCollection().Doc(a.ID)
	if a.ID == "" {
		ref = r.announcementCollection().NewDoc()
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

// Update applies a patch to an existing announcement.
func (r *AnnouncementRepositoryFS) Update(
	ctx context.Context,
	id string,
	p announcement.AnnouncementPatch,
) (announcement.Announcement, error) {
	ref := r.announcementCollection().Doc(id)

	updates := []firestore.Update{}
	now := time.Now().UTC()

	if p.Title != nil {
		updates = append(updates, firestore.Update{Path: "title", Value: *p.Title})
	}
	if p.Content != nil {
		updates = append(updates, firestore.Update{Path: "content", Value: *p.Content})
	}
	if p.TargetToken != nil {
		updates = append(updates, firestore.Update{Path: "targetToken", Value: *p.TargetToken})
	}
	if p.Published != nil {
		updates = append(updates, firestore.Update{Path: "published", Value: *p.Published})
	}
	if p.PublishedAt != nil {
		updates = append(updates, firestore.Update{Path: "publishedAt", Value: p.PublishedAt})
	}
	if p.Attachments != nil {
		updates = append(updates, firestore.Update{Path: "attachments", Value: *p.Attachments})
	}
	if p.UpdatedBy != nil {
		updates = append(updates, firestore.Update{Path: "updatedBy", Value: *p.UpdatedBy})
	}

	// Always bump updatedAt
	updates = append(updates, firestore.Update{Path: "updatedAt", Value: now})

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
	ref := r.announcementCollection().Doc(id)
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

	iter := r.announcementCollection().
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
		if err := doc.DataTo(&a); err == nil {
			if a.ID == "" {
				a.ID = doc.Ref.ID
			}
			items = append(items, a)
		}
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

	q := r.announcementCollection().
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

// ListAvatars returns avatar subcollection documents.
func (r *AnnouncementRepositoryFS) ListAvatars(
	ctx context.Context,
	announcementID string,
	filter announcement.AnnouncementAvatarFilter,
) ([]announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	iter := r.avatarCollection(announcementID).Documents(ctx)
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

// GetAvatar retrieves one avatar subcollection document.
func (r *AnnouncementRepositoryFS) GetAvatar(
	ctx context.Context,
	announcementID, avatarID string,
) (announcement.AnnouncementAvatar, error) {
	doc, err := r.avatarCollection(announcementID).Doc(avatarID).Get(ctx)
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

// UpsertAvatar saves one avatar subcollection document.
func (r *AnnouncementRepositoryFS) UpsertAvatar(
	ctx context.Context,
	announcementID string,
	avatar announcement.AnnouncementAvatar,
) (announcement.AnnouncementAvatar, error) {
	if avatar.AvatarID == "" {
		return announcement.AnnouncementAvatar{}, announcement.ErrInvalidID
	}

	_, err := r.avatarCollection(announcementID).Doc(avatar.AvatarID).Set(ctx, avatar)
	if err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	return avatar, nil
}

// UpdateAvatar applies a patch to an avatar subcollection document.
func (r *AnnouncementRepositoryFS) UpdateAvatar(
	ctx context.Context,
	announcementID, avatarID string,
	patch announcement.AnnouncementAvatarPatch,
) (announcement.AnnouncementAvatar, error) {
	updates := []firestore.Update{}

	if patch.IsRead != nil {
		updates = append(updates, firestore.Update{Path: "isRead", Value: *patch.IsRead})
	}

	if len(updates) == 0 {
		return r.GetAvatar(ctx, announcementID, avatarID)
	}

	_, err := r.avatarCollection(announcementID).Doc(avatarID).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AnnouncementAvatar{}, announcement.ErrNotFound
		}
		return announcement.AnnouncementAvatar{}, err
	}

	return r.GetAvatar(ctx, announcementID, avatarID)
}

// DeleteAvatar removes an avatar subcollection document.
func (r *AnnouncementRepositoryFS) DeleteAvatar(
	ctx context.Context,
	announcementID, avatarID string,
) error {
	ref := r.avatarCollection(announcementID).Doc(avatarID)

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

// ListAttachments returns attachment metadata with simple pagination.
func (r *AnnouncementRepositoryFS) ListAttachments(
	ctx context.Context,
	filter announcement.AttachmentFilter,
	_ announcement.Sort,
	page announcement.Page,
) (announcement.PageResult[announcement.AttachmentFile], error) {
	if r.Client == nil {
		return announcement.PageResult[announcement.AttachmentFile]{}, errors.New("firestore client is nil")
	}

	iter := r.Client.CollectionGroup("attachments").Documents(ctx)
	defer iter.Stop()

	var items []announcement.AttachmentFile
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return announcement.PageResult[announcement.AttachmentFile]{}, err
		}

		var f announcement.AttachmentFile
		if err := doc.DataTo(&f); err != nil {
			return announcement.PageResult[announcement.AttachmentFile]{}, err
		}

		normalizeAttachmentFromDoc(doc, &f)

		if !matchesAttachmentFilter(f, filter) {
			continue
		}

		items = append(items, f)
	}

	total := len(items)
	if total == 0 {
		return announcement.PageResult[announcement.AttachmentFile]{
			Items:      []announcement.AttachmentFile{},
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

	return announcement.PageResult[announcement.AttachmentFile]{
		Items:      items[offset:end],
		TotalCount: total,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListAttachmentsByCursor returns attachment metadata using a simple ID-based cursor.
func (r *AnnouncementRepositoryFS) ListAttachmentsByCursor(
	ctx context.Context,
	filter announcement.AttachmentFilter,
	_ announcement.Sort,
	cpage announcement.CursorPage,
) (announcement.CursorPageResult[announcement.AttachmentFile], error) {
	if r.Client == nil {
		return announcement.CursorPageResult[announcement.AttachmentFile]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	it := r.Client.CollectionGroup("attachments").Documents(ctx)
	defer it.Stop()

	after := cpage.After
	skipping := after != ""

	var items []announcement.AttachmentFile

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return announcement.CursorPageResult[announcement.AttachmentFile]{}, err
		}

		var f announcement.AttachmentFile
		if err := doc.DataTo(&f); err != nil {
			return announcement.CursorPageResult[announcement.AttachmentFile]{}, err
		}

		normalizeAttachmentFromDoc(doc, &f)

		if !matchesAttachmentFilter(f, filter) {
			continue
		}

		if skipping {
			if f.ID >= after {
				continue
			}
			skipping = false
		}

		items = append(items, f)

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

	return announcement.CursorPageResult[announcement.AttachmentFile]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// GetAttachment retrieves one attachment metadata document.
func (r *AnnouncementRepositoryFS) GetAttachment(
	ctx context.Context,
	announcementID string,
	fileName string,
) (announcement.AttachmentFile, error) {
	doc, err := r.attachmentDoc(announcementID, fileName).Get(ctx)
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

// GetAttachmentsByAnnouncementID retrieves all attachment metadata documents for one announcement.
func (r *AnnouncementRepositoryFS) GetAttachmentsByAnnouncementID(
	ctx context.Context,
	announcementID string,
) ([]announcement.AttachmentFile, error) {
	if announcementID == "" {
		return nil, announcement.ErrInvalidAnnouncementID
	}

	iter := r.attachmentCollection(announcementID).Documents(ctx)
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

// CreateAttachment inserts attachment metadata.
func (r *AnnouncementRepositoryFS) CreateAttachment(
	ctx context.Context,
	f announcement.AttachmentFile,
) (announcement.AttachmentFile, error) {
	if f.AnnouncementID == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidAnnouncementID
	}
	if f.ID == "" {
		return announcement.AttachmentFile{}, announcement.ErrInvalidID
	}

	_, err := r.attachmentCollection(f.AnnouncementID).Doc(f.ID).Set(ctx, f)
	if err != nil {
		return announcement.AttachmentFile{}, err
	}

	return f, nil
}

// UpdateAttachment applies a patch to attachment metadata.
func (r *AnnouncementRepositoryFS) UpdateAttachment(
	ctx context.Context,
	announcementID string,
	fileName string,
	patch announcement.AttachmentFilePatch,
) (announcement.AttachmentFile, error) {
	ref := r.attachmentDoc(announcementID, fileName)

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

	if len(updates) == 0 {
		return r.GetAttachment(ctx, announcementID, fileName)
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AttachmentFile{}, announcement.ErrNotFound
		}
		return announcement.AttachmentFile{}, err
	}

	return r.GetAttachment(ctx, announcementID, fileName)
}

// DeleteAttachment removes one attachment metadata document.
func (r *AnnouncementRepositoryFS) DeleteAttachment(
	ctx context.Context,
	announcementID string,
	fileName string,
) error {
	ref := r.attachmentDoc(announcementID, fileName)

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

// DeleteAllAttachmentsByAnnouncementID removes all attachment metadata documents for one announcement.
func (r *AnnouncementRepositoryFS) DeleteAllAttachmentsByAnnouncementID(
	ctx context.Context,
	announcementID string,
) error {
	if announcementID == "" {
		return announcement.ErrInvalidAnnouncementID
	}

	iter := r.attachmentCollection(announcementID).Documents(ctx)
	defer iter.Stop()

	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		batch.Delete(doc.Ref)
		count++

		if count == 500 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.Client.Batch()
			count = 0
		}
	}

	if count > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
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

func matchesAttachmentFilter(
	f announcement.AttachmentFile,
	filter announcement.AttachmentFilter,
) bool {
	if filter.AnnouncementID != nil && f.AnnouncementID != *filter.AnnouncementID {
		return false
	}

	if filter.FileName != nil && f.FileName != *filter.FileName {
		return false
	}

	if len(filter.MimeTypes) > 0 && !containsString(filter.MimeTypes, f.MimeType) {
		return false
	}

	if filter.SizeMin != nil && f.FileSize < *filter.SizeMin {
		return false
	}

	if filter.SizeMax != nil && f.FileSize > *filter.SizeMax {
		return false
	}

	if filter.ObjectPathLike != "" && !strings.HasPrefix(f.ObjectPath, filter.ObjectPathLike) {
		return false
	}

	if filter.SearchQuery != "" {
		q := strings.ToLower(filter.SearchQuery)
		if !strings.Contains(strings.ToLower(f.FileName), q) &&
			!strings.Contains(strings.ToLower(f.FileURL), q) &&
			!strings.Contains(strings.ToLower(f.ObjectPath), q) {
			return false
		}
	}

	return true
}
