// backend/internal/adapters/out/firestore/list_image_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	listuc "narratives/internal/application/usecase/list"
	listimgdom "narratives/internal/domain/listImage"
)

// Firestore schema (canonical only):
// - lists/{listId}/images/{imageId}   // ✅ docID is imageId
// fields:
// - id            : string   (optional; recommended: imageId)
// - list_id       : string
// - url           : string   (REQUIRED; no rebuild fallback in repo)
// - file_name     : string
// - size          : number
// - display_order : number
// - bucket        : string   (optional; debug only; may be empty)
// - object_path   : string   (MUST be "lists/{listId}/images/{imageId}")
// - created_at    : timestamp
// - updated_at    : timestamp
//
// NOTE:
// - Legacy URL parsing / DefaultBucket fallback are removed.
// - Canonical object path is: lists/{listId}/images/{imageId}

type ListImageRepositoryFS struct {
	Client *gfs.Client
}

func NewListImageRepositoryFS(client *gfs.Client) *ListImageRepositoryFS {
	return &ListImageRepositoryFS{Client: client}
}

func (r *ListImageRepositoryFS) listCol(listID string) *gfs.CollectionRef {
	return r.Client.Collection("lists").Doc(listID).Collection("images")
}

// compile-time checks (ports)
var _ listuc.ListImageReader = (*ListImageRepositoryFS)(nil)
var _ listuc.ListImageByIDReader = (*ListImageRepositoryFS)(nil)
var _ listuc.ListImageRecordRepository = (*ListImageRepositoryFS)(nil)

// Optional capability for usecase.DeleteImage (type-asserted in usecase)
type listImageRecordDeleter interface {
	Delete(ctx context.Context, listID string, imageID string) error
}

var _ listImageRecordDeleter = (*ListImageRepositoryFS)(nil)

// ============================================================
// Port: ListImageRecordRepository
// ============================================================

// Upsert stores list image record into Firestore subcollection.
// docID policy: imageId
func (r *ListImageRepositoryFS) Upsert(ctx context.Context, img listimgdom.ListImage) (listimgdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimgdom.ListImage{}, errors.New("firestore client is nil")
	}

	listID := strings.TrimSpace(img.ListID)
	if listID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidListID
	}

	imageID := strings.TrimSpace(img.ID)
	if imageID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidID
	}
	if strings.Contains(imageID, "/") {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("invalid_image_id")
	}

	// ✅ canonical objectPath is required (legacy removed)
	objectPath := strings.TrimLeft(strings.TrimSpace(img.ObjectPath), "/")
	if objectPath == "" {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_required")
	}
	if !isCanonicalObjectPath(objectPath, listID, imageID) {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_not_canonical")
	}

	// fileName is required
	fileName := strings.TrimSpace(img.FileName)
	if fileName == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidFileName
	}

	// ✅ url is required (no DefaultBucket rebuild in repo)
	u := strings.TrimSpace(img.URL)
	if u == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidURL
	}

	// bucket is optional/debug only. (Do NOT parse URL in repo; legacy removed.)
	bucket := ""

	// stored "id" field in firestore:
	storedID := imageID

	// times
	now := time.Now().UTC()
	ref := r.listCol(listID).Doc(imageID)

	// Transaction to preserve created_at on update
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		createdAt := now

		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			// new doc: createdAt = now
		} else {
			// existing: preserve created_at if present
			if v, ok := snap.Data()["created_at"]; ok {
				if t, ok2 := v.(time.Time); ok2 && !t.IsZero() {
					createdAt = t.UTC()
				}
			}
		}

		data := map[string]any{
			"id":            storedID,
			"list_id":       listID,
			"url":           u,
			"file_name":     fileName,
			"size":          img.Size,
			"display_order": img.DisplayOrder,

			"bucket":      strings.TrimSpace(bucket),
			"object_path": objectPath, // ✅ canonical only

			"created_at": createdAt,
			"updated_at": now,
		}

		return tx.Set(ref, data, gfs.MergeAll)
	})
	if err != nil {
		return listimgdom.ListImage{}, err
	}

	// Return domain object
	out, derr := listimgdom.New(
		imageID,
		listID,
		u,
		objectPath,
		fileName,
		img.Size,
		img.DisplayOrder,
	)
	if derr != nil {
		// best-effort fallback (do not break)
		return listimgdom.ListImage{
			ID:           imageID,
			ListID:       listID,
			URL:          u,
			ObjectPath:   objectPath,
			FileName:     fileName,
			Size:         img.Size,
			DisplayOrder: img.DisplayOrder,
		}, nil
	}
	return out, nil
}

// Delete deletes Firestore record: /lists/{listId}/images/{imageId}
// (canonical only)
func (r *ListImageRepositoryFS) Delete(ctx context.Context, listID string, imageID string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listimgdom.ErrInvalidListID
	}
	if imageID == "" {
		return listimgdom.ErrInvalidID
	}
	if strings.Contains(imageID, "/") {
		return usecase.ErrInvalidArgument("invalid_image_id")
	}

	ref := r.listCol(listID).Doc(imageID)
	_, err := ref.Delete(ctx)
	if err != nil {
		// idempotent: not found => success
		if status.Code(err) == codes.NotFound {
			return listimgdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Port: ListImageReader
// ============================================================

// ListByListID returns images for a list ordered by displayOrder asc.
func (r *ListImageRepositoryFS) ListByListID(ctx context.Context, listID string) ([]listimgdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listimgdom.ListImage{}, nil
	}

	it := r.listCol(listID).
		OrderBy("display_order", gfs.Asc).
		OrderBy(gfs.DocumentID, gfs.Asc).
		Documents(ctx)
	defer it.Stop()

	out := make([]listimgdom.ListImage, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		img, ok := decodeListImageDoc(doc, listID)
		if !ok {
			continue
		}
		out = append(out, img)
	}

	return out, nil
}

// ============================================================
// Port: ListImageByIDReader
// ============================================================

// GetByID supports:
// - canonical objectPath: "lists/{listId}/images/{imageId}"  => direct read
// - imageId only: "{imageId}" => collectionGroup query by DocumentID
//
// ✅ Legacy removed: URL input is NOT supported.
func (r *ListImageRepositoryFS) GetByID(ctx context.Context, id string) (listimgdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimgdom.ListImage{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return listimgdom.ListImage{}, listimgdom.ErrNotFound
	}

	// 1) If canonical objectPath, direct read by listId + imageId
	if looksLikeCanonicalObjectPath(id) {
		listID, imageID, ok := splitCanonicalObjectPath(id)
		if ok {
			doc, err := r.listCol(listID).Doc(imageID).Get(ctx)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return listimgdom.ListImage{}, listimgdom.ErrNotFound
				}
				return listimgdom.ListImage{}, err
			}
			img, ok := decodeListImageDoc(doc, listID)
			if !ok {
				return listimgdom.ListImage{}, listimgdom.ErrNotFound
			}
			return img, nil
		}
	}

	// 2) Fallback: treat as imageId and search by DocumentID in collection group
	imageID := id
	if strings.Contains(imageID, "/") {
		return listimgdom.ListImage{}, listimgdom.ErrNotFound
	}

	q := r.Client.CollectionGroup("images").
		Where(gfs.DocumentID, "==", imageID).
		Limit(1)

	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return listimgdom.ListImage{}, listimgdom.ErrNotFound
		}
		return listimgdom.ListImage{}, err
	}

	// derive listID from parent path (lists/{listId}/images/{imageId})
	listID := ""
	if doc != nil && doc.Ref != nil && doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
		listID = strings.TrimSpace(doc.Ref.Parent.Parent.ID)
	}

	img, ok := decodeListImageDoc(doc, listID)
	if !ok {
		return listimgdom.ListImage{}, listimgdom.ErrNotFound
	}
	return img, nil
}

