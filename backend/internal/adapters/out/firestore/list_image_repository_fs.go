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
	listdom "narratives/internal/domain/list"
)

type ListImageRepositoryFS struct {
	Client *gfs.Client
}

func NewListImageRepositoryFS(client *gfs.Client) *ListImageRepositoryFS {
	return &ListImageRepositoryFS{Client: client}
}

func (r *ListImageRepositoryFS) listCol(listID string) *gfs.CollectionRef {
	return r.Client.Collection("lists").Doc(listID).Collection("images")
}

var _ usecase.ListImageReader = (*ListImageRepositoryFS)(nil)
var _ usecase.ListImageRecordRepository = (*ListImageRepositoryFS)(nil)
var _ usecase.ListImageRecordByIDReader = (*ListImageRepositoryFS)(nil)

func (r *ListImageRepositoryFS) FindByListID(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	return r.ListByListID(ctx, listID)
}

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

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, usecase.ErrInvalidArgument("invalid_image_id")
	}

	urlStr := strings.TrimSpace(img.URL)
	if urlStr == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageURL
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

	updatedAt := now
	if img.UpdatedAt != nil && !img.UpdatedAt.IsZero() {
		updatedAt = img.UpdatedAt.UTC()
	}

	updatedBy := ""
	if img.UpdatedBy != nil {
		updatedBy = strings.TrimSpace(*img.UpdatedBy)
	}

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
			"display_order": displayOrder,
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
		displayOrder,
		createdAt,
		createdBy,
	)
	if err != nil {
		return listdom.ListImage{}, err
	}

	out.UpdatedAt = &updatedAt
	if updatedBy != "" {
		out.UpdatedBy = &updatedBy
	}

	return out, nil
}

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

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
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

// GetByID is kept as a best-effort legacy helper.
// New code should prefer GetByListIDAndID because image records are scoped by listId.
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
		DisplayOrder int       `firestore:"display_order"`
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

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, false
	}

	urlStr := strings.TrimSpace(raw.URL)
	if urlStr == "" {
		return listdom.ListImage{}, false
	}

	createdAt := raw.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	} else {
		createdAt = createdAt.UTC()
	}

	createdBy := strings.TrimSpace(raw.CreatedBy)
	if createdBy == "" {
		return listdom.ListImage{}, false
	}

	displayOrder := raw.DisplayOrder
	if displayOrder < 0 {
		displayOrder = 0
	}

	img, err := listdom.NewListImage(
		imageID,
		listID,
		urlStr,
		displayOrder,
		createdAt,
		createdBy,
	)
	if err != nil {
		return listdom.ListImage{}, false
	}

	if !raw.UpdatedAt.IsZero() {
		updatedAt := raw.UpdatedAt.UTC()
		img.UpdatedAt = &updatedAt
	}

	updatedBy := strings.TrimSpace(raw.UpdatedBy)
	if updatedBy != "" {
		img.UpdatedBy = &updatedBy
	}

	return img, true
}
