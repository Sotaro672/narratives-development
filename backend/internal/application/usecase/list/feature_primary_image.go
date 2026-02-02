// backend/internal/application/usecase/list/feature_primary_image.go
//
// Responsibility:
//   - List の代表画像（List.ImageID）を更新する。
//   - 入力が「URL」か「imageId（FirestoreのimagesサブコレのID）」か
//     「GCS objectPath/URL」かを判定して URL を解決し、ListPatcher に委譲する。
//
// Features:
// - SetPrimaryImage
package list

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// ListImageRecordByIDReader is an optional extended contract for Firestore subcollection.
// If you already have a separate Firestore repo, implement this there.
type ListImageRecordByIDReader interface {
	GetByID(ctx context.Context, listID string, imageID string) (listimgdom.ListImage, error)
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

	lid := strings.TrimSpace(listID)
	iid := strings.TrimSpace(imageID)
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
			normalizeStrPtr(updatedBy),
		)
	}

	// 2) Firestore: /lists/{listId}/images/{imageId} から解決できるなら最優先
	// （今回の推奨方針の source of truth）
	imageURL := ""
	if uc.listImageRecordRepo != nil {
		if r, ok := uc.listImageRecordRepo.(ListImageRecordByIDReader); ok {
			img, err := r.GetByID(ctx, lid, iid)
			if err == nil {
				if strings.TrimSpace(img.ListID) != "" && strings.TrimSpace(img.ListID) != lid {
					return listdom.List{}, errors.New("list: image belongs to other list")
				}
				imageURL = strings.TrimSpace(img.URL)
				if imageURL == "" {
					// Firestore record should carry URL; if empty, treat as invalid
					return listdom.List{}, listdom.ErrInvalidImageID
				}
			}
		}
	}

	// 3) fallback: GCS 側で解決（入力が objectPath/URL 互換のときのみ）
	if imageURL == "" {
		if uc.imageByIDReader == nil {
			return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage (imageByIDReader)")
		}

		img, err := uc.imageByIDReader.GetByID(ctx, iid)
		if err != nil {
			return listdom.List{}, err
		}

		if strings.TrimSpace(img.ListID) != "" && strings.TrimSpace(img.ListID) != lid {
			return listdom.List{}, errors.New("list: image belongs to other list")
		}

		imageURL = strings.TrimSpace(img.URL)
		if imageURL == "" {
			// ✅ fallback を厳格化：img.ID が GCS URL なら採用、objectPath なら PublicURL 生成
			idv := strings.TrimSpace(img.ID)
			if isImageURL(idv) {
				imageURL = idv
			} else if isLikelyObjectPathForList(idv, lid) {
				// idv is expected to be "{listId}/{imageId}/{fileName}"
				imageURL = listimgdom.PublicURL(listimgdom.DefaultBucket, idv)
			}
		}
	}

	if strings.TrimSpace(imageURL) == "" {
		log.Printf("[list_usecase] primaryImage resolved=%t url=%q listID=%s input=%s", false, "", lid, iid)
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	log.Printf("[list_usecase] primaryImage resolved=%t url=%q listID=%s input=%s", true, imageURL, lid, iid)

	return uc.listPatcher.UpdateImageID(
		ctx,
		lid,
		imageURL,
		now.UTC(),
		normalizeStrPtr(updatedBy),
	)
}

// isLikelyObjectPathForList checks whether s looks like "{listId}/{something}/{fileName}"
// and starts with the given listId.
func isLikelyObjectPathForList(s string, listID string) bool {
	s = strings.TrimLeft(strings.TrimSpace(s), "/")
	if s == "" {
		return false
	}
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return false
	}
	return strings.TrimSpace(parts[0]) == strings.TrimSpace(listID)
}
