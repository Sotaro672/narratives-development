// backend/internal/adapters/out/gcs/listImage_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	gcscommon "narratives/internal/adapters/out/gcs/common"
	listimagedom "narratives/internal/domain/listImage"
)

// GCS-based implementation for ListImageRepository.
// 元の ListImageRepositoryPG (PostgreSQL) を、GCS 上のオブジェクトメタ情報を前提とした実装に置き換えたもの。
// - 画像本体は GCS に保存されている前提
// - メタ情報は GCS ObjectAttrs / Metadata から構成
// - ListImage.ID は GCS の object 名（パス）として扱う
type ListImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// デフォルトバケット（必要に応じて環境に合わせて変更）
const defaultListImageBucket = "narratives_development_list_image"

// コンストラクタ
func NewListImageRepositoryGCS(client *storage.Client, bucket string) *ListImageRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultListImageBucket
	}
	return &ListImageRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *ListImageRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultListImageBucket
	}
	return b
}

// ─────────────────────────────────
// Queries
// ─────────────────────────────────

// GetByID satisfies usecase.ListImageByIDReader.
// ID は GCS object 名として扱う。
func (r *ListImageRepositoryGCS) GetByID(ctx context.Context, imageID string) (listimagedom.ListImage, error) {
	if r.Client == nil {
		return listimagedom.ListImage{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}
	id := strings.TrimSpace(imageID)
	if id == "" {
		return listimagedom.ListImage{}, listimagedom.ErrNotFound
	}

	bucket := r.bucket()
	attrs, err := r.Client.Bucket(bucket).Object(id).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}

	return buildListImageFromAttrs(bucket, attrs), nil
}

// ListByListID satisfies usecase.ListImageReader.
// listID に紐づく GCS オブジェクト（パス prefix または metadata）を列挙。
func (r *ListImageRepositoryGCS) ListByListID(ctx context.Context, listID string) ([]listimagedom.ListImage, error) {
	if r.Client == nil {
		return nil, errors.New("ListImageRepositoryGCS: nil storage client")
	}
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return nil, nil
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var out []listimagedom.ListImage
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		img := buildListImageFromAttrs(bucket, attrs)
		if img.ListID == listID {
			out = append(out, img)
		}
	}

	// display_order ASC, created_at ASC, id ASC に揃える
	applyListImageSortForListID(out)

	return out, nil
}

func (r *ListImageRepositoryGCS) Exists(ctx context.Context, imageID string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("ListImageRepositoryGCS: nil storage client")
	}
	id := strings.TrimSpace(imageID)
	if id == "" {
		return false, nil
	}
	bucket := r.bucket()
	_, err := r.Client.Bucket(bucket).Object(id).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ListImageRepositoryGCS) Count(ctx context.Context, filter listimagedom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("ListImageRepositoryGCS: nil storage client")
	}
	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	total := 0
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		img := buildListImageFromAttrs(bucket, attrs)
		if matchListImageFilter(img, filter) {
			total++
		}
	}
	return total, nil
}

