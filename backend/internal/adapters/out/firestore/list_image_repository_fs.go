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
	listdom "narratives/internal/domain/list"
)

// Firestore schema:
//
//	lists/{listId}/images/{imageId}
//
// fields:
// - id            : string
// - list_id       : string
// - url           : string   Firebase Storage downloadURL
// - file_name     : string
// - content_type  : string
// - size          : number
// - display_order : number
// - object_path   : string   Firebase Storage object path
// - created_at    : timestamp
// - created_by    : string
// - updated_at    : timestamp
// - updated_by    : string
//
// Firebase Storage migration policy:
// - backend does not issue signed URLs
// - backend does not use GCS bucket / public URL fallback
// - frontend uploads directly to Firebase Storage
// - frontend sends downloadURL / objectPath / fileName / contentType / size
// - repository persists only Firestore image records
//
// Canonical Firebase Storage objectPath:
//
//	lists/{listId}/images/{imageId}/{fileName}

type ListImageRepositoryFS struct {
	Client *gfs.Client
}

func NewListImageRepositoryFS(client *gfs.Client) *ListImageRepositoryFS {
	return &ListImageRepositoryFS{Client: client}
}

func (r *ListImageRepositoryFS) listCol(listID string) *gfs.CollectionRef {
	return r.Client.Collection("lists").Doc(listID).Collection("images")
}

// compile-time checks
var _ listuc.ListImageReader = (*ListImageRepositoryFS)(nil)
var _ listuc.ListImageByIDReader = (*ListImageRepositoryFS)(nil)
var _ listuc.ListImageRecordRepository = (*ListImageRepositoryFS)(nil)
var _ listuc.ListImageRecordByIDReader = (*ListImageRepositoryFS)(nil)

// ============================================================
// Port: CatalogQuery ListImageRepository compatibility
// ============================================================

// FindByListID is for mall catalog query layer.
// It returns all images for a list ordered by displayOrder asc.
func (r *ListImageRepositoryFS) FindByListID(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	return r.ListByListID(ctx, listID)
}

// ============================================================
// Port: ListImageRecordRepository
// ============================================================

// Upsert stores list image record into Firestore subcollection.
//
// docID policy:
// - imageId is Firestore docID
//
// Firebase Storage policy:
// - URL is Firebase Storage downloadURL
// - ObjectPath is Firebase Storage objectPath
// - ObjectPath must be: lists/{listId}/images/{imageId}/{fileName}
func (r *ListImageRepositoryFS) Upsert(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New("firestore client is nil")
	}

	listID := strings.TrimSpace(img.ListID)
	if listID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	imageID := strings.TrimSpace(img.ID)
	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}
	if strings.Contains(imageID, "/") {
		return listdom.ListImage{}, usecase.ErrInvalidArgument("invalid_image_id")
	}

	objectPath := strings.TrimLeft(strings.TrimSpace(img.ObjectPath), "/")
	if objectPath == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageObjectPath
	}
	if !isCanonicalFirebaseStorageObjectPath(objectPath, listID, imageID) {
		return listdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_not_canonical")
	}

	fileName := strings.TrimSpace(img.FileName)
	if fileName == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageFileName
	}

	urlStr := strings.TrimSpace(img.URL)
	if urlStr == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageURL
	}

	contentType := strings.ToLower(strings.TrimSpace(img.ContentType))
	if contentType == "" {
		contentType = inferImageContentTypeFromFileName(fileName)
	}

	createdBy := strings.TrimSpace(img.CreatedBy)
	if createdBy == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageCreatedBy
	}

	displayOrder := img.DisplayOrder
	if displayOrder < 0 {
		displayOrder = 0
	}

	now := time.Now().UTC()

	createdAt := img.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	} else {
		createdAt = createdAt.UTC()
	}

	updatedAt := img.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	} else {
		updatedAt = updatedAt.UTC()
	}

	updatedBy := strings.TrimSpace(img.UpdatedBy)

	ref := r.listCol(listID).Doc(imageID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		finalCreatedAt := createdAt
		finalCreatedBy := createdBy

		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
		} else {
			data := snap.Data()

			if v, ok := data["created_at"].(time.Time); ok && !v.IsZero() {
				finalCreatedAt = v.UTC()
			}

			if v, ok := data["created_by"].(string); ok && strings.TrimSpace(v) != "" {
				finalCreatedBy = strings.TrimSpace(v)
			}
		}

		data := map[string]any{
			"id":            imageID,
			"list_id":       listID,
			"url":           urlStr,
			"file_name":     fileName,
			"content_type":  contentType,
			"size":          img.Size,
			"display_order": displayOrder,
			"object_path":   objectPath,
			"created_at":    finalCreatedAt,
			"created_by":    finalCreatedBy,
			"updated_at":    updatedAt,
			"updated_by":    updatedBy,
		}

		return tx.Set(ref, data, gfs.MergeAll)
	})
	if err != nil {
		return listdom.ListImage{}, err
	}

	out, err := listdom.NewListImage(
		imageID,
		listID,
		urlStr,
		objectPath,
		fileName,
		contentType,
		img.Size,
		displayOrder,
		createdAt,
		createdBy,
	)
	if err != nil {
		return listdom.ListImage{}, err
	}

	out.UpdatedAt = updatedAt
	out.UpdatedBy = updatedBy

	return out, nil
}

