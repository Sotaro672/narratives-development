// backend/internal/application/usecase/list/feature_primary_image.go
//
// Responsibility:
//   - List の代表画像（List.ImageID）を更新する。
//   - 入力が「URL」か「imageId（FirestoreのimagesサブコレのDocID）」かを判定して URL を解決し、ListPatcher に委譲する。
//   - GCS bucket は env 固定（usecase 側で DefaultBucket を使って URL を捏造しない）。
//
// Features:
// - SetPrimaryImage
package list

import (
	"context"
	"errors"
	"log"
	"time"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// ListImageRecordByIDReader is an optional extended contract for Firestore subcollection.
// NOTE:
// - Record store is expected to be /lists/{listId}/images/{imageId} (docID = imageId).
// - Keep signature aligned with adapters/out/firestore/list_image_repository_fs.go:GetByID(ctx, id string).
type ListImageRecordByIDReader interface {
	GetByID(ctx context.Context, imageID string) (listimgdom.ListImage, error)
}

func (uc *ListUsecase) SetPrimaryImage(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	if uc.listPatcher == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage")
	}

	// ✅ Trim/normalize をしない（渡された値をそのまま扱う）
	lid := listID
	iid := imageID

	if lid == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}
	if iid == "" {
		return listdom.List{}, listdom.ErrEmptyImageID
	}

	// 1) URL が直接渡されている場合（primary URL cache としてそのまま使う）
	if isImageURL(iid) {
		log.Printf("[list_usecase] primaryImage resolved=%t url=%q listID=%s input=%s", true, iid, lid, iid)

		return uc.listPatcher.UpdateImageID(
			ctx,
			lid,
			iid, // URL
			now.UTC(),
			updatedBy, // ✅ normalizeStrPtr を使わない
		)
	}

	// 2) Firestore: /lists/{listId}/images/{imageId} から解決できるなら最優先
	// （今回の推奨方針の source of truth）
	imageURL := ""
	if uc.listImageRecordRepo != nil {
		if r, ok := uc.listImageRecordRepo.(ListImageRecordByIDReader); ok {
			img, err := r.GetByID(ctx, iid) // ✅ imageId (docID)
			if err == nil {
				// Safety: image must belong to the same list
				// ✅ TrimSpace をしない（値をそのまま比較）
				if img.ListID != "" && img.ListID != lid {
					return listdom.List{}, errors.New("list: image belongs to other list")
				}

				imageURL = img.URL
				if imageURL == "" {
					// Firestore record should carry URL; if empty, treat as invalid
					return listdom.List{}, listdom.ErrInvalidImageID
				}
			}
		}
	}

	// 3) fallback: GCS 側で解決（入力が objectPath/URL 互換のときのみ）
	// NOTE:
	// - This path is "legacy/fallback". With canonical record store, (2) should handle primary resolution.
	// - Do NOT generate public URL using DefaultBucket here (bucket is env-fixed and should be handled by adapter).
	if imageURL == "" {
		if uc.imageByIDReader == nil {
			return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage (imageByIDReader)")
		}

		img, err := uc.imageByIDReader.GetByID(ctx, iid)
		if err != nil {
			return listdom.List{}, err
		}

		// ✅ TrimSpace をしない（値をそのまま比較）
		if img.ListID != "" && img.ListID != lid {
			return listdom.List{}, errors.New("list: image belongs to other list")
		}

		imageURL = img.URL
		if imageURL == "" {
			// ✅ strict: if adapter didn't resolve URL, treat as invalid.
			// (Do not fabricate URL with DefaultBucket.)
			return listdom.List{}, listdom.ErrInvalidImageID
		}
	}

	// ✅ TrimSpace をしない
	if imageURL == "" {
		log.Printf("[list_usecase] primaryImage resolved=%t url=%q listID=%s input=%s", false, "", lid, iid)
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	log.Printf("[list_usecase] primaryImage resolved=%t url=%q listID=%s input=%s", true, imageURL, lid, iid)

	return uc.listPatcher.UpdateImageID(
		ctx,
		lid,
		imageURL,
		now.UTC(),
		updatedBy, // ✅ normalizeStrPtr を使わない
	)
}
