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

// Firestore implementation of announcement.Repository
type AnnouncementRepositoryFS struct {
	Client *firestore.Client
}

func NewAnnouncementRepositoryFS(client *firestore.Client) *AnnouncementRepositoryFS {
	return &AnnouncementRepositoryFS{Client: client}
}

// Compile-time check: ensure this satisfies announcement.Repository.
var _ announcement.Repository = (*AnnouncementRepositoryFS)(nil)

func (r *AnnouncementRepositoryFS) announcementCollection() *firestore.CollectionRef {
	return r.Client.Collection("announcements")
}

func (r *AnnouncementRepositoryFS) avatarCollection(announcementID string) *firestore.CollectionRef {
	return r.announcementCollection().Doc(announcementID).Collection("avatars")
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

// Exists checks if an announcement with the given ID exists.
func (r *AnnouncementRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.announcementCollection().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

// Save upserts an announcement.
func (r *AnnouncementRepositoryFS) Save(
	ctx context.Context,
	a announcement.Announcement,
	_ *common.SaveOptions,
) (announcement.Announcement, error) {
	ref := r.announcementCollection().Doc(a.ID)
	if a.ID == "" {
		ref = r.announcementCollection().NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
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
	if p.DeletedAt != nil {
		updates = append(updates, firestore.Update{Path: "deletedAt", Value: p.DeletedAt})
	}
	if p.DeletedBy != nil {
		updates = append(updates, firestore.Update{Path: "deletedBy", Value: *p.DeletedBy})
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

// Count returns the number of announcements (full scan).
func (r *AnnouncementRepositoryFS) Count(ctx context.Context, _ announcement.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	iter := r.announcementCollection().Documents(ctx)
	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// Search performs a simple contains-based search on title and content.
func (r *AnnouncementRepositoryFS) Search(ctx context.Context, query string) ([]announcement.Announcement, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	iter := r.announcementCollection().Documents(ctx)
	var results []announcement.Announcement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a announcement.Announcement
		if err := doc.DataTo(&a); err == nil {
			if contains(a.Title, query) || contains(a.Content, query) {
				if a.ID == "" {
					a.ID = doc.Ref.ID
				}
				results = append(results, a)
			}
		}
	}
	return results, nil
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

// Utility: case-insensitive substring check.
func contains(s, sub string) bool {
	if s == "" || sub == "" {
		return false
	}
	return len(s) >= len(sub) && stringContainsInsensitive(s, sub)
}

func stringContainsInsensitive(s, sub string) bool {
	sLower := toLowerASCII(s)
	subLower := toLowerASCII(sub)
	return stringContains(sLower, subLower)
}

func toLowerASCII(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+'a'-'A')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func stringContains(s, sub string) bool {
	ls, lsub := len(s), len(sub)
	if lsub == 0 {
		return true
	}
	if lsub > ls {
		return false
	}
	for i := 0; i <= ls-lsub; i++ {
		if s[i:i+lsub] == sub {
			return true
		}
	}
	return false
}
