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

var _ listdom.ImageRepository = (*ListImageRepositoryFS)(nil)

func (r *ListImageRepositoryFS) FindByListID(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	return r.ListByListID(ctx, listID)
}

func (r *ListImageRepositoryFS) Create(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New("firestore client is nil")
	}

	img.ID = strings.TrimSpace(img.ID)
	img.ListID = strings.TrimSpace(img.ListID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ListID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if img.ID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(img.ID, "/") || strings.Contains(img.ID, "://") {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if img.URL == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageURL
	}

	if img.CreatedBy == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageCreatedBy
	}

	if img.DisplayOrder < 0 {
		return listdom.ListImage{}, listdom.ErrInvalidListImageDisplayOrder
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	} else {
		img.CreatedAt = img.CreatedAt.UTC()
	}

	if img.UpdatedAt != nil && !img.UpdatedAt.IsZero() {
		t := img.UpdatedAt.UTC()
		img.UpdatedAt = &t
	}

	if img.UpdatedBy != nil {
		v := strings.TrimSpace(*img.UpdatedBy)
		if v == "" {
			img.UpdatedBy = nil
		} else {
			img.UpdatedBy = &v
		}
	}

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, err
	}

	ref := r.listCol(img.ListID).Doc(img.ID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err == nil {
			return listdom.ErrListImageConflict
		}

		if status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(ref, encodeListImageDoc(img)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return listdom.ErrListImageConflict
			}
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, listdom.ErrListImageConflict) {
			return listdom.ListImage{}, listdom.ErrListImageConflict
		}
		return listdom.ListImage{}, err
	}

	return r.GetByListIDAndID(ctx, img.ListID, img.ID)
}

func (r *ListImageRepositoryFS) Update(
	ctx context.Context,
	listID string,
	imageID string,
	patch listdom.ListImagePatch,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).Doc(imageID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return listdom.ErrListImageNotFound
			}
			return err
		}

		cur, ok := decodeListImageDoc(doc, listID)
		if !ok {
			return listdom.ErrListImageNotFound
		}

		changed := false
		clearUpdatedAt := false
		clearUpdatedBy := false

		if patch.URL != nil {
			v := strings.TrimSpace(*patch.URL)
			if v == "" {
				return listdom.ErrInvalidListImageURL
			}
			cur.URL = v
			changed = true
		}

		if patch.DisplayOrder != nil {
			if *patch.DisplayOrder < 0 {
				return listdom.ErrInvalidListImageDisplayOrder
			}
			cur.DisplayOrder = *patch.DisplayOrder
			changed = true
		}

		if patch.UpdatedBy != nil {
			v := strings.TrimSpace(*patch.UpdatedBy)
			if v == "" {
				cur.UpdatedBy = nil
				clearUpdatedBy = true
			} else {
				cur.UpdatedBy = &v
			}
			changed = true
		}

		if patch.UpdatedAt != nil {
			if patch.UpdatedAt.IsZero() {
				cur.UpdatedAt = nil
				clearUpdatedAt = true
			} else {
				t := patch.UpdatedAt.UTC()
				cur.UpdatedAt = &t
			}
			changed = true
		} else if changed {
			t := time.Now().UTC()
			cur.UpdatedAt = &t
		}

		if !changed {
			return nil
		}

		if err := cur.Validate(); err != nil {
			return err
		}

		data := encodeListImageDoc(cur)

		if clearUpdatedAt {
			data["updated_at"] = gfs.Delete
		}

		if clearUpdatedBy {
			data["updated_by"] = gfs.Delete
		}

		if err := tx.Set(ref, data, gfs.MergeAll); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, listdom.ErrListImageNotFound) {
			return listdom.ListImage{}, listdom.ErrListImageNotFound
		}
		return listdom.ListImage{}, err
	}

	return r.GetByListIDAndID(ctx, listID, imageID)
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
		return listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).Doc(imageID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return listdom.ErrListImageNotFound
			}
			return err
		}

		return tx.Delete(ref)
	})
	if err != nil {
		if errors.Is(err, listdom.ErrListImageNotFound) {
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

func encodeListImageDoc(img listdom.ListImage) map[string]any {
	m := map[string]any{
		"id":            img.ID,
		"list_id":       img.ListID,
		"url":           img.URL,
		"display_order": img.DisplayOrder,
		"created_at":    img.CreatedAt.UTC(),
		"created_by":    img.CreatedBy,
	}

	if img.UpdatedAt != nil && !img.UpdatedAt.IsZero() {
		m["updated_at"] = img.UpdatedAt.UTC()
	}

	if img.UpdatedBy != nil {
		if v := strings.TrimSpace(*img.UpdatedBy); v != "" {
			m["updated_by"] = v
		}
	}

	return m
}

func decodeListImageDoc(
	doc *gfs.DocumentSnapshot,
	fallbackListID string,
) (listdom.ListImage, bool) {
	if doc == nil || doc.Ref == nil {
		return listdom.ListImage{}, false
	}

	var raw struct {
		ID           string     `firestore:"id"`
		ListID       string     `firestore:"list_id"`
		URL          string     `firestore:"url"`
		DisplayOrder int        `firestore:"display_order"`
		CreatedAt    time.Time  `firestore:"created_at"`
		CreatedBy    string     `firestore:"created_by"`
		UpdatedAt    *time.Time `firestore:"updated_at"`
		UpdatedBy    *string    `firestore:"updated_by"`
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

	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		updatedAt := raw.UpdatedAt.UTC()
		img.UpdatedAt = &updatedAt
	}

	if raw.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*raw.UpdatedBy)
		if updatedBy != "" {
			img.UpdatedBy = &updatedBy
		}
	}

	return img, true
}
