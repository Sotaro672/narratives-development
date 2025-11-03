package usecase

import (
    "context"
    "time"

    listdom "narratives/internal/domain/list"
    listimgdom "narratives/internal/domain/listImage"
)

// ListReader は List 単体取得の契約です。
type ListReader interface {
    GetByID(ctx context.Context, id string) (listdom.List, error)
}

// ListPatcher は List.ImageID を更新できる契約です。
type ListPatcher interface {
    // List.ImageID を imageID に更新し、更新済み List を返します。
    UpdateImageID(ctx context.Context, listID string, imageID string, now time.Time, updatedBy *string) (listdom.List, error)
}

// ListImageReader は ListID に紐づく ListImage 一覧の取得契約です。
type ListImageReader interface {
    ListByListID(ctx context.Context, listID string) ([]listimgdom.ListImage, error)
}

// ListImageByIDReader は ListImage を主キーで取得する契約です。
type ListImageByIDReader interface {
    GetByID(ctx context.Context, id string) (listimgdom.ListImage, error)
}

// ListImageObjectSaver は GCS に存在するオブジェクトから ListImage を保存する契約です。
// bucket が空の場合は実装側で listimgdom.DefaultBucket を使用してください。
type ListImageObjectSaver interface {
    SaveFromBucketObject(
        ctx context.Context,
        id string,
        listID string,
        bucket string,
        objectPath string,
        size int64,
        displayOrder int,
        createdBy string,
        createdAt time.Time,
    ) (listimgdom.ListImage, error)
}

// ListAggregate は List とその画像一覧のビューです。
type ListAggregate struct {
    List   listdom.List          `json:"list"`
    Images []listimgdom.ListImage`json:"images"`
}

// ListUsecase は List と ListImage をまとめて扱います。
type ListUsecase struct {
    listReader        ListReader
    listPatcher       ListPatcher
    imageReader       ListImageReader
    imageByIDReader   ListImageByIDReader
    imageObjectSaver  ListImageObjectSaver
}

// NewListUsecase はユースケースを初期化します。
// いずれの依存も nil 可（未接続機能は ErrNotSupported で返却）。
func NewListUsecase(
    listReader ListReader,
    listPatcher ListPatcher,
    imageReader ListImageReader,
    imageByIDReader ListImageByIDReader,
    imageObjectSaver ListImageObjectSaver,
) *ListUsecase {
    return &ListUsecase{
        listReader:       listReader,
        listPatcher:      listPatcher,
        imageReader:      imageReader,
        imageByIDReader:  imageByIDReader,
        imageObjectSaver: imageObjectSaver,
    }
}

// GetByID は List を返します。
func (uc *ListUsecase) GetByID(ctx context.Context, id string) (listdom.List, error) {
    return uc.listReader.GetByID(ctx, id)
}

// GetImages は ListID に紐づく画像一覧を返します（未接続時は空配列）。
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

// GetAggregate は List と画像一覧をまとめて返します。
func (uc *ListUsecase) GetAggregate(ctx context.Context, id string) (ListAggregate, error) {
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

// SaveImageFromGCS は GCS の bucket/objectPath から公開URLを構築し、ListImage を保存します。
// bucket が空なら実装側で listimgdom.DefaultBucket を使用してください。
func (uc *ListUsecase) SaveImageFromGCS(
    ctx context.Context,
    id string,
    listID string,
    bucket string,
    objectPath string,
    size int64,
    displayOrder int,
    createdBy string,
    createdAt time.Time,
) (listimgdom.ListImage, error) {
    if uc.imageObjectSaver == nil {
        return listimgdom.ListImage{}, ErrNotSupported("List.SaveImageFromGCS")
    }
    return uc.imageObjectSaver.SaveFromBucketObject(ctx, id, listID, bucket, objectPath, size, displayOrder, createdBy, createdAt)
}

// SetPrimaryImage は指定の ListImage を List の代表画像に設定します。
func (uc *ListUsecase) SetPrimaryImage(
    ctx context.Context,
    listID string,
    imageID string,
    now time.Time,
    updatedBy *string,
) (listdom.List, error) {
    if uc.listPatcher == nil {
        return listdom.List{}, ErrNotSupported("List.SetPrimaryImage")
    }
    // 画像の所属整合性チェック（可能なら）
    if uc.imageByIDReader != nil {
        img, err := uc.imageByIDReader.GetByID(ctx, imageID)
        if err != nil {
            return listdom.List{}, err
        }
        // list.List.SetPrimaryImage と同等のチェック（ListID 一致）
        if stringsTrim(img.ListID) != stringsTrim(listID) {
            return listdom.List{}, listdom.ErrImageBelongsToOtherList
        }
    }
    return uc.listPatcher.UpdateImageID(ctx, listID, imageID, now.UTC(), normalizeStrPtr(updatedBy))
}

// 内部ヘルパー
func stringsTrim(s string) string {
    return trimSpace(s)
}
func trimSpace(s string) string {
    return string([]byte(s))[:len(s)]
}
func normalizeStrPtr(p *string) *string {
    if p == nil {
        return nil
    }
    s := *p
    if t := stringsTrim(s); t == "" {
        return nil
    }
    return &s
}