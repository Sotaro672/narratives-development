// backend\internal\adapters\out\firestore\list_image_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	listuc "narratives/internal/application/usecase/list"
	listimgdom "narratives/internal/domain/listImage"
)

// Firestore schema (recommended):
// - lists/{listId}/images/{imageId}
// fields:
// - id            : string   (recommended: objectPath "{listId}/{imageId}/{fileName}")
// - list_id       : string
// - url           : string
// - file_name     : string
// - size          : number
// - display_order : number
// - bucket        : string   (optional)
// - object_path   : string   (optional)
// - created_at    : timestamp
// - updated_at    : timestamp
//
// NOTE:
// - docID is "imageId" (2nd segment of objectPath).
// - "id" field can store objectPath for easy GCS deletion & tracing.

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

// ============================================================
// Port: ListImageRecordRepository
// ============================================================

// Upsert stores list image record into Firestore subcollection.
// docID policy: imageId (derived from objectPath "{listId}/{imageId}/{fileName}")
func (r *ListImageRepositoryFS) Upsert(ctx context.Context, img listimgdom.ListImage) (listimgdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimgdom.ListImage{}, errors.New("firestore client is nil")
	}

	listID := strings.TrimSpace(img.ListID)
	if listID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidListID
	}

	// Determine bucket/objectPath as best-effort
	bucket := ""
	objectPath := ""

	// Prefer img.ID when it looks like an objectPath
	if looksLikeObjectPath(img.ID) {
		objectPath = strings.TrimLeft(strings.TrimSpace(img.ID), "/")
	}

	// If URL is a GCS public URL, parse bucket/objectPath
	if bucket == "" || objectPath == "" {
		if b, obj, ok := listimgdom.ParseGCSURL(strings.TrimSpace(img.URL)); ok {
			if bucket == "" {
				bucket = strings.TrimSpace(b)
			}
			if objectPath == "" {
				objectPath = strings.TrimLeft(strings.TrimSpace(obj), "/")
			}
		}
	}

	// Derive imageId (docID)
	imageID := ""
	if objectPath != "" {
		imageID = extractImageIDFromObjectPath(objectPath)
	}
	if imageID == "" {
		// fallback: allow "img.ID is imageId" pattern
		if !strings.Contains(strings.TrimSpace(img.ID), "/") {
			imageID = strings.TrimSpace(img.ID)
		}
	}
	if imageID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidID
	}

	// fileName
	fileName := strings.TrimSpace(img.FileName)
	if fileName == "" && objectPath != "" {
		fileName = path.Base(objectPath)
	}
	if fileName == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidFileName
	}

	// url
	u := strings.TrimSpace(img.URL)
	if u == "" {
		// best-effort: if we know bucket/objectPath, rebuild public URL
		if bucket != "" && objectPath != "" {
			u = listimgdom.PublicURL(bucket, objectPath)
		}
	}
	if strings.TrimSpace(u) == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidURL
	}

	// id field stored in firestore: recommend objectPath (stable, supports deletion/debug)
	storedID := strings.TrimSpace(img.ID)
	if storedID == "" && objectPath != "" {
		storedID = objectPath
	}
	// if storedID is a GCS URL, store objectPath instead (keeps ID compact)
	if _, obj, ok := listimgdom.ParseGCSURL(storedID); ok {
		storedID = strings.TrimLeft(strings.TrimSpace(obj), "/")
	}

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
			"object_path": strings.TrimLeft(strings.TrimSpace(objectPath), "/"),

			"created_at": createdAt,
			"updated_at": now,
		}

		return tx.Set(ref, data, gfs.MergeAll)
	})
	if err != nil {
		return listimgdom.ListImage{}, err
	}

	// Return domain object (best-effort validation)
	out, derr := listimgdom.New(
		storedID,
		listID,
		u,
		fileName,
		img.Size,
		img.DisplayOrder,
	)
	if derr != nil {
		return listimgdom.ListImage{
			ID:           storedID,
			ListID:       listID,
			URL:          u,
			FileName:     fileName,
			Size:         img.Size,
			DisplayOrder: img.DisplayOrder,
		}, nil
	}
	return out, nil
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

// GetByID supports the following id forms:
// - objectPath: "{listId}/{imageId}/{fileName}"  => direct read
// - URL: "https://storage.googleapis.com/{bucket}/{listId}/{imageId}/{fileName}" => direct read
// - imageId only: "{imageId}" => collectionGroup query by DocumentID
func (r *ListImageRepositoryFS) GetByID(ctx context.Context, id string) (listimgdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimgdom.ListImage{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return listimgdom.ListImage{}, listimgdom.ErrNotFound
	}

	// 1) If URL, parse to objectPath
	if _, obj, ok := listimgdom.ParseGCSURL(id); ok {
		id = strings.TrimLeft(strings.TrimSpace(obj), "/")
	}

	// 2) If objectPath-like, direct read by listId + imageId
	if looksLikeObjectPath(id) {
		listID, imageID, ok := splitListImageObjectPath(id)
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

	// 3) Fallback: treat as imageId and search by DocumentID in collection group
	imageID := id
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
	if doc == nil {
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
		Bucket       string    `firestore:"bucket"`
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

	// id
	id := strings.TrimSpace(raw.ID)
	if id == "" {
		// prefer object_path
		if strings.TrimSpace(raw.ObjectPath) != "" {
			id = strings.TrimLeft(strings.TrimSpace(raw.ObjectPath), "/")
		} else {
			// as last resort, use docID
			id = strings.TrimSpace(doc.Ref.ID)
		}
	}

	url := strings.TrimSpace(raw.URL)
	if url == "" {
		// best-effort rebuild if possible
		if strings.TrimSpace(raw.Bucket) != "" && strings.TrimSpace(raw.ObjectPath) != "" {
			url = listimgdom.PublicURL(strings.TrimSpace(raw.Bucket), strings.TrimLeft(strings.TrimSpace(raw.ObjectPath), "/"))
		}
	}

	fileName := strings.TrimSpace(raw.FileName)
	if fileName == "" && strings.TrimSpace(raw.ObjectPath) != "" {
		fileName = path.Base(strings.TrimLeft(strings.TrimSpace(raw.ObjectPath), "/"))
	}

	li, err := listimgdom.New(
		id,
		listID,
		url,
		fileName,
		raw.Size,
		raw.DisplayOrder,
	)
	if err != nil {
		// best-effort fallback
		return listimgdom.ListImage{
			ID:           id,
			ListID:       listID,
			URL:          url,
			FileName:     fileName,
			Size:         raw.Size,
			DisplayOrder: raw.DisplayOrder,
		}, true
	}

	return li, true
}

// ============================================================
// Path helpers
// ============================================================

// looksLikeObjectPath returns true when it resembles "{listId}/{imageId}/...".
func looksLikeObjectPath(s string) bool {
	p := strings.TrimLeft(strings.TrimSpace(s), "/")
	parts := strings.Split(p, "/")
	return len(parts) >= 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != ""
}

// splitListImageObjectPath expects "{listId}/{imageId}/{fileName}" (at least 3 segments).
func splitListImageObjectPath(objectPath string) (listID string, imageID string, ok bool) {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		return "", "", false
	}
	listID = strings.TrimSpace(parts[0])
	imageID = strings.TrimSpace(parts[1])
	if listID == "" || imageID == "" {
		return "", "", false
	}
	return listID, imageID, true
}

// extractImageIDFromObjectPath expects "{listId}/{imageId}/{fileName}" and returns imageId.
func extractImageIDFromObjectPath(objectPath string) string {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
