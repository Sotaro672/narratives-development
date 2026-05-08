// backend/internal/application/usecase/list/feature_aggregate.go
//
// Responsibility:
// - List と ListImage の「集約ビュー」を提供する（read-model寄り）。
// - 依存が未接続の場合の挙動（NotSupported/空配列）を定義する。
//
// Firebase Storage migration policy:
// - domain/listImage は削除済み
// - ListImage は domain/list.ListImage を使う
// - backend は GCS signed URL / GCS bucket / GCS object を扱わない
// - ListImage.URL は Firebase Storage downloadURL
// - ListImage.ObjectPath は Firebase Storage objectPath
//
// Features:
// - GetImages
// - GetAggregate
package list

import (
	"context"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

func (uc *ListUsecase) GetImages(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	if uc == nil || uc.imageReader == nil {
		return []listdom.ListImage{}, nil
	}

	items, err := uc.imageReader.ListByListID(ctx, listID)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []listdom.ListImage{}, nil
	}

	return items, nil
}

func (uc *ListUsecase) GetAggregate(
	ctx context.Context,
	id string,
) (ListAggregate, error) {
	if uc == nil || uc.listReader == nil {
		return ListAggregate{}, usecase.ErrNotSupported("List.GetAggregate")
	}

	li, err := uc.listReader.GetByID(ctx, id)
	if err != nil {
		return ListAggregate{}, err
	}

	images := []listdom.ListImage{}

	if uc.imageReader != nil {
		items, err := uc.imageReader.ListByListID(ctx, id)
		if err != nil {
			return ListAggregate{}, err
		}

		if items != nil {
			images = items
		}
	}

	return ListAggregate{
		List:   li,
		Images: images,
	}, nil
}
