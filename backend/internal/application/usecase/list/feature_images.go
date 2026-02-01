// backend/internal/application/usecase/list/feature_images.go
//
// Responsibility:
// - ListImage の保存（GCS object -> domain ListImage）を提供する。
// - 保存時に URL 解決結果をログへ出す（運用上の検証用）。
//
// Features:
// - SaveImageFromGCS
package list

import (
	"context"
	"log"
	"strings"

	usecase "narratives/internal/application/usecase"
	listimgdom "narratives/internal/domain/listImage"
)

func (uc *ListUsecase) SaveImageFromGCS(
	ctx context.Context,
	id string,
	listID string,
	bucket string,
	objectPath string,
	size int64,
	displayOrder int,
) (listimgdom.ListImage, error) {
	if uc.imageObjectSaver == nil {
		return listimgdom.ListImage{}, usecase.ErrNotSupported("List.SaveImageFromGCS")
	}

	img, err := uc.imageObjectSaver.SaveFromBucketObject(
		ctx,
		strings.TrimSpace(id),
		strings.TrimSpace(listID),
		strings.TrimSpace(bucket),
		strings.TrimSpace(objectPath),
		size,
		displayOrder,
	)
	if err != nil {
		return listimgdom.ListImage{}, err
	}

	log.Printf(
		"[list_usecase] listImage URL resolved=%t url=%q listID=%s imageID=%s bucketHint=%s objectPath=%s",
		strings.TrimSpace(img.URL) != "",
		strings.TrimSpace(img.URL),
		strings.TrimSpace(listID),
		strings.TrimSpace(id),
		strings.TrimSpace(bucket),
		strings.TrimSpace(objectPath),
	)

	return img, nil
}