// Delete deletes Firestore record:
//
//	lists/{listId}/images/{imageId}
//
// Firebase Storage object deletion is intentionally not handled here.
func (r *ListImageRepositoryFS) Delete(
	ctx context.Context,
	listID string,
	imageID string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ErrInvalidListImageListID
	}
	if imageID == "" {
		return listdom.ErrInvalidListImageID
	}
	if strings.Contains(imageID, "/") {
		return usecase.ErrInvalidArgument("invalid_image_id")
	}

	ref := r.listCol(listID).Doc(imageID)

	_, err := ref.Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.ErrListImageNotFound
		}
		return err
	}

	return nil
}

// ============================================================
// Port: ListImageReader
// ============================================================

// ListByListID returns images for a list ordered by displayOrder asc.
func (r *ListImageRepositoryFS) ListByListID(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listdom.ListImage{}, nil
	}

	it := r.listCol(listID).
		OrderBy("display_order", gfs.Asc).
		OrderBy(gfs.DocumentID, gfs.Asc).
		Documents(ctx)
	defer it.Stop()

	out := make([]listdom.ListImage, 0, 8)

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
// Port: ListImageByIDReader / ListImageRecordByIDReader
// ============================================================

// GetByListIDAndID gets a ListImage by listId + imageId.
//
// This is the preferred method for primary image update because the caller
// already knows listID. It directly reads:
//
//	lists/{listId}/images/{imageId}
//
// No CollectionGroup query is needed.
func (r *ListImageRepositoryFS) GetByListIDAndID(
	ctx context.Context,
	listID string,
	imageID string,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	doc, err := r.listCol(listID).Doc(imageID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.ListImage{}, listdom.ErrListImageNotFound
		}
		return listdom.ListImage{}, err
	}

	img, ok := decodeListImageDoc(doc, listID)
	if !ok {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	return img, nil
}

// GetByID gets a ListImage by imageId only.
//
// Compatibility method for older callers.
//
// - URL input is not supported
// - objectPath input is not supported
// - id must be Firestore image docID
//
// Note:
//   - This intentionally avoids Where(gfs.DocumentID, "==", imageID)
//     because that can produce "__key__ filter value must be a Key".
//   - Prefer GetByListIDAndID when listID is known.
func (r *ListImageRepositoryFS) GetByID(
	ctx context.Context,
	imageID string,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New("firestore client is nil")
	}

	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}
	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	q := r.Client.CollectionGroup("images").
		Where("id", "==", imageID).
		Limit(1)

	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return listdom.ListImage{}, listdom.ErrListImageNotFound
		}
		return listdom.ListImage{}, err
	}

	listID := ""
	if doc != nil &&
		doc.Ref != nil &&
		doc.Ref.Parent != nil &&
		doc.Ref.Parent.Parent != nil {
		listID = doc.Ref.Parent.Parent.ID
	}

	img, ok := decodeListImageDoc(doc, listID)
	if !ok {
		return listdom.ListImage{}, listdom.ErrListImageNotFound
	}

	return img, nil
}

