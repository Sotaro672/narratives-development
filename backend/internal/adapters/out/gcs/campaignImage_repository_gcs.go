// backend/internal/adapters/out/gcs/campaignImage_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	cimgdom "narratives/internal/domain/campaignImage"
)

// GCS-based implementation for CampaignImage.
// - 画像ファイルは GCS に保存されている前提
// - メタ情報も基本的に GCS Object の属性/Metadata から構成する
// - ID は GCS の object 名（パス）として扱う
type CampaignImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// デフォルトバケット（元の実装と同等の役割）
const defaultCampaignImageBucket = "narratives_development_campaign_image"

func NewCampaignImageRepositoryGCS(client *storage.Client, bucket string) *CampaignImageRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultCampaignImageBucket
	}
	return &CampaignImageRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *CampaignImageRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultCampaignImageBucket
	}
	return b
}

// =======================
// Queries
// =======================

// List:
// - GCS オブジェクトを全走査して Filter/Sort/Page をメモリ上で適用
// - Sort は現状ほぼダミー（ID DESC をデフォルトとして扱う）
func (r *CampaignImageRepositoryGCS) List(
	ctx context.Context,
	filter cimgdom.Filter,
	sort cimgdom.Sort,
	page cimgdom.Page,
) (cimgdom.PageResult[cimgdom.CampaignImage], error) {
	if r.Client == nil {
		return cimgdom.PageResult[cimgdom.CampaignImage]{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})
	var all []cimgdom.CampaignImage

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return cimgdom.PageResult[cimgdom.CampaignImage]{}, err
		}
		img := buildCampaignImageFromAttrs(bucket, attrs)
		if matchCampaignImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	// sort（必要最低限: ID DESC / ASC のみ対応）
	applyCampaignImageSort(all, sort)

	// paging
	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return cimgdom.PageResult[cimgdom.CampaignImage]{
			Items:      []cimgdom.CampaignImage{},
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

	return cimgdom.PageResult[cimgdom.CampaignImage]{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCursor:
// - ID 昇順でソートしたとみなし、CursorPage.After 以降を返すシンプル実装
func (r *CampaignImageRepositoryGCS) ListByCursor(
	ctx context.Context,
	filter cimgdom.Filter,
	_ cimgdom.Sort,
	cpage cimgdom.CursorPage,
) (cimgdom.CursorPageResult[cimgdom.CampaignImage], error) {
	if r.Client == nil {
		return cimgdom.CursorPageResult[cimgdom.CampaignImage]{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []cimgdom.CampaignImage
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return cimgdom.CursorPageResult[cimgdom.CampaignImage]{}, err
		}
		img := buildCampaignImageFromAttrs(bucket, attrs)
		if matchCampaignImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	// ID 昇順にソート
	applyCampaignImageSortAscByID(all)

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
		return cimgdom.CursorPageResult[cimgdom.CampaignImage]{
			Items:      []cimgdom.CampaignImage{},
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

	return cimgdom.CursorPageResult[cimgdom.CampaignImage]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// GetByID:
// - id を object 名として扱い、GCS Attrs から構成
func (r *CampaignImageRepositoryGCS) GetByID(ctx context.Context, id string) (cimgdom.CampaignImage, error) {
	if r.Client == nil {
		return cimgdom.CampaignImage{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
	}

	bucket := r.bucket()
	attrs, err := r.Client.Bucket(bucket).Object(id).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}

	return buildCampaignImageFromAttrs(bucket, attrs), nil
}

func (r *CampaignImageRepositoryGCS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	id = strings.TrimSpace(id)
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

func (r *CampaignImageRepositoryGCS) Count(ctx context.Context, filter cimgdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("CampaignImageRepositoryGCS: nil storage client")
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
		img := buildCampaignImageFromAttrs(bucket, attrs)
		if matchCampaignImageFilter(img, filter) {
			total++
		}
	}
	return total, nil
}

// =======================
// Mutations (GCSメタ情報更新ベース)
// =======================

func (r *CampaignImageRepositoryGCS) Create(ctx context.Context, img cimgdom.CampaignImage) (cimgdom.CampaignImage, error) {
	// GCS ではオブジェクト作成自体は別レイヤー（アップロード側）で行う想定。
	// ここでは既存オブジェクトのメタ情報更新として Save を利用。
	return r.Save(ctx, img, nil)
}

func (r *CampaignImageRepositoryGCS) Update(ctx context.Context, id string, patch cimgdom.CampaignImagePatch) (cimgdom.CampaignImage, error) {
	if r.Client == nil {
		return cimgdom.CampaignImage{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
	}

	bucket := r.bucket()
	obj := r.Client.Bucket(bucket).Object(id)

	// まず既存 attrs を取得
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if patch.CampaignID != nil {
		// CampaignID はパス先頭要素から推測しているため、
		// ここでは GCS パス自体は変更しない（必要なら別レイヤーで rename）。
		meta["campaign_id"] = strings.TrimSpace(*patch.CampaignID)
	}
	if patch.Width != nil {
		meta["width"] = strconv.Itoa(*patch.Width)
	}
	if patch.Height != nil {
		meta["height"] = strconv.Itoa(*patch.Height)
	}
	if patch.FileSize != nil {
		meta["file_size"] = strconv.FormatInt(*patch.FileSize, 10)
	}
	if patch.MimeType != nil {
		ua.ContentType = strings.TrimSpace(*patch.MimeType)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return cimgdom.CampaignImage{}, err
	}

	return buildCampaignImageFromAttrs(bucket, newAttrs), nil
}

func (r *CampaignImageRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return cimgdom.ErrNotFound
	}

	bucket := r.bucket()
	err := r.Client.Bucket(bucket).Object(id).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return cimgdom.ErrNotFound
	}
	return err
}

// Save:
// - 渡された CampaignImage 情報から対象オブジェクトを解決し、メタ情報を更新した上で CampaignImage を返す
func (r *CampaignImageRepositoryGCS) Save(
	ctx context.Context,
	img cimgdom.CampaignImage,
	_ *cimgdom.SaveOptions,
) (cimgdom.CampaignImage, error) {
	if r.Client == nil {
		return cimgdom.CampaignImage{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}

	defaultBucket := r.bucket()
	objPath, bucket, err := r.resolveObjectFromImage(img, defaultBucket)
	if err != nil {
		return cimgdom.CampaignImage{}, err
	}

	obj := r.Client.Bucket(bucket).Object(objPath)

	// 既存 attrs を取得（存在しない場合はエラー）
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	// CampaignID はパス先頭要素から推測するが、明示された場合は metadata にも保持
	if strings.TrimSpace(img.CampaignID) != "" {
		meta["campaign_id"] = strings.TrimSpace(img.CampaignID)
	}

	if img.Width != nil {
		meta["width"] = strconv.Itoa(*img.Width)
	}
	if img.Height != nil {
		meta["height"] = strconv.Itoa(*img.Height)
	}
	if img.FileSize != nil {
		meta["file_size"] = strconv.FormatInt(*img.FileSize, 10)
	}
	if img.MimeType != nil && strings.TrimSpace(*img.MimeType) != "" {
		ua.ContentType = strings.TrimSpace(*img.MimeType)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return cimgdom.CampaignImage{}, err
	}

	return buildCampaignImageFromAttrs(bucket, newAttrs), nil
}

// SaveFromBucketObject:
// - bucket/objectPath とメタ情報から GCS オブジェクトのメタを更新し、CampaignImage を返す
func (r *CampaignImageRepositoryGCS) SaveFromBucketObject(
	ctx context.Context,
	id string,
	campaignID string,
	bucket string,
	objectPath string,
	width, height *int,
	fileSize *int64,
	mimeType *string,
	_ *string, // legacy
	_ time.Time, // legacy
) (cimgdom.CampaignImage, error) {
	if r.Client == nil {
		return cimgdom.CampaignImage{}, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.bucket()
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return cimgdom.CampaignImage{}, fmt.Errorf("campaignImage: empty objectPath")
	}

	handle := r.Client.Bucket(b).Object(obj)

	// 既存 attrs を取得
	attrs, err := handle.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if strings.TrimSpace(campaignID) != "" {
		meta["campaign_id"] = strings.TrimSpace(campaignID)
	}
	if width != nil {
		meta["width"] = strconv.Itoa(*width)
	}
	if height != nil {
		meta["height"] = strconv.Itoa(*height)
	}
	if fileSize != nil {
		meta["file_size"] = strconv.FormatInt(*fileSize, 10)
	}
	if mimeType != nil && strings.TrimSpace(*mimeType) != "" {
		ua.ContentType = strings.TrimSpace(*mimeType)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := handle.Update(ctx, ua)
	if err != nil {
		return cimgdom.CampaignImage{}, err
	}

	img := buildCampaignImageFromAttrs(b, newAttrs)

	// ID が指定されている場合は上書き的に扱う（ただし objectPath は変更しない）
	if strings.TrimSpace(id) != "" {
		img.ID = strings.TrimSpace(id)
	}

	return img, nil
}

// =======================
// GCSDeleteOpsProvider 実装
// =======================

// BuildDeleteOps:
// - 渡された ID / URL から削除対象オブジェクトを推定
func (r *CampaignImageRepositoryGCS) BuildDeleteOps(ctx context.Context, ids []string) ([]cimgdom.GCSDeleteOp, error) {
	_ = ctx

	if len(ids) == 0 {
		return nil, nil
	}

	ops := make([]cimgdom.GCSDeleteOp, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if b, obj, ok := parseGCSURL(id); ok {
			ops = append(ops, cimgdom.GCSDeleteOp{Bucket: b, ObjectPath: obj})
			continue
		}
		// ID を object path とみなす
		ops = append(ops, cimgdom.GCSDeleteOp{
			Bucket:     r.bucket(),
			ObjectPath: strings.TrimLeft(id, "/"),
		})
	}
	return ops, nil
}

// BuildDeleteOpsByCampaignID:
// - campaignID/ プレフィックス配下のオブジェクトを削除対象とみなす
func (r *CampaignImageRepositoryGCS) BuildDeleteOpsByCampaignID(ctx context.Context, campaignID string) ([]cimgdom.GCSDeleteOp, error) {
	if r.Client == nil {
		return nil, errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, nil
	}

	bucket := r.bucket()
	prefix := campaignID + "/"

	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var ops []cimgdom.GCSDeleteOp
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		ops = append(ops, cimgdom.GCSDeleteOp{
			Bucket:     bucket,
			ObjectPath: attrs.Name,
		})
	}
	return ops, nil
}

// =======================
// Helpers
// =======================

func gcsPublicURL(bucket, objectPath string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultCampaignImageBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)
}

func parseGCSURL(u string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(parsed.Host)
	if host != "storage.googleapis.com" && host != "storage.cloud.google.com" {
		return "", "", false
	}
	p := strings.TrimLeft(parsed.EscapedPath(), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.SplitN(p, "/", 2)
	if len(parts) < 2 {
		return "", "", false
	}
	bucket := parts[0]
	objectPath, _ := url.PathUnescape(parts[1])
	return bucket, objectPath, true
}

// buildCampaignImageFromAttrs converts GCS ObjectAttrs -> CampaignImage.
func buildCampaignImageFromAttrs(bucket string, attrs *storage.ObjectAttrs) cimgdom.CampaignImage {
	name := strings.TrimSpace(attrs.Name)

	// キャンペーンIDは "<campaignID>/..." の形式から推測
	var campaignID string
	if name != "" {
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			campaignID = strings.TrimSpace(parts[0])
		}
	}

	publicURL := gcsPublicURL(bucket, name)

	var widthPtr, heightPtr *int
	if w, ok := parseIntMeta(attrs.Metadata, "width"); ok {
		widthPtr = &w
	}
	if h, ok := parseIntMeta(attrs.Metadata, "height"); ok {
		heightPtr = &h
	}

	var sizePtr *int64
	if sz, ok := parseInt64Meta(attrs.Metadata, "file_size"); ok {
		sizePtr = &sz
	} else if attrs.Size > 0 {
		sz := attrs.Size
		sizePtr = &sz
	}

	var mtPtr *string
	if ct := strings.TrimSpace(attrs.ContentType); ct != "" {
		mtPtr = &ct
	} else if mt, ok := attrs.Metadata["mime_type"]; ok && strings.TrimSpace(mt) != "" {
		s := strings.TrimSpace(mt)
		mtPtr = &s
	}

	return cimgdom.CampaignImage{
		ID:         name,
		CampaignID: campaignID,
		ImageURL:   publicURL,
		Width:      widthPtr,
		Height:     heightPtr,
		FileSize:   sizePtr,
		MimeType:   mtPtr,
	}
}

func parseIntMeta(md map[string]string, key string) (int, bool) {
	if md == nil {
		return 0, false
	}
	if v, ok := md[key]; ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, false
		}
		n, err := strconv.Atoi(v)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func parseInt64Meta(md map[string]string, key string) (int64, bool) {
	if md == nil {
		return 0, false
	}
	if v, ok := md[key]; ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

// Filter in-memory (GCS オブジェクトから構成された CampaignImage に対して適用)
func matchCampaignImageFilter(img cimgdom.CampaignImage, f cimgdom.Filter) bool {
	// SearchQuery: imageURL / mimeType に対する部分一致
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		if !strings.Contains(strings.ToLower(img.ImageURL), lq) {
			mt := ""
			if img.MimeType != nil {
				mt = *img.MimeType
			}
			if !strings.Contains(strings.ToLower(mt), lq) {
				return false
			}
		}
	}

	// CampaignID
	if f.CampaignID != nil && strings.TrimSpace(*f.CampaignID) != "" {
		if img.CampaignID != strings.TrimSpace(*f.CampaignID) {
			return false
		}
	}
	if len(f.CampaignIDs) > 0 {
		ok := false
		for _, cid := range f.CampaignIDs {
			if img.CampaignID == strings.TrimSpace(cid) && img.CampaignID != "" {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// MimeTypes
	if len(f.MimeTypes) > 0 {
		if img.MimeType == nil || strings.TrimSpace(*img.MimeType) == "" {
			return false
		}
		mt := strings.TrimSpace(*img.MimeType)
		ok := false
		for _, m := range f.MimeTypes {
			if mt == strings.TrimSpace(m) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Size / Dimension ranges
	if f.WidthMin != nil && (img.Width == nil || *img.Width < *f.WidthMin) {
		return false
	}
	if f.WidthMax != nil && (img.Width == nil || *img.Width > *f.WidthMax) {
		return false
	}
	if f.HeightMin != nil && (img.Height == nil || *img.Height < *f.HeightMin) {
		return false
	}
	if f.HeightMax != nil && (img.Height == nil || *img.Height > *f.HeightMax) {
		return false
	}
	if f.FileSizeMin != nil && (img.FileSize == nil || *img.FileSize < *f.FileSizeMin) {
		return false
	}
	if f.FileSizeMax != nil && (img.FileSize == nil || *img.FileSize > *f.FileSizeMax) {
		return false
	}

	return true
}

// ソート: 今回はシンプルに ID 基準のみ対応（必要なら拡張）
func applyCampaignImageSort(items []cimgdom.CampaignImage, sort cimgdom.Sort) {
	col := strings.ToLower(string(sort.Column))
	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	switch col {
	case "id":
		if dir == "ASC" {
			applyCampaignImageSortAscByID(items)
		} else {
			// DESC
			for i := 0; i < len(items)-1; i++ {
				for j := i + 1; j < len(items); j++ {
					if items[i].ID < items[j].ID {
						items[i], items[j] = items[j], items[i]
					}
				}
			}
		}
	default:
		// デフォルト: ID DESC
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i].ID < items[j].ID {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
}

func applyCampaignImageSortAscByID(items []cimgdom.CampaignImage) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].ID > items[j].ID {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// resolveObjectFromImage:
// - ImageURL が GCS URL ならそこから bucket/object を解決
// - それ以外は ID を object 名として扱う
// - それでも無理ならエラー
func (r *CampaignImageRepositoryGCS) resolveObjectFromImage(
	img cimgdom.CampaignImage,
	defaultBucket string,
) (objectPath, bucket string, err error) {
	if strings.TrimSpace(img.ImageURL) != "" {
		if b, obj, ok := parseGCSURL(img.ImageURL); ok {
			return strings.TrimLeft(obj, "/"), b, nil
		}
	}
	if id := strings.TrimSpace(img.ID); id != "" {
		return strings.TrimLeft(id, "/"), defaultBucket, nil
	}
	return "", "", fmt.Errorf("campaignImage: cannot resolve object path from input")
}

// DeleteObjects helper if needed externally.
func (r *CampaignImageRepositoryGCS) DeleteObjects(ctx context.Context, ops []cimgdom.GCSDeleteOp) error {
	if r.Client == nil {
		return errors.New("CampaignImageRepositoryGCS: nil storage client")
	}
	if len(ops) == 0 {
		return nil
	}

	var errs []error
	for _, op := range ops {
		b := strings.TrimSpace(op.Bucket)
		if b == "" {
			b = r.bucket()
		}
		obj := strings.TrimLeft(strings.TrimSpace(op.ObjectPath), "/")
		if obj == "" {
			continue
		}
		if err := r.Client.Bucket(b).Object(obj).Delete(ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
			errs = append(errs, fmt.Errorf("%s/%s: %w", b, obj, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}
