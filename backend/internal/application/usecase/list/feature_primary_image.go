// backend/internal/application/usecase/list/feature_primary_image.go
//
// Responsibility:
// - List の代表画像（List.ImageID）を更新する。
// - 入力が「URL」か「ListImage.ID」かを判定して URL を解決し、ListPatcher に委譲する。
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

	// 1) URL が直接渡されている場合
	if isImageURL(iid) {
		log.Printf("[list_usecase] listImage URL resolved=%t url=%q listID=%s imageID=%s", true, iid, lid, iid)

		return uc.listPatcher.UpdateImageID(
			ctx,
			lid,
			iid, // URL
			now.UTC(),
			normalizeStrPtr(updatedBy),
		)
	}

	// 2) ListImage.ID とみなして解決 → URL を設定
	if uc.imageByIDReader == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage (imageByIDReader)")
	}

	img, err := uc.imageByIDReader.GetByID(ctx, iid)
	if err != nil {
		return listdom.List{}, err
	}

	if strings.TrimSpace(img.ListID) != lid {
		return listdom.List{}, errors.New("list: image belongs to other list")
	}

	imageURL := strings.TrimSpace(img.URL)
	if imageURL == "" {
		if isImageURL(strings.TrimSpace(img.ID)) {
			imageURL = strings.TrimSpace(img.ID)
		} else if strings.TrimSpace(img.ID) != "" {
			imageURL = listimgdom.PublicURL(listimgdom.DefaultBucket, strings.TrimSpace(img.ID))
		}
	}
	if strings.TrimSpace(imageURL) == "" {
		log.Printf("[list_usecase] listImage URL resolved=%t url=%q listID=%s imageID=%s", false, "", lid, iid)
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	log.Printf("[list_usecase] listImage URL resolved=%t url=%q listID=%s imageID=%s", true, imageURL, lid, iid)

	return uc.listPatcher.UpdateImageID(
		ctx,
		lid,
		imageURL,
		now.UTC(),
		normalizeStrPtr(updatedBy),
	)
}