// ============================================================
// Decode helpers
// ============================================================

func decodeListImageDoc(
	doc *gfs.DocumentSnapshot,
	fallbackListID string,
) (listdom.ListImage, bool) {
	if doc == nil || doc.Ref == nil {
		return listdom.ListImage{}, false
	}

	var raw struct {
		ID           string    `firestore:"id"`
		ListID       string    `firestore:"list_id"`
		URL          string    `firestore:"url"`
		FileName     string    `firestore:"file_name"`
		ContentType  string    `firestore:"content_type"`
		Size         int64     `firestore:"size"`
		DisplayOrder int       `firestore:"display_order"`
		ObjectPath   string    `firestore:"object_path"`
		CreatedAt    time.Time `firestore:"created_at"`
		CreatedBy    string    `firestore:"created_by"`
		UpdatedAt    time.Time `firestore:"updated_at"`
		UpdatedBy    string    `firestore:"updated_by"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return listdom.ListImage{}, false
	}

	listID := strings.TrimSpace(raw.ListID)
	if listID == "" {
		listID = strings.TrimSpace(fallbackListID)
	}
	if listID == "" {
		return listdom.ListImage{}, false
	}

	imageID := strings.TrimSpace(doc.Ref.ID)
	if imageID == "" {
		imageID = strings.TrimSpace(raw.ID)
	}
	if imageID == "" {
		return listdom.ListImage{}, false
	}

	urlStr := strings.TrimSpace(raw.URL)
	if urlStr == "" {
		return listdom.ListImage{}, false
	}

	fileName := strings.TrimSpace(raw.FileName)
	if fileName == "" {
		return listdom.ListImage{}, false
	}

	contentType := strings.ToLower(strings.TrimSpace(raw.ContentType))
	if contentType == "" {
		contentType = inferImageContentTypeFromFileName(fileName)
	}

	objectPath := strings.TrimLeft(strings.TrimSpace(raw.ObjectPath), "/")
	if objectPath == "" {
		objectPath = listdom.CanonicalListImageObjectPath(listID, imageID, fileName)
	}

	if !isCanonicalFirebaseStorageObjectPath(objectPath, listID, imageID) {
		return listdom.ListImage{}, false
	}

	createdAt := raw.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	createdBy := strings.TrimSpace(raw.CreatedBy)
	if createdBy == "" {
		return listdom.ListImage{}, false
	}

	img, err := listdom.NewListImage(
		imageID,
		listID,
		urlStr,
		objectPath,
		fileName,
		contentType,
		raw.Size,
		raw.DisplayOrder,
		createdAt,
		createdBy,
	)
	if err != nil {
		return listdom.ListImage{}, false
	}

	if !raw.UpdatedAt.IsZero() {
		img.UpdatedAt = raw.UpdatedAt.UTC()
	}
	img.UpdatedBy = strings.TrimSpace(raw.UpdatedBy)

	return img, true
}

// ============================================================
// Firebase Storage path helpers
// ============================================================

func isCanonicalFirebaseStorageObjectPath(
	objectPath string,
	listID string,
	imageID string,
) bool {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" || listID == "" || imageID == "" {
		return false
	}

	parts := strings.Split(p, "/")

	// Expected:
	// lists/{listId}/images/{imageId}/{fileName}
	if len(parts) != 5 {
		return false
	}

	if parts[0] != "lists" {
		return false
	}
	if parts[1] != listID {
		return false
	}
	if parts[2] != "images" {
		return false
	}
	if parts[3] != imageID {
		return false
	}
	if parts[4] == "" {
		return false
	}

	return true
}

func inferImageContentTypeFromFileName(fileName string) string {
	name := strings.ToLower(strings.TrimSpace(fileName))

	switch {
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	default:
		return ""
	}
}