// List:
// - GCS オブジェクトを全走査して Filter/Sort/Page をメモリ上で適用
func (r *ListImageRepositoryGCS) List(
	ctx context.Context,
	filter listimagedom.Filter,
	sort listimagedom.Sort,
	page listimagedom.Page,
) (listimagedom.PageResult[listimagedom.ListImage], error) {
	if r.Client == nil {
		return listimagedom.PageResult[listimagedom.ListImage]{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []listimagedom.ListImage
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return listimagedom.PageResult[listimagedom.ListImage]{}, err
		}
		img := buildListImageFromAttrs(bucket, attrs)
		if matchListImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	applyListImageSort(all, sort)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return listimagedom.PageResult[listimagedom.ListImage]{
			Items:      []listimagedom.ListImage{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return listimagedom.PageResult[listimagedom.ListImage]{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCursor:
// - ID 昇順を前提に CursorPage.After 以降を返す。
func (r *ListImageRepositoryGCS) ListByCursor(
	ctx context.Context,
	filter listimagedom.Filter,
	_ listimagedom.Sort,
	cpage listimagedom.CursorPage,
) (listimagedom.CursorPageResult[listimagedom.ListImage], error) {
	if r.Client == nil {
		return listimagedom.CursorPageResult[listimagedom.ListImage]{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []listimagedom.ListImage
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return listimagedom.CursorPageResult[listimagedom.ListImage]{}, err
		}
		img := buildListImageFromAttrs(bucket, attrs)
		if matchListImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	// ID 昇順
	applyListImageSortByIDAsc(all)

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	after := strings.TrimSpace(cpage.After)

	start := 0
	if after != "" {
		for i, img := range all {
			if img.ID > after {
				start = i
				break
			}
		}
	}

	if start >= len(all) {
		return listimagedom.CursorPageResult[listimagedom.ListImage]{
			Items:      []listimagedom.ListImage{},
			NextCursor: nil,
			Limit:      limit,
		}, nil
	}

	end := start + limit
	if end > len(all) {
		end = len(all)
	}

	items := all[start:end]

	var next *string
	if end < len(all) {
		n := items[len(items)-1].ID
		next = &n
	}

	return listimagedom.CursorPageResult[listimagedom.ListImage]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ─────────────────────────────────
// Mutations
// ─────────────────────────────────

// SaveFromBucketObject satisfies usecase.ListImageObjectSaver in a GCS world.
// - 実際のアップロードは別レイヤーで済んでいる前提で、ここでは既存オブジェクトのメタ情報を更新して ListImage を返す。
// - bucket が空ならデフォルトバケット使用
// - objectPath を ID として採用し、公開 URL を構築
func (r *ListImageRepositoryGCS) SaveFromBucketObject(
	ctx context.Context,
	id string,
	listID string,
	bucket string,
	objectPath string,
	size int64,
	displayOrder int,
	createdBy string,
	createdAt time.Time,
) (listimagedom.ListImage, error) {
	if r.Client == nil {
		return listimagedom.ListImage{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.bucket()
	}
	objName := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objName == "" {
		return listimagedom.ListImage{}, fmt.Errorf("listImage: empty objectPath")
	}

	obj := r.Client.Bucket(b).Object(objName)

	// オブジェクトが存在する前提
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if list := strings.TrimSpace(listID); list != "" {
		meta["list_id"] = list
	}
	if displayOrder > 0 {
		meta["display_order"] = fmt.Sprint(displayOrder)
	}
	if size > 0 {
		meta["size"] = fmt.Sprint(size)
	}
	if cb := strings.TrimSpace(createdBy); cb != "" {
		meta["created_by"] = cb
	}
	if !createdAt.IsZero() {
		meta["created_at"] = createdAt.UTC().Format(time.RFC3339Nano)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	img := buildListImageFromAttrs(b, newAttrs)

	// 明示 ID が指定されていれば上書き（通常は objName=ID）
	if strings.TrimSpace(id) != "" {
		img.ID = strings.TrimSpace(id)
	}

	return img, nil
}

// Create:
// - GCSでは実オブジェクト作成は別レイヤーなので、Save と同様にメタ更新扱い。
func (r *ListImageRepositoryGCS) Create(ctx context.Context, img listimagedom.ListImage) (listimagedom.ListImage, error) {
	return r.Save(ctx, img, nil)
}

// Update:
// - ID で対象オブジェクトを取得し、patch を metadata に反映して保存。
func (r *ListImageRepositoryGCS) Update(ctx context.Context, imageID string, patch listimagedom.ListImagePatch) (listimagedom.ListImage, error) {
	if r.Client == nil {
		return listimagedom.ListImage{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}

	id := strings.TrimSpace(imageID)
	if id == "" {
		return listimagedom.ListImage{}, listimagedom.ErrNotFound
	}

	bucket := r.bucket()
	obj := r.Client.Bucket(bucket).Object(id)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if patch.URL != nil {
		meta["url"] = strings.TrimSpace(*patch.URL)
	}
	if patch.FileName != nil {
		meta["file_name"] = strings.TrimSpace(*patch.FileName)
	}
	if patch.Size != nil {
		meta["size"] = fmt.Sprint(*patch.Size)
	}
	if patch.DisplayOrder != nil {
		meta["display_order"] = fmt.Sprint(*patch.DisplayOrder)
	}
	if patch.UpdatedBy != nil {
		meta["updated_by"] = strings.TrimSpace(*patch.UpdatedBy)
	}
	if patch.DeletedBy != nil {
		meta["deleted_by"] = strings.TrimSpace(*patch.DeletedBy)
	}
	if patch.DeletedAt != nil {
		// Save削除時刻（nilの場合は「変更なし」扱いとし、明示クリアは別仕様で必要なら拡張）
		t := patch.DeletedAt.UTC()
		meta["deleted_at"] = t.Format(time.RFC3339Nano)
	}

	// updated_at
	if patch.UpdatedAt != nil {
		if !patch.UpdatedAt.IsZero() {
			meta["updated_at"] = patch.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
	} else if len(meta) > 0 {
		meta["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	return buildListImageFromAttrs(bucket, newAttrs), nil
}

// Save:
// - ID or URL から対象オブジェクトを解決し、指定内容を metadata として反映。
func (r *ListImageRepositoryGCS) Save(
	ctx context.Context,
	img listimagedom.ListImage,
	_ *listimagedom.SaveOptions,
) (listimagedom.ListImage, error) {
	if r.Client == nil {
		return listimagedom.ListImage{}, errors.New("ListImageRepositoryGCS: nil storage client")
	}

	defaultBucket := r.bucket()
	objName, bucket, err := r.resolveObjectFromImage(img, defaultBucket)
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	obj := r.Client.Bucket(bucket).Object(objName)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if strings.TrimSpace(img.ListID) != "" {
		meta["list_id"] = strings.TrimSpace(img.ListID)
	}
	if strings.TrimSpace(img.FileName) != "" {
		meta["file_name"] = strings.TrimSpace(img.FileName)
	}
	if img.Size > 0 {
		meta["size"] = fmt.Sprint(img.Size)
	}
	if img.DisplayOrder > 0 {
		meta["display_order"] = fmt.Sprint(img.DisplayOrder)
	}
	if strings.TrimSpace(img.CreatedBy) != "" {
		meta["created_by"] = strings.TrimSpace(img.CreatedBy)
	}
	if img.UpdatedBy != nil && strings.TrimSpace(*img.UpdatedBy) != "" {
		meta["updated_by"] = strings.TrimSpace(*img.UpdatedBy)
	}
	if img.DeletedBy != nil && strings.TrimSpace(*img.DeletedBy) != "" {
		meta["deleted_by"] = strings.TrimSpace(*img.DeletedBy)
	}
	if img.DeletedAt != nil {
		meta["deleted_at"] = img.DeletedAt.UTC().Format(time.RFC3339Nano)
	}
	if !img.CreatedAt.IsZero() {
		meta["created_at"] = img.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if img.UpdatedAt != nil {
		meta["updated_at"] = img.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	out := buildListImageFromAttrs(bucket, newAttrs)

	// img.URL が明示されていれば、それも優先して上書き（メタとは独立した表示用）
	if strings.TrimSpace(img.URL) != "" {
		out.URL = strings.TrimSpace(img.URL)
	}

	// img.ID が明示されていれば上書き
	if strings.TrimSpace(img.ID) != "" {
		out.ID = strings.TrimSpace(img.ID)
	}

	return out, nil
}

// Upload はこのレイヤーでは実ファイルを扱わないため未実装。
func (r *ListImageRepositoryGCS) Upload(ctx context.Context, _ listimagedom.UploadImageInput) (*listimagedom.ListImage, error) {
	_ = ctx
	return nil, listimagedom.ErrUploadFailed
}

func (r *ListImageRepositoryGCS) Delete(ctx context.Context, imageID string) error {
	if r.Client == nil {
		return errors.New("ListImageRepositoryGCS: nil storage client")
	}
	id := strings.TrimSpace(imageID)
	if id == "" {
		return listimagedom.ErrNotFound
	}
	bucket := r.bucket()
	err := r.Client.Bucket(bucket).Object(id).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return listimagedom.ErrNotFound
	}
	return err
}

// ─────────────────────────────────
// Helpers
// ─────────────────────────────────

// GCS ObjectAttrs -> ListImage 変換
func buildListImageFromAttrs(bucket string, attrs *storage.ObjectAttrs) listimagedom.ListImage {
	name := strings.TrimSpace(attrs.Name)

	// ListID の決定:
	// 1. metadata["list_id"]
	// 2. "listID/xxx" のパスプレフィックス
	var listID string
	if v, ok := attrs.Metadata["list_id"]; ok && strings.TrimSpace(v) != "" {
		listID = strings.TrimSpace(v)
	} else if name != "" {
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			listID = strings.TrimSpace(parts[0])
		}
	}

	// 公開URL（共通ユーティリティ利用）
	publicURL := gcscommon.GCSPublicURL(bucket, name, defaultListImageBucket)

	// FileName: パスの末尾
	fileName := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 && idx < len(name)-1 {
		fileName = name[idx+1:]
	}

	// Size: metadata["size"] or attrs.Size
	var size int64
	if sz, ok := gcscommon.ParseInt64Meta(attrs.Metadata, "size"); ok {
		size = sz
	} else if attrs.Size > 0 {
		size = attrs.Size
	}

	// DisplayOrder: metadata["display_order"]
	var displayOrder int
	if v, ok := gcscommon.ParseIntMeta(attrs.Metadata, "display_order"); ok {
		displayOrder = v
	}

	// CreatedAt / UpdatedAt: GCSのCreated/Updatedフィールドを利用
	createdAt := attrs.Created
	if createdAt.IsZero() {
		createdAt = attrs.Updated
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	createdAt = createdAt.UTC()

	var updatedAtPtr *time.Time
	if !attrs.Updated.IsZero() {
		u := attrs.Updated.UTC()
		updatedAtPtr = &u
	}

	// audit fields from metadata
	createdBy := strings.TrimSpace(attrs.Metadata["created_by"])
	if createdBy == "" {
		createdBy = "system"
	}

	var updatedByPtr *string
	if v := strings.TrimSpace(attrs.Metadata["updated_by"]); v != "" {
		updatedByPtr = &v
	}
	var deletedAtPtr *time.Time
	if v := strings.TrimSpace(attrs.Metadata["deleted_at"]); v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			tu := t.UTC()
			deletedAtPtr = &tu
		}
	}
	var deletedByPtr *string
	if v := strings.TrimSpace(attrs.Metadata["deleted_by"]); v != "" {
		deletedByPtr = &v
	}

	return listimagedom.ListImage{
		ID:           name,
		ListID:       listID,
		URL:          publicURL,
		FileName:     fileName,
		Size:         size,
		DisplayOrder: displayOrder,
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		UpdatedAt:    updatedAtPtr,
		UpdatedBy:    updatedByPtr,
		DeletedAt:    deletedAtPtr,
		DeletedBy:    deletedByPtr,
	}
}

// Filter: GCSベース ListImage に対して、元の Filter と同様の条件を適用
func matchListImageFilter(img listimagedom.ListImage, f listimagedom.Filter) bool {
	// SearchQuery: file_name / url に対する部分一致
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		if !strings.Contains(strings.ToLower(img.FileName), lq) &&
			!strings.Contains(strings.ToLower(img.URL), lq) {
			return false
		}
	}

	// IDs
	if len(f.IDs) > 0 {
		ok := false
		for _, id := range f.IDs {
			if strings.TrimSpace(id) == img.ID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// ListID
	if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
		if img.ListID != strings.TrimSpace(*f.ListID) {
			return false
		}
	}
	if len(f.ListIDs) > 0 {
		ok := false
		for _, lid := range f.ListIDs {
			if img.ListID == strings.TrimSpace(lid) && img.ListID != "" {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// FileNameLike
	if f.FileNameLike != nil && strings.TrimSpace(*f.FileNameLike) != "" {
		like := strings.ToLower(strings.TrimSpace(*f.FileNameLike))
		if !strings.Contains(strings.ToLower(img.FileName), like) {
			return false
		}
	}

	// size range
	if f.MinSize != nil && img.Size < *f.MinSize {
		return false
	}
	if f.MaxSize != nil && img.Size > *f.MaxSize {
		return false
	}

	// display_order range
	if f.MinDisplayOrd != nil && img.DisplayOrder < *f.MinDisplayOrd {
		return false
	}
	if f.MaxDisplayOrd != nil && img.DisplayOrder > *f.MaxDisplayOrd {
		return false
	}

	// audit by user
	if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
		if img.CreatedBy != strings.TrimSpace(*f.CreatedBy) {
			return false
		}
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		if img.UpdatedBy == nil || *img.UpdatedBy != strings.TrimSpace(*f.UpdatedBy) {
			return false
		}
	}
	if f.DeletedBy != nil && strings.TrimSpace(*f.DeletedBy) != "" {
		if img.DeletedBy == nil || *img.DeletedBy != strings.TrimSpace(*f.DeletedBy) {
			return false
		}
	}

	// date ranges (Created/Updated/Deleted)
	if f.CreatedFrom != nil && img.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !img.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil {
		if img.UpdatedAt == nil || img.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if img.UpdatedAt == nil || !img.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}
	if f.DeletedFrom != nil {
		if img.DeletedAt == nil || img.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if img.DeletedAt == nil || !img.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			if img.DeletedAt == nil {
				return false
			}
		} else {
			if img.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// ソート: 元の buildListImageOrderBy の意味を GCS上の in-memory slice に適用
func applyListImageSort(items []listimagedom.ListImage, sortOpt listimagedom.Sort) {
	col := strings.ToLower(string(sortOpt.Column))
	dir := strings.ToUpper(string(sortOpt.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}

	// default composite order: display_order ASC, created_at ASC, id ASC
	if col == "" {
		applyListImageSortForListID(items)
		return
	}

	// comparator
	less := func(i, j int) bool {
		switch col {
		case "id":
			if dir == "ASC" {
				return items[i].ID < items[j].ID
			}
			return items[i].ID > items[j].ID
		case "listid", "list_id":
			if items[i].ListID == items[j].ListID {
				if dir == "ASC" {
					return items[i].ID < items[j].ID
				}
				return items[i].ID > items[j].ID
			}
			if dir == "ASC" {
				return items[i].ListID < items[j].ListID
			}
			return items[i].ListID > items[j].ListID
		case "url":
			if dir == "ASC" {
				return items[i].URL < items[j].URL
			}
			return items[i].URL > items[j].URL
		case "filename", "file_name":
			if dir == "ASC" {
				return items[i].FileName < items[j].FileName
			}
			return items[i].FileName > items[j].FileName
		case "size":
			if dir == "ASC" {
				return items[i].Size < items[j].Size
			}
			return items[i].Size > items[j].Size
		case "displayorder", "display_order":
			if items[i].DisplayOrder == items[j].DisplayOrder {
				if dir == "ASC" {
					return items[i].ID < items[j].ID
				}
				return items[i].ID > items[j].ID
			}
			if dir == "ASC" {
				return items[i].DisplayOrder < items[j].DisplayOrder
			}
			return items[i].DisplayOrder > items[j].DisplayOrder
		case "createdat", "created_at":
			if dir == "ASC" {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
			return items[j].CreatedAt.Before(items[i].CreatedAt)
		case "updatedat", "updated_at":
			ui := items[i].UpdatedAt
			uj := items[j].UpdatedAt
			// nils first for ASC
			if ui == nil && uj == nil {
				if dir == "ASC" {
					return items[i].ID < items[j].ID
				}
				return items[i].ID > items[j].ID
			}
			if ui == nil {
				return dir == "ASC"
			}
			if uj == nil {
				return dir != "ASC"
			}
			if dir == "ASC" {
				return ui.Before(*uj)
			}
			return uj.Before(*ui)
		default:
			// fallback: display_order ASC, created_at ASC, id ASC
			if items[i].DisplayOrder != items[j].DisplayOrder {
				return items[i].DisplayOrder < items[j].DisplayOrder
			}
			if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
			return items[i].ID < items[j].ID
		}
	}

	sort.SliceStable(items, less)
}

// list_id 単位のデフォルト順序: display_order ASC, created_at ASC, id ASC
func applyListImageSortForListID(items []listimagedom.ListImage) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			a, b := items[i], items[j]
			swap := false
			if a.DisplayOrder != b.DisplayOrder {
				swap = a.DisplayOrder > b.DisplayOrder
			} else if !a.CreatedAt.Equal(b.CreatedAt) {
				swap = a.CreatedAt.After(b.CreatedAt)
			} else {
				swap = a.ID > b.ID
			}
			if swap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// ID 昇順ソート
func applyListImageSortByIDAsc(items []listimagedom.ListImage) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].ID > items[j].ID {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// resolveObjectFromImage:
// - URL が GCS URL ならそこから bucket/object を解決
// - それ以外は ID を object 名として扱う
func (r *ListImageRepositoryGCS) resolveObjectFromImage(
	img listimagedom.ListImage,
	defaultBucket string,
) (objectPath, bucket string, err error) {
	if strings.TrimSpace(img.URL) != "" {
		if b, obj, ok := gcscommon.ParseGCSURL(img.URL); ok {
			return strings.TrimLeft(obj, "/"), b, nil
		}
	}
	if id := strings.TrimSpace(img.ID); id != "" {
		return strings.TrimLeft(id, "/"), defaultBucket, nil
	}
	return "", "", fmt.Errorf("listImage: cannot resolve object path from input")
}