// ============================================================
// Decode helpers
// ============================================================

func decodeListImageDoc(doc *gfs.DocumentSnapshot, fallbackListID string) (listimgdom.ListImage, bool) {
	if doc == nil || doc.Ref == nil {
		return listimgdom.ListImage{}, false
	}

	var raw struct {
		ID           string    `firestore:"id"`
		ListID       string    `firestore:"list_id"`
		URL          string    `firestore:"url"`
		FileName     string    `firestore:"file_name"`
		Size         int64     `firestore:"size"`
		DisplayOrder int       `firestore:"display_order"`
		CreatedAt    time.Time `firestore:"created_at"`
		UpdatedAt    time.Time `firestore:"updated_at"`
		ObjectPath   string    `firestore:"object_path"`
		Bucket       string    `firestore:"bucket"` // debug only; may be empty
	}

	if err := doc.DataTo(&raw); err != nil {
		return listimgdom.ListImage{}, false
	}

	listID := strings.TrimSpace(raw.ListID)
	if listID == "" {
		listID = strings.TrimSpace(fallbackListID)
	}
	if listID == "" {
		return listimgdom.ListImage{}, false
	}

	// ✅ imageID is docID
	imageID := strings.TrimSpace(doc.Ref.ID)
	if imageID == "" {
		return listimgdom.ListImage{}, false
	}

	// ✅ canonical objectPath required; if missing, rebuild canonically
	objectPath := strings.TrimLeft(strings.TrimSpace(raw.ObjectPath), "/")
	if objectPath == "" {
		objectPath = listimgdom.CanonicalObjectPath(listID, imageID)
	}

	// ✅ url: do not rebuild from bucket (legacy removed)
	urlStr := strings.TrimSpace(raw.URL)

	fileName := strings.TrimSpace(raw.FileName)

	li, err := listimgdom.New(
		imageID,
		listID,
		urlStr,
		objectPath,
		fileName,
		raw.Size,
		raw.DisplayOrder,
	)
	if err != nil {
		// best-effort fallback (do not break reads)
		return listimgdom.ListImage{
			ID:           imageID,
			ListID:       listID,
			URL:          urlStr,
			ObjectPath:   objectPath,
			FileName:     fileName,
			Size:         raw.Size,
			DisplayOrder: raw.DisplayOrder,
		}, true
	}

	return li, true
}

// ============================================================
// Canonical path helpers (legacy removed)
// ============================================================

func looksLikeCanonicalObjectPath(s string) bool {
	p := strings.TrimLeft(strings.TrimSpace(s), "/")
	if p == "" {
		return false
	}
	parts := strings.Split(p, "/")
	// must be exactly: ["lists", "{listId}", "images", "{imageId}"]
	if len(parts) != 4 {
		return false
	}
	return strings.TrimSpace(parts[0]) == "lists" &&
		strings.TrimSpace(parts[1]) != "" &&
		strings.TrimSpace(parts[2]) == "images" &&
		strings.TrimSpace(parts[3]) != ""
}

func splitCanonicalObjectPath(objectPath string) (listID string, imageID string, ok bool) {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) != 4 {
		return "", "", false
	}
	if strings.TrimSpace(parts[0]) != "lists" || strings.TrimSpace(parts[2]) != "images" {
		return "", "", false
	}
	listID = strings.TrimSpace(parts[1])
	imageID = strings.TrimSpace(parts[3])
	if listID == "" || imageID == "" {
		return "", "", false
	}
	return listID, imageID, true
}

func isCanonicalObjectPath(objectPath string, listID string, imageID string) bool {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)
	if p == "" || listID == "" || imageID == "" {
		return false
	}
	want := "lists/" + listID + "/images/" + imageID
	return p == want
}
