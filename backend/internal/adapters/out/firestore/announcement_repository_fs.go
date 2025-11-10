// backend/internal/adapters/out/firestore/announcement_repository_fs.go
package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	announcement "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// ========================================
// Firestore implementation of announcement.Repository
// ========================================
type AnnouncementRepositoryFS struct {
	Client *firestore.Client
}

// NewAnnouncementRepositoryFS creates a Firestore-backed repository.
func NewAnnouncementRepositoryFS(client *firestore.Client) *AnnouncementRepositoryFS {
	return &AnnouncementRepositoryFS{Client: client}
}

// ========================================
// GetByID
// ========================================
func (r *AnnouncementRepositoryFS) GetByID(ctx context.Context, id string) (announcement.Announcement, error) {
	doc, err := r.Client.Collection("announcements").Doc(id).Get(ctx)
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

	// FirestoreのドキュメントIDを補完
	if a.ID == "" {
		a.ID = doc.Ref.ID
	}

	return a, nil
}

// ========================================
// Exists
// ========================================
func (r *AnnouncementRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.Client.Collection("announcements").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ========================================
// Create
// ========================================
func (r *AnnouncementRepositoryFS) Create(ctx context.Context, a announcement.Announcement) (announcement.Announcement, error) {
	ref := r.Client.Collection("announcements").Doc(a.ID)
	if a.ID == "" {
		ref = r.Client.Collection("announcements").NewDoc()
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

// ========================================
// Save (upsert)
// ========================================
func (r *AnnouncementRepositoryFS) Save(ctx context.Context, a announcement.Announcement, _ *common.SaveOptions) (announcement.Announcement, error) {
	ref := r.Client.Collection("announcements").Doc(a.ID)
	if a.ID == "" {
		ref = r.Client.Collection("announcements").NewDoc()
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

// ========================================
// Update (partial update)
// ========================================
func (r *AnnouncementRepositoryFS) Update(ctx context.Context, id string, p announcement.AnnouncementPatch) (announcement.Announcement, error) {
	ref := r.Client.Collection("announcements").Doc(id)

	updates := []firestore.Update{}
	now := time.Now().UTC()

	if p.Title != nil {
		updates = append(updates, firestore.Update{Path: "title", Value: *p.Title})
	}
	if p.Content != nil {
		updates = append(updates, firestore.Update{Path: "content", Value: *p.Content})
	}
	if p.Category != nil {
		updates = append(updates, firestore.Update{Path: "category", Value: *p.Category})
	}
	if p.TargetAudience != nil {
		updates = append(updates, firestore.Update{Path: "targetAudience", Value: *p.TargetAudience})
	}
	if p.TargetToken != nil {
		updates = append(updates, firestore.Update{Path: "targetToken", Value: *p.TargetToken})
	}
	if p.TargetProducts != nil {
		updates = append(updates, firestore.Update{Path: "targetProducts", Value: *p.TargetProducts})
	}
	if p.TargetAvatars != nil {
		updates = append(updates, firestore.Update{Path: "targetAvatars", Value: *p.TargetAvatars})
	}
	if p.IsPublished != nil {
		updates = append(updates, firestore.Update{Path: "isPublished", Value: *p.IsPublished})
	}
	if p.PublishedAt != nil {
		updates = append(updates, firestore.Update{Path: "publishedAt", Value: p.PublishedAt})
	}
	if p.Attachments != nil {
		updates = append(updates, firestore.Update{Path: "attachments", Value: *p.Attachments})
	}
	if p.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: *p.Status})
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

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
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

// ========================================
// Delete
// ========================================
func (r *AnnouncementRepositoryFS) Delete(ctx context.Context, id string) error {
	ref := r.Client.Collection("announcements").Doc(id)
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

// ========================================
// List (simple query)
// ========================================
// Firestoreではfilterの一部のみを簡易対応。
func (r *AnnouncementRepositoryFS) List(ctx context.Context, _ announcement.Filter, _ common.Sort, _ common.Page) (common.PageResult[announcement.Announcement], error) {
	iter := r.Client.Collection("announcements").OrderBy("createdAt", firestore.Desc).Documents(ctx)
	var items []announcement.Announcement
	for {
		doc, err := iter.Next()
		if err == firestore.Done {
			break
		}
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}
		var a announcement.Announcement
		if err := doc.DataTo(&a); err == nil {
			a.ID = doc.Ref.ID
			items = append(items, a)
		}
	}
	return common.PageResult[announcement.Announcement]{Items: items, TotalCount: len(items), Page: 1, PerPage: len(items)}, nil
}

// ========================================
// Count
// ========================================
func (r *AnnouncementRepositoryFS) Count(ctx context.Context, _ announcement.Filter) (int, error) {
	iter := r.Client.Collection("announcements").Documents(ctx)
	count := 0
	for {
		_, err := iter.Next()
		if err == firestore.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// ========================================
// Search (title/content contains keyword)
// ========================================
func (r *AnnouncementRepositoryFS) Search(ctx context.Context, query string) ([]announcement.Announcement, error) {
	iter := r.Client.Collection("announcements").Documents(ctx)
	var results []announcement.Announcement
	for {
		doc, err := iter.Next()
		if err == firestore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a announcement.Announcement
		if err := doc.DataTo(&a); err == nil {
			if contains(a.Title, query) || contains(a.Content, query) {
				a.ID = doc.Ref.ID
				results = append(results, a)
			}
		}
	}
	return results, nil
}

// ========================================
// Utility: contains
// ========================================
func contains(s, sub string) bool {
	if s == "" || sub == "" {
		return false
	}
	return len(s) >= len(sub) && (stringContainsInsensitive(s, sub))
}

func stringContainsInsensitive(s, sub string) bool {
	sLower := []rune{}
	subLower := []rune{}
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			sLower = append(sLower, r+'a'-'A')
		} else {
			sLower = append(sLower, r)
		}
	}
	for _, r := range sub {
		if r >= 'A' && r <= 'Z' {
			subLower = append(subLower, r+'a'-'A')
		} else {
			subLower = append(subLower, r)
		}
	}
	return string(sLower) == string(subLower) || (len(sLower) > len(subLower) && stringContains(string(sLower[1:]), string(subLower)))
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || (len(s) > 0 && (s[:len(sub)] == sub || stringContains(s[1:], sub))))
}
