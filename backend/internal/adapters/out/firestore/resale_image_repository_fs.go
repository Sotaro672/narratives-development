// backend/internal/adapters/out/firestore/resale_image_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	resaledom "narratives/internal/domain/resale"
)

// ResaleImageRepositoryFS implements resale.ImageRepository using Firestore.
//
// Collection:
// - resales/{resaleId}/conditionImages/{imageId}
//
// Image policy:
// - Backend stores only Firebase Storage download URL.
// - Backend does not manage objectPath, fileName, contentType, or size.
// - Image record is scoped by resaleId.
// - imageID alone should not be used as a global lookup key.
type ResaleImageRepositoryFS struct {
	Client *gfs.Client
}

func NewResaleImageRepositoryFS(client *gfs.Client) *ResaleImageRepositoryFS {
	return &ResaleImageRepositoryFS{Client: client}
}

var _ resaledom.ImageRepository = (*ResaleImageRepositoryFS)(nil)

func (r *ResaleImageRepositoryFS) conditionImagesCol(
	resaleID string,
) *gfs.CollectionRef {
	return r.Client.
		Collection("resales").
		Doc(resaleID).
		Collection(resaleConditionImagesSub)
}

// ============================================================
// Resale image query
// ============================================================

func (r *ResaleImageRepositoryFS) ListByResaleID(
	ctx context.Context,
	resaleID string,
) ([]resaledom.ResaleImage, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	resaleID = strings.TrimSpace(resaleID)
	if resaleID == "" {
		return []resaledom.ResaleImage{}, nil
	}

	it := r.conditionImagesCol(resaleID).
		OrderBy("display_order", gfs.Asc).
		OrderBy(gfs.DocumentID, gfs.Asc).
		Documents(ctx)
	defer it.Stop()

	out := make([]resaledom.ResaleImage, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		img, ok := decodeResaleImageDoc(doc, resaleID)
		if !ok {
			continue
		}

		out = append(out, img)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DisplayOrder == out[j].DisplayOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].DisplayOrder < out[j].DisplayOrder
	})

	return out, nil
}

// ============================================================
// Resale image write
// ============================================================

func (r *ResaleImageRepositoryFS) Create(
	ctx context.Context,
	img resaledom.ResaleImage,
) (resaledom.ResaleImage, error) {
	if r == nil || r.Client == nil {
		return resaledom.ResaleImage{}, errors.New("firestore client is nil")
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
		return resaledom.ResaleImage{}, err
	}

	ref := r.conditionImagesCol(img.ResaleID).Doc(img.ID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err == nil {
			return resaledom.ErrConditionImageConflict
		}

		if status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(ref, encodeResaleImageDoc(img)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return resaledom.ErrConditionImageConflict
			}
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrConditionImageConflict) {
			return resaledom.ResaleImage{}, resaledom.ErrConditionImageConflict
		}
		return resaledom.ResaleImage{}, err
	}

	return img, nil
}

func (r *ResaleImageRepositoryFS) Update(
	ctx context.Context,
	resaleID string,
	imageID string,
	patch resaledom.ResaleImagePatch,
) (resaledom.ResaleImage, error) {
	if r == nil || r.Client == nil {
		return resaledom.ResaleImage{}, errors.New("firestore client is nil")
	}

	resaleID = strings.TrimSpace(resaleID)
	imageID = strings.TrimSpace(imageID)

	if resaleID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageResaleID
	}

	if imageID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageID
	}

	ref := r.conditionImagesCol(resaleID).Doc(imageID)

	var updated resaledom.ResaleImage

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resaledom.ErrConditionImageNotFound
			}
			return err
		}

		cur, ok := decodeResaleImageDoc(doc, resaleID)
		if !ok {
			return resaledom.ErrConditionImageNotFound
		}

		changed := false
		clearUpdatedAt := false
		clearUpdatedBy := false

		if patch.URL != nil {
			v := strings.TrimSpace(*patch.URL)
			if err := cur.UpdateURL(v); err != nil {
				return err
			}
			changed = true
		}

		if patch.DisplayOrder != nil {
			if err := cur.SetDisplayOrder(*patch.DisplayOrder); err != nil {
				return err
			}
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
			updated = cur
			return nil
		}

		if err := cur.Validate(); err != nil {
			return err
		}

		data := encodeResaleImageDoc(cur)

		if clearUpdatedAt {
			data["updated_at"] = gfs.Delete
		}

		if clearUpdatedBy {
			data["updated_by"] = gfs.Delete
		}

		if err := tx.Set(ref, data, gfs.MergeAll); err != nil {
			return err
		}

		updated = cur
		return nil
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrConditionImageNotFound) {
			return resaledom.ResaleImage{}, resaledom.ErrConditionImageNotFound
		}
		return resaledom.ResaleImage{}, err
	}

	return updated, nil
}

func (r *ResaleImageRepositoryFS) Delete(
	ctx context.Context,
	resaleID string,
	imageID string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	resaleID = strings.TrimSpace(resaleID)
	imageID = strings.TrimSpace(imageID)

	if resaleID == "" {
		return resaledom.ErrInvalidConditionImageResaleID
	}

	if imageID == "" {
		return resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.ErrInvalidConditionImageID
	}

	ref := r.conditionImagesCol(resaleID).Doc(imageID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return resaledom.ErrConditionImageNotFound
			}
			return err
		}

		return tx.Delete(ref)
	})
	if err != nil {
		if errors.Is(err, resaledom.ErrConditionImageNotFound) {
			return resaledom.ErrConditionImageNotFound
		}
		return err
	}

	return nil
}

// ============================================================
// Firestore encode/decode - resale image
// ============================================================

func encodeResaleImageDoc(img resaledom.ResaleImage) map[string]any {
	m := map[string]any{
		"id":            img.ID,
		"resale_id":     img.ResaleID,
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

func decodeResaleImageDoc(
	doc *gfs.DocumentSnapshot,
	fallbackResaleID string,
) (resaledom.ResaleImage, bool) {
	if doc == nil || doc.Ref == nil {
		return resaledom.ResaleImage{}, false
	}

	var raw struct {
		ID           string     `firestore:"id"`
		ResaleID     string     `firestore:"resale_id"`
		URL          string     `firestore:"url"`
		DisplayOrder int        `firestore:"display_order"`
		CreatedAt    time.Time  `firestore:"created_at"`
		CreatedBy    string     `firestore:"created_by"`
		UpdatedAt    *time.Time `firestore:"updated_at"`
		UpdatedBy    *string    `firestore:"updated_by"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return resaledom.ResaleImage{}, false
	}

	resaleID := raw.ResaleID
	if resaleID == "" {
		resaleID = fallbackResaleID
	}

	if resaleID == "" {
		return resaledom.ResaleImage{}, false
	}

	imageID := doc.Ref.ID
	if imageID == "" {
		imageID = raw.ID
	}

	if imageID == "" {
		return resaledom.ResaleImage{}, false
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.ResaleImage{}, false
	}

	createdAt := raw.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	} else {
		createdAt = createdAt.UTC()
	}

	img, err := resaledom.NewResaleImage(
		imageID,
		resaleID,
		raw.URL,
		raw.DisplayOrder,
		createdAt,
		raw.CreatedBy,
	)
	if err != nil {
		return resaledom.ResaleImage{}, false
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

	if err := img.Validate(); err != nil {
		return resaledom.ResaleImage{}, false
	}

	return img, true
}
