// backend/internal/application/usecase/list_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// ListReader は List 単体取得の契約です。
type ListReader interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// ListLister は List 一覧取得の契約です（GET /lists 用）。
type ListLister interface {
	List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error)
	Count(ctx context.Context, filter listdom.Filter) (int, error)
}

// ListCreator は List 作成の契約です。
type ListCreator interface {
	// Create は list を永続化し、保存結果（ID採番等を含む）を返します。
	Create(ctx context.Context, item listdom.List) (listdom.List, error)
}

// ListUpdater は List 本体更新の契約です（PUT/PATCH /lists/{id} 用）。
type ListUpdater interface {
	Update(ctx context.Context, item listdom.List) (listdom.List, error)
}

// ★ domain.Repository 互換の「patch Update」(Update(ctx, id, patch)) を直接叩ける場合に使う。
type ListPatchUpdater interface {
	Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
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
	List   listdom.List           `json:"list"`
	Images []listimgdom.ListImage `json:"images"`
}

// ListUsecase は List と ListImage をまとめて扱います。
type ListUsecase struct {
	listReader       ListReader
	listLister       ListLister  // GET /lists
	listCreator      ListCreator // POST /lists (optional)
	listUpdater      ListUpdater // PUT/PATCH /lists/{id} (optional)
	listPatcher      ListPatcher
	imageReader      ListImageReader
	imageByIDReader  ListImageByIDReader
	imageObjectSaver ListImageObjectSaver
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
	uc := &ListUsecase{
		listReader:       listReader,
		listLister:       nil, // auto-wire below
		listCreator:      nil,
		listUpdater:      nil, // auto-wire below
		listPatcher:      listPatcher,
		imageReader:      imageReader,
		imageByIDReader:  imageByIDReader,
		imageObjectSaver: imageObjectSaver,
	}

	// 既存DIを壊さずに、listReader(実体はrepo)が ListLister/ListUpdater を実装していれば自動で配線
	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}
		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}

	return uc
}

// 作成にも対応したコンストラクタ（既存呼び出しを壊さない）
func NewListUsecaseWithCreator(
	listReader ListReader,
	listCreator ListCreator,
	listPatcher ListPatcher,
	imageReader ListImageReader,
	imageByIDReader ListImageByIDReader,
	imageObjectSaver ListImageObjectSaver,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:       listReader,
		listLister:       nil, // auto-wire below
		listCreator:      listCreator,
		listUpdater:      nil, // auto-wire below
		listPatcher:      listPatcher,
		imageReader:      imageReader,
		imageByIDReader:  imageByIDReader,
		imageObjectSaver: imageObjectSaver,
	}

	// listReader が ListLister/ListUpdater を実装していれば優先
	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}
		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}
	// 念のため: listCreator(同じrepoを渡しているケース)が実装していれば配線
	if uc.listLister == nil && listCreator != nil {
		if lister, ok := any(listCreator).(ListLister); ok {
			uc.listLister = lister
		}
	}
	if uc.listUpdater == nil && listCreator != nil {
		if updater, ok := any(listCreator).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}

	return uc
}

// List は List 一覧を返します（GET /lists）
func (uc *ListUsecase) List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error) {
	log.Printf("[list_usecase] List called filter=%s sort=%s page=%s",
		dumpAsJSON(filter),
		dumpAsJSON(sort),
		dumpAsJSON(page),
	)

	if uc.listLister == nil {
		log.Printf("[list_usecase] List NOT supported (listLister is nil)")
		return listdom.PageResult[listdom.List]{}, ErrNotSupported("List.List")
	}

	out, err := uc.listLister.List(ctx, filter, sort, page)
	if err != nil {
		log.Printf("[list_usecase] List failed err=%v", err)
		return listdom.PageResult[listdom.List]{}, err
	}

	log.Printf("[list_usecase] List ok count=%d page=%d perPage=%d totalPages=%d",
		len(out.Items),
		out.Page,
		out.PerPage,
		out.TotalPages,
	)

	return out, nil
}

// Count も必要なら使えるようにしておく（任意）
func (uc *ListUsecase) Count(ctx context.Context, filter listdom.Filter) (int, error) {
	if uc.listLister == nil {
		return 0, ErrNotSupported("List.Count")
	}
	return uc.listLister.Count(ctx, filter)
}

// Create は List を作成します。
func (uc *ListUsecase) Create(ctx context.Context, item listdom.List) (listdom.List, error) {
	log.Printf("[list_usecase] Create called item=%s", dumpAsJSON(item))

	if uc.listCreator == nil {
		log.Printf("[list_usecase] Create NOT supported (listCreator is nil)")
		return listdom.List{}, ErrNotSupported("List.Create")
	}

	created, err := uc.listCreator.Create(ctx, item)
	if err != nil {
		log.Printf("[list_usecase] Create failed err=%v item=%s", err, dumpAsJSON(item))
		return listdom.List{}, err
	}

	log.Printf("[list_usecase] Create ok created=%s", dumpAsJSON(created))
	return created, nil
}

