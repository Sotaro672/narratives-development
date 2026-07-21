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

	listdom "narratives/internal/domain/list"
)

type ListImageRepositoryFS struct {
	Client *gfs.Client
}

func NewListImageRepositoryFS(
	client *gfs.Client,
) *ListImageRepositoryFS {
	return &ListImageRepositoryFS{
		Client: client,
	}
}

var _ listdom.ImageRepository = (*ListImageRepositoryFS)(nil)

func (r *ListImageRepositoryFS) listCol(
	listID string,
) *gfs.CollectionRef {
	return r.Client.
		Collection("lists").
		Doc(listID).
		Collection("images")
}

// ============================================================
// Query
// ============================================================

func (r *ListImageRepositoryFS) GetByID(
	ctx context.Context,
	listID string,
	imageID string,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New(
			"firestore client is nil",
		)
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") ||
		strings.Contains(imageID, "://") {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	doc, err := r.listCol(listID).Doc(imageID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.ListImage{}, listdom.ErrNotFound
		}

		return listdom.ListImage{}, err
	}

	img, ok := decodeListImageDoc(doc, listID)
	if !ok {
		return listdom.ListImage{}, listdom.ErrNotFound
	}

	return img, nil
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

// ============================================================
// Write
// ============================================================

func (r *ListImageRepositoryFS) Create(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New(
			"firestore client is nil",
		)
	}

	img.ListID = strings.TrimSpace(img.ListID)
	img.ID = strings.TrimSpace(img.ID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ListID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	if img.ID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	if strings.Contains(img.ID, "/") ||
		strings.Contains(img.ID, "://") {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	if img.URL == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageURL
	}

	if img.CreatedBy == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageCreatedBy
	}

	if img.DisplayOrder < 0 {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageDisplayOrder
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	} else {
		img.CreatedAt = img.CreatedAt.UTC()
	}

	if img.UpdatedAt != nil {
		if img.UpdatedAt.IsZero() {
			img.UpdatedAt = nil
		} else {
			updatedAt := img.UpdatedAt.UTC()
			img.UpdatedAt = &updatedAt
		}
	}

	if img.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*img.UpdatedBy)

		if updatedBy == "" {
			img.UpdatedBy = nil
		} else {
			img.UpdatedBy = &updatedBy
		}
	}

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, err
	}

	ref := r.listCol(img.ListID).Doc(img.ID)

	var created listdom.ListImage

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			doc, err := tx.Get(ref)

			if err == nil {
				existing, ok := decodeListImageDoc(
					doc,
					img.ListID,
				)
				if !ok {
					return listdom.ErrConflict
				}

				if !equivalentListImageCreate(
					existing,
					img,
				) {
					return listdom.ErrConflict
				}

				created = existing
				return nil
			}

			if status.Code(err) != codes.NotFound {
				return err
			}

			if err := tx.Create(
				ref,
				encodeListImageDoc(img),
			); err != nil {
				if status.Code(err) == codes.AlreadyExists {
					return listdom.ErrConflict
				}

				return err
			}

			created = img
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, listdom.ErrConflict) {
			return listdom.ListImage{},
				listdom.ErrConflict
		}

		return listdom.ListImage{}, err
	}

	return created, nil
}

func (r *ListImageRepositoryFS) Update(
	ctx context.Context,
	listID string,
	imageID string,
	patch listdom.ListImagePatch,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{}, errors.New(
			"firestore client is nil",
		)
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") ||
		strings.Contains(imageID, "://") {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).Doc(imageID)

	var updated listdom.ListImage

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			doc, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return listdom.ErrNotFound
				}

				return err
			}

			current, ok := decodeListImageDoc(
				doc,
				listID,
			)
			if !ok {
				return listdom.ErrNotFound
			}

			changed := false
			clearUpdatedAt := false
			clearUpdatedBy := false

			if patch.URL != nil {
				value := strings.TrimSpace(*patch.URL)
				if value == "" {
					return listdom.ErrInvalidListImageURL
				}

				current.URL = value
				changed = true
			}

			if patch.DisplayOrder != nil {
				if *patch.DisplayOrder < 0 {
					return listdom.
						ErrInvalidListImageDisplayOrder
				}

				current.DisplayOrder =
					*patch.DisplayOrder

				changed = true
			}

			if patch.UpdatedBy != nil {
				value := strings.TrimSpace(
					*patch.UpdatedBy,
				)

				if value == "" {
					current.UpdatedBy = nil
					clearUpdatedBy = true
				} else {
					current.UpdatedBy = &value
				}

				changed = true
			}

			if patch.UpdatedAt != nil {
				if patch.UpdatedAt.IsZero() {
					current.UpdatedAt = nil
					clearUpdatedAt = true
				} else {
					updatedAt :=
						patch.UpdatedAt.UTC()

					current.UpdatedAt = &updatedAt
				}

				changed = true
			} else if changed {
				updatedAt := time.Now().UTC()
				current.UpdatedAt = &updatedAt
			}

			if !changed {
				updated = current
				return nil
			}

			if err := current.Validate(); err != nil {
				return err
			}

			data := encodeListImageDoc(current)

			if clearUpdatedAt {
				data["updated_at"] = gfs.Delete
			}

			if clearUpdatedBy {
				data["updated_by"] = gfs.Delete
			}

			if err := tx.Set(
				ref,
				data,
				gfs.MergeAll,
			); err != nil {
				return err
			}

			updated = current
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, listdom.ErrNotFound) {
			return listdom.ListImage{},
				listdom.ErrNotFound
		}

		return listdom.ListImage{}, err
	}

	return updated, nil
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

	if strings.Contains(imageID, "/") ||
		strings.Contains(imageID, "://") {
		return listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).Doc(imageID)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			_, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return nil
				}

				return err
			}

			return tx.Delete(ref)
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// ============================================================
// Idempotency helpers
// ============================================================

// equivalentListImageCreate compares fields supplied by the image creation
// request.
//
// CreatedAt is intentionally excluded because a retried HTTP request may
// generate a new current timestamp while representing the same logical
// creation request.
func equivalentListImageCreate(
	existing listdom.ListImage,
	incoming listdom.ListImage,
) bool {
	return existing.ID == incoming.ID &&
		existing.ListID == incoming.ListID &&
		existing.URL == incoming.URL &&
		existing.DisplayOrder == incoming.DisplayOrder &&
		existing.CreatedBy == incoming.CreatedBy
}

// ============================================================
// Firestore encode/decode
// ============================================================

func encodeListImageDoc(
	img listdom.ListImage,
) map[string]any {
	data := map[string]any{
		"id":            img.ID,
		"list_id":       img.ListID,
		"url":           img.URL,
		"display_order": img.DisplayOrder,
		"created_at":    img.CreatedAt.UTC(),
		"created_by":    img.CreatedBy,
	}

	if img.UpdatedAt != nil &&
		!img.UpdatedAt.IsZero() {
		data["updated_at"] = img.UpdatedAt.UTC()
	}

	if img.UpdatedBy != nil {
		if value := strings.TrimSpace(
			*img.UpdatedBy,
		); value != "" {
			data["updated_by"] = value
		}
	}

	return data
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

	if strings.Contains(imageID, "/") ||
		strings.Contains(imageID, "://") {
		return listdom.ListImage{}, false
	}

	imageURL := strings.TrimSpace(raw.URL)
	if imageURL == "" {
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
		imageURL,
		displayOrder,
		createdAt,
		createdBy,
	)
	if err != nil {
		return listdom.ListImage{}, false
	}

	if raw.UpdatedAt != nil &&
		!raw.UpdatedAt.IsZero() {
		updatedAt := raw.UpdatedAt.UTC()
		img.UpdatedAt = &updatedAt
	}

	if raw.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(
			*raw.UpdatedBy,
		)

		if updatedBy != "" {
			img.UpdatedBy = &updatedBy
		}
	}

	return img, true
}
