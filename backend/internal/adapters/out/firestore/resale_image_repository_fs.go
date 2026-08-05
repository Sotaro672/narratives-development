// backend/internal/adapters/out/firestore/resale_image_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
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

func NewResaleImageRepositoryFS(
	client *gfs.Client,
) *ResaleImageRepositoryFS {
	return &ResaleImageRepositoryFS{
		Client: client,
	}
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
		return nil,
			errors.New("firestore client is nil")
	}

	resaleID = strings.TrimSpace(resaleID)
	if resaleID == "" {
		return []resaledom.ResaleImage{}, nil
	}

	return listOrderedImageDocuments(
		ctx,
		r.conditionImagesCol(resaleID),
		func(
			doc *gfs.DocumentSnapshot,
		) (resaledom.ResaleImage, bool) {
			return decodeResaleImageDoc(
				doc,
				resaleID,
			)
		},
	)
}

// ============================================================
// Resale image write
// ============================================================

func (r *ResaleImageRepositoryFS) Create(
	ctx context.Context,
	img resaledom.ResaleImage,
) (resaledom.ResaleImage, error) {
	if r == nil || r.Client == nil {
		return resaledom.ResaleImage{},
			errors.New("firestore client is nil")
	}

	img.ResaleID = strings.TrimSpace(img.ResaleID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ResaleID == "" {
		return resaledom.ResaleImage{},
			resaledom.ErrInvalidConditionImageResaleID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(img.ID)
	if !ok {
		return resaledom.ResaleImage{},
			resaledom.ErrInvalidConditionImageID
	}

	img.ID = normalizedImageID

	img.CreatedAt =
		normalizeImageCreatedAt(img.CreatedAt)

	img.UpdatedAt =
		normalizeImageUpdatedAt(img.UpdatedAt)

	img.UpdatedBy =
		normalizeImageUpdatedBy(img.UpdatedBy)

	if err := img.Validate(); err != nil {
		return resaledom.ResaleImage{}, err
	}

	ref := r.conditionImagesCol(img.ResaleID).
		Doc(img.ID)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			_, err := tx.Get(ref)
			if err == nil {
				return resaledom.
					ErrConditionImageConflict
			}

			if status.Code(err) != codes.NotFound {
				return err
			}

			if err := tx.Create(
				ref,
				encodeResaleImageDoc(img),
			); err != nil {
				if status.Code(err) ==
					codes.AlreadyExists {
					return resaledom.
						ErrConditionImageConflict
				}

				return err
			}

			return nil
		},
	)
	if err != nil {
		if errors.Is(
			err,
			resaledom.ErrConditionImageConflict,
		) {
			return resaledom.ResaleImage{},
				resaledom.ErrConditionImageConflict
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
		return resaledom.ResaleImage{},
			errors.New("firestore client is nil")
	}

	resaleID = strings.TrimSpace(resaleID)
	if resaleID == "" {
		return resaledom.ResaleImage{},
			resaledom.ErrInvalidConditionImageResaleID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(imageID)
	if !ok {
		return resaledom.ResaleImage{},
			resaledom.ErrInvalidConditionImageID
	}

	ref := r.conditionImagesCol(resaleID).
		Doc(normalizedImageID)

	var updated resaledom.ResaleImage

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
					return resaledom.
						ErrConditionImageNotFound
				}

				return err
			}

			current, ok := decodeResaleImageDoc(
				doc,
				resaleID,
			)
			if !ok {
				return resaledom.
					ErrConditionImageNotFound
			}

			changed := false
			clearUpdatedAt := false
			clearUpdatedBy := false

			if patch.URL != nil {
				value := strings.TrimSpace(
					*patch.URL,
				)

				if err := current.UpdateURL(
					value,
				); err != nil {
					return err
				}

				changed = true
			}

			if patch.DisplayOrder != nil {
				if err := current.SetDisplayOrder(
					*patch.DisplayOrder,
				); err != nil {
					return err
				}

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

			data := encodeResaleImageDoc(current)

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
		if errors.Is(
			err,
			resaledom.ErrConditionImageNotFound,
		) {
			return resaledom.ResaleImage{},
				resaledom.ErrConditionImageNotFound
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
	if resaleID == "" {
		return resaledom.
			ErrInvalidConditionImageResaleID
	}

	normalizedImageID, ok :=
		normalizeImageDocumentID(imageID)
	if !ok {
		return resaledom.
			ErrInvalidConditionImageID
	}

	ref := r.conditionImagesCol(resaleID).
		Doc(normalizedImageID)

	err := deleteImageDocument(
		ctx,
		r.Client,
		ref,
		resaledom.ErrConditionImageNotFound,
	)
	if err != nil {
		if errors.Is(
			err,
			resaledom.ErrConditionImageNotFound,
		) {
			return resaledom.
				ErrConditionImageNotFound
		}

		return err
	}

	return nil
}

// ============================================================
// Domain and Firestore conversion
// ============================================================

func resaleImageToDocument(
	img resaledom.ResaleImage,
) imageDocument {
	return imageDocument{
		ID:           img.ID,
		OwnerID:      img.ResaleID,
		URL:          img.URL,
		DisplayOrder: img.DisplayOrder,
		CreatedAt:    img.CreatedAt,
		CreatedBy:    img.CreatedBy,
		UpdatedAt:    img.UpdatedAt,
		UpdatedBy:    img.UpdatedBy,
	}
}

func encodeResaleImageDoc(
	img resaledom.ResaleImage,
) map[string]any {
	return encodeImageDocument(
		"resale_id",
		resaleImageToDocument(img),
	)
}

func decodeResaleImageDoc(
	doc *gfs.DocumentSnapshot,
	fallbackResaleID string,
) (resaledom.ResaleImage, bool) {
	raw, ok := decodeImageDocument(
		doc,
		fallbackResaleID,
		"resale_id",
	)
	if !ok {
		return resaledom.ResaleImage{}, false
	}

	img, err := resaledom.NewResaleImage(
		raw.ID,
		raw.OwnerID,
		raw.URL,
		raw.DisplayOrder,
		raw.CreatedAt,
		raw.CreatedBy,
	)
	if err != nil {
		return resaledom.ResaleImage{}, false
	}

	img.UpdatedAt = raw.UpdatedAt
	img.UpdatedBy = raw.UpdatedBy

	if err := img.Validate(); err != nil {
		return resaledom.ResaleImage{}, false
	}

	return img, true
}
