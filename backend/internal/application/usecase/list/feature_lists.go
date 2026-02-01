// backend/internal/application/usecase/list/feature_lists.go
//
// Responsibility:
// - List の CRUD（読み取り/一覧/作成/更新）を提供する。
// - 作成時の readableId 自動付与と（可能なら）best-effort 永続化を行う。
// - 画像バケット(prefix)初期化を（可能なら）best-effort で実行する。
//
// Features:
// - List / Count / Create / Update / GetByID
package list

import (
	"context"
	"log"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

func (uc *ListUsecase) List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error) {
	if uc.listLister == nil {
		return listdom.PageResult[listdom.List]{}, usecase.ErrNotSupported("List.List")
	}
	return uc.listLister.List(ctx, filter, sort, page)
}

func (uc *ListUsecase) Count(ctx context.Context, filter listdom.Filter) (int, error) {
	if uc.listLister == nil {
		return 0, usecase.ErrNotSupported("List.Count")
	}
	return uc.listLister.Count(ctx, filter)
}

func (uc *ListUsecase) Create(ctx context.Context, item listdom.List) (listdom.List, error) {
	if uc.listCreator == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.Create")
	}

	created, err := uc.listCreator.Create(ctx, item)
	if err != nil {
		return listdom.List{}, err
	}

	// readableId を自動付与（返却値に必ず乗せる）
	if strings.TrimSpace(created.ReadableID) == "" {
		rid := generateReadableID(strings.TrimSpace(created.ID), created.CreatedAt)
		created.ReadableID = rid

		// 永続化（可能なら）: patch update が使える場合のみ best-effort
		if pu := uc.getPatchUpdater(); pu != nil {
			now := time.Now().UTC()
			patch := listdom.ListPatch{
				ReadableID: &rid,
				UpdatedAt:  &now,
			}
			if _, e := pu.Update(ctx, strings.TrimSpace(created.ID), patch); e != nil {
				log.Printf("[list_usecase] readableId persist failed (ignored): listID=%s err=%v", strings.TrimSpace(created.ID), e)
			}
		}
	}

	// listId 名のバケット（または prefix）初期化（実装があれば）
	listID := strings.TrimSpace(created.ID)
	if listID != "" && uc.imageObjectSaver != nil {
		if init, ok := any(uc.imageObjectSaver).(ListImageBucketInitializer); ok {
			_ = init.EnsureListBucket(ctx, listID) // 失敗しても list 作成は成立しているので握りつぶし
		}
	}

	return created, nil
}

func (uc *ListUsecase) Update(ctx context.Context, item listdom.List) (listdom.List, error) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	// 最優先: patch Update(Update(ctx, id, patch)) が叩けるならそれを使う
	patch := buildPatchFromItem(item)

	if uc.listReader != nil {
		if pu, ok := any(uc.listReader).(ListPatchUpdater); ok {
			return pu.Update(ctx, id, patch)
		}
	}
	if uc.listCreator != nil {
		if pu, ok := any(uc.listCreator).(ListPatchUpdater); ok {
			return pu.Update(ctx, id, patch)
		}
	}

	// fallback: Update(ctx, item) が配線されているならそれを使う
	if uc.listUpdater == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.Update")
	}
	return uc.listUpdater.Update(ctx, item)
}

func (uc *ListUsecase) GetByID(ctx context.Context, id string) (listdom.List, error) {
	if uc.listReader == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.GetByID")
	}
	return uc.listReader.GetByID(ctx, id)
}
