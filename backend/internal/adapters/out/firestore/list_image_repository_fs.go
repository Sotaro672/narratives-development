// backend/internal/adapters/out/firestore/list_image_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
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
		return listdom.ListImage{},
			errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(imageID)
	if !ok {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	doc, err := r.listCol(listID).
		Doc(normalizedImageID).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.ListImage{},
				listdom.ErrNotFound
		}

		return listdom.ListImage{}, err
	}

	img, ok := decodeListImageDoc(
		doc,
		listID,
	)
	if !ok {
		return listdom.ListImage{},
			listdom.ErrNotFound
	}

	return img, nil
}

func (r *ListImageRepositoryFS) ListByListID(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listdom.ListImage{}, nil
	}

	return listOrderedImageDocuments(
		ctx,
		r.listCol(listID),
		func(
			doc *gfs.DocumentSnapshot,
		) (listdom.ListImage, bool) {
			return decodeListImageDoc(
				doc,
				listID,
			)
		},
	)
}

// ============================================================
// Write
// ============================================================

func (r *ListImageRepositoryFS) Create(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listdom.ListImage{},
			errors.New("firestore client is nil")
	}

	img.ListID = strings.TrimSpace(img.ListID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ListID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(img.ID)
	if !ok {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	img.ID = normalizedImageID

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

	img.CreatedAt =
		normalizeImageCreatedAt(img.CreatedAt)

	img.UpdatedAt =
		normalizeImageUpdatedAt(img.UpdatedAt)

	img.UpdatedBy =
		normalizeImageUpdatedBy(img.UpdatedBy)

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
				if status.Code(err) ==
					codes.AlreadyExists {
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
		return listdom.ListImage{},
			errors.New("firestore client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageListID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(imageID)
	if !ok {
		return listdom.ListImage{},
			listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).
		Doc(normalizedImageID)

	var updated listdom.ListImage

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			doc, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
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
				value := strings.TrimSpace(
					*patch.URL,
				)
				if value == "" {
					return listdom.
						ErrInvalidListImageURL
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
				value :=
					normalizeImageUpdatedBy(
						patch.UpdatedBy,
					)

				if value == nil {
					current.UpdatedBy = nil
					clearUpdatedBy = true
				} else {
					current.UpdatedBy = value
				}

				changed = true
			}

			if patch.UpdatedAt != nil {
				value :=
					normalizeImageUpdatedAt(
						patch.UpdatedAt,
					)

				if value == nil {
					current.UpdatedAt = nil
					clearUpdatedAt = true
				} else {
					current.UpdatedAt = value
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
	if listID == "" {
		return listdom.ErrInvalidListImageListID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(imageID)
	if !ok {
		return listdom.ErrInvalidListImageID
	}

	ref := r.listCol(listID).
		Doc(normalizedImageID)

	// List画像の削除は冪等とし、
	// 対象が存在しない場合も成功扱いにする。
	return deleteImageDocument(
		ctx,
		r.Client,
		ref,
		nil,
	)
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
		existing.DisplayOrder ==
			incoming.DisplayOrder &&
		existing.CreatedBy == incoming.CreatedBy
}

// ============================================================
// Domain and Firestore conversion
// ============================================================

func listImageToDocument(
	img listdom.ListImage,
) imageDocument {
	return imageDocument{
		ID:           img.ID,
		OwnerID:      img.ListID,
		URL:          img.URL,
		DisplayOrder: img.DisplayOrder,
		CreatedAt:    img.CreatedAt,
		CreatedBy:    img.CreatedBy,
		UpdatedAt:    img.UpdatedAt,
		UpdatedBy:    img.UpdatedBy,
	}
}

func encodeListImageDoc(
	img listdom.ListImage,
) map[string]any {
	return encodeImageDocument(
		"list_id",
		listImageToDocument(img),
	)
}

func decodeListImageDoc(
	doc *gfs.DocumentSnapshot,
	fallbackListID string,
) (listdom.ListImage, bool) {
	raw, ok := decodeImageDocument(
		doc,
		fallbackListID,
		"list_id",
	)
	if !ok {
		return listdom.ListImage{}, false
	}

	if raw.URL == "" {
		return listdom.ListImage{}, false
	}

	if raw.CreatedBy == "" {
		return listdom.ListImage{}, false
	}

	displayOrder := raw.DisplayOrder
	if displayOrder < 0 {
		displayOrder = 0
	}

	img, err := listdom.NewListImage(
		raw.ID,
		raw.OwnerID,
		raw.URL,
		displayOrder,
		raw.CreatedAt,
		raw.CreatedBy,
	)
	if err != nil {
		return listdom.ListImage{}, false
	}

	img.UpdatedAt = raw.UpdatedAt
	img.UpdatedBy = raw.UpdatedBy

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, false
	}

	return img, true
}