// Update は List 本体を更新します（タイトル/説明/価格/ステータス等）。
func (uc *ListUsecase) Update(ctx context.Context, item listdom.List) (listdom.List, error) {
	log.Printf("[list_usecase] Update called item=%s", dumpAsJSON(item))

	id := strings.TrimSpace(item.ID)
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	// ✅ 最優先: domain.Repository 互換の patch Update(Update(ctx, id, patch)) が叩けるならそれを使う
	patch := buildPatchFromItem(item)

	if uc.listReader != nil {
		if pu, ok := any(uc.listReader).(ListPatchUpdater); ok {
			updated, err := pu.Update(ctx, id, patch)
			if err != nil {
				log.Printf("[list_usecase] Update failed (patch via listReader) err=%v item=%s", err, dumpAsJSON(item))
				return listdom.List{}, err
			}
			log.Printf("[list_usecase] Update ok (patch via listReader) updated=%s", dumpAsJSON(updated))
			return updated, nil
		}
	}
	if uc.listCreator != nil {
		if pu, ok := any(uc.listCreator).(ListPatchUpdater); ok {
			updated, err := pu.Update(ctx, id, patch)
			if err != nil {
				log.Printf("[list_usecase] Update failed (patch via listCreator) err=%v item=%s", err, dumpAsJSON(item))
				return listdom.List{}, err
			}
			log.Printf("[list_usecase] Update ok (patch via listCreator) updated=%s", dumpAsJSON(updated))
			return updated, nil
		}
	}

	// fallback: Update(ctx, item) が配線されているならそれを使う
	if uc.listUpdater == nil {
		log.Printf("[list_usecase] Update NOT supported (listUpdater is nil)")
		return listdom.List{}, ErrNotSupported("List.Update")
	}

	updated, err := uc.listUpdater.Update(ctx, item)
	if err != nil {
		log.Printf("[list_usecase] Update failed err=%v item=%s", err, dumpAsJSON(item))
		return listdom.List{}, err
	}

	log.Printf("[list_usecase] Update ok updated=%s", dumpAsJSON(updated))
	return updated, nil
}

// GetByID は List を返します。
func (uc *ListUsecase) GetByID(ctx context.Context, id string) (listdom.List, error) {
	if uc.listReader == nil {
		return listdom.List{}, ErrNotSupported("List.GetByID")
	}
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
	if uc.listReader == nil {
		return ListAggregate{}, ErrNotSupported("List.GetAggregate")
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

// SaveImageFromGCS は GCS の bucket/objectPath から ListImage を保存します。
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
	log.Printf("[list_usecase] SaveImageFromGCS called listID=%s imageID=%s bucket=%s objectPath=%s size=%d displayOrder=%d createdBy=%s createdAt=%s",
		strings.TrimSpace(listID),
		strings.TrimSpace(id),
		strings.TrimSpace(bucket),
		strings.TrimSpace(objectPath),
		size,
		displayOrder,
		strings.TrimSpace(createdBy),
		createdAt.UTC().Format(time.RFC3339),
	)

	if uc.imageObjectSaver == nil {
		log.Printf("[list_usecase] SaveImageFromGCS NOT supported (imageObjectSaver is nil)")
		return listimgdom.ListImage{}, ErrNotSupported("List.SaveImageFromGCS")
	}

	return uc.imageObjectSaver.SaveFromBucketObject(
		ctx,
		strings.TrimSpace(id),
		strings.TrimSpace(listID),
		strings.TrimSpace(bucket),
		strings.TrimSpace(objectPath),
		size,
		displayOrder,
		strings.TrimSpace(createdBy),
		createdAt.UTC(),
	)
}

// SetPrimaryImage は指定の ListImage を List の代表画像に設定します。
func (uc *ListUsecase) SetPrimaryImage(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	log.Printf("[list_usecase] SetPrimaryImage called listID=%s imageID=%s now=%s updatedBy=%v",
		strings.TrimSpace(listID),
		strings.TrimSpace(imageID),
		now.UTC().Format(time.RFC3339),
		normalizeStrPtr(updatedBy),
	)

	if uc.listPatcher == nil {
		log.Printf("[list_usecase] SetPrimaryImage NOT supported (listPatcher is nil)")
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage")
	}

	// 画像の所属整合性チェック（可能なら）
	if uc.imageByIDReader != nil {
		img, err := uc.imageByIDReader.GetByID(ctx, imageID)
		if err != nil {
			return listdom.List{}, err
		}
		if strings.TrimSpace(img.ListID) != strings.TrimSpace(listID) {
			return listdom.List{}, listdom.ErrImageBelongsToOtherList
		}
	}

	return uc.listPatcher.UpdateImageID(
		ctx,
		strings.TrimSpace(listID),
		strings.TrimSpace(imageID),
		now.UTC(),
		normalizeStrPtr(updatedBy),
	)
}

// ==============================
// helpers
// ==============================

func buildPatchFromItem(item listdom.List) listdom.ListPatch {
	// PUT 相当: 主要フィールドは常に送られてくる前提で patch を埋める
	statusV := item.Status
	assigneeV := strings.TrimSpace(item.AssigneeID)
	imageV := strings.TrimSpace(item.ImageID)
	titleV := strings.TrimSpace(item.Title)
	descV := strings.TrimSpace(item.Description)

	var updatedByV *string
	if item.UpdatedBy != nil {
		v := strings.TrimSpace(*item.UpdatedBy)
		if v != "" {
			updatedByV = &v
		}
	}

	now := time.Now().UTC()
	updatedAtV := now
	if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
		updatedAtV = item.UpdatedAt.UTC()
	}

	// ✅ prices: nil(未指定)なら patch に入れない（意図せず全削除を防ぐ）
	var pricesPtr *[]listdom.ListPriceRow
	if item.Prices != nil {
		pv := item.Prices
		pricesPtr = &pv
	}

	return listdom.ListPatch{
		Status:      &statusV,
		AssigneeID:  &assigneeV,
		ImageID:     &imageV,
		Title:       &titleV,
		Description: &descV,
		UpdatedBy:   updatedByV,
		UpdatedAt:   &updatedAtV,
		Prices:      pricesPtr,
	}
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}

func dumpAsJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<json_marshal_failed>"
	}
	return string(b)
}
