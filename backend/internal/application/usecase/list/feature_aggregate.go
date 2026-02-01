// backend/internal/application/usecase/list/feature_aggregate.go
//
// Responsibility:
// - List と ListImage の「集約ビュー」を提供する（read-model寄り）。
// - 依存が未接続の場合の挙動（NotSupported/空配列）を定義する。
//
// Features:
// - GetImages
// - GetAggregate
package list

import (
	"context"

	usecase "narratives/internal/application/usecase"
	listimgdom "narratives/internal/domain/listImage"
)

func (uc *ListUsecase) GetImages(ctx context.Context, listID string) ([]listimgdom.ListImage, error) {
	if uc.imageReader == nil {
		return []listimgdom.ListImage{}, nil
	}
	items, err := uc.imageReader.ListByListID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []listimgdom.ListImage{}, nil
	}
	return items, nil
}

func (uc *ListUsecase) GetAggregate(ctx context.Context, id string) (ListAggregate, error) {
	if uc.listReader == nil {
		return ListAggregate{}, usecase.ErrNotSupported("List.GetAggregate")
	}

	li, err := uc.listReader.GetByID(ctx, id)
	if err != nil {
		return ListAggregate{}, err
	}

	var images []listimgdom.ListImage
	if uc.imageReader != nil {
		items, err := uc.imageReader.ListByListID(ctx, id)
		if err != nil {
			return ListAggregate{}, err
		}
		images = items
	}

	return ListAggregate{List: li, Images: images}, nil
}
