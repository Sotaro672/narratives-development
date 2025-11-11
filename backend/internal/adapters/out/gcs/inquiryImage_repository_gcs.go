// backend/internal/adapters/out/firestore/inquiryImage_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	idom "narratives/internal/domain/inquiryImage"
)

// GCS-based implementation of inquiryImage.Repository.
//
// - 画像本体は GCS に保存されている前提
// - メタ情報も GCS Object の属性/Metadata を元に構築する
// - InquiryID/ファイル名はオブジェクト名やメタデータから復元する
type InquiryImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// デフォルトバケット（domain 側の定義を優先）
const defaultInquiryImageBucket = idom.DefaultBucket

func NewInquiryImageRepositoryGCS(client *storage.Client, bucket string) *InquiryImageRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultInquiryImageBucket
	}
	return &InquiryImageRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *InquiryImageRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultInquiryImageBucket
	}
	return b
}

// =======================
// Aggregate queries
// =======================

// GetImagesByInquiryID:
// - 対象 inquiryID にひもづく GCS オブジェクトを列挙し、InquiryImage を組み立てる
// - 紐付け判定は以下いずれか:
//   - Metadata["inquiry_id"] == inquiryID
//   - オブジェクト名が "<inquiryID>/" で始まる
//   - オブジェクト名が "inquiry_images/<inquiryID>/" で始まる（後方互換）
func (r *InquiryImageRepositoryGCS) GetImagesByInquiryID(ctx context.Context, inquiryID string) (*idom.InquiryImage, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil, idom.ErrNotFound
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var images []idom.ImageFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if belongsToInquiry(inquiryID, attrs) {
			images = append(images, buildImageFileFromAttrs(bucket, inquiryID, attrs))
		}
	}

	if len(images) == 0 {
		return nil, idom.ErrNotFound
	}

	// created_at ASC, file_name ASC 的な並びに揃える
	sortImagesByCreatedAndName(images)

	return &idom.InquiryImage{
		ID:     inquiryID,
		Images: images,
	}, nil
}

// Exists checks if an image exists for given (inquiryID, fileName).
// - 主に以下を試行:
//   - オブジェクト名: "inquiry_images/<inquiryID>/<fileName>"
//   - オブジェクト名: "<inquiryID>/<fileName>"
//   - いずれも無ければ false
func (r *InquiryImageRepositoryGCS) Exists(ctx context.Context, inquiryID, fileName string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	fileName = strings.TrimSpace(fileName)
	if inquiryID == "" || fileName == "" {
		return false, nil
	}
	bucket := r.bucket()

	candidates := []string{
		path.Join("inquiry_images", inquiryID, fileName),
		path.Join(inquiryID, fileName),
	}

	for _, objName := range candidates {
		_, err := r.Client.Bucket(bucket).Object(objName).Attrs(ctx)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, storage.ErrObjectNotExist) {
			// 本当のエラー
			return false, err
		}
	}
	return false, nil
}

// =======================
// Listing
// =======================

// ListImages:
// - バケット内を走査し、Filter/Sort/Page をメモリ上で適用
func (r *InquiryImageRepositoryGCS) ListImages(
	ctx context.Context,
	filter idom.Filter,
	sort idom.Sort,
	page idom.Page,
) (idom.PageResult[idom.ImageFile], error) {
	if r.Client == nil {
		return idom.PageResult[idom.ImageFile]{}, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []idom.ImageFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return idom.PageResult[idom.ImageFile]{}, err
		}
		img := buildImageFileFromAttrs(bucket, "", attrs)
		if matchImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	applyImageSort(all, sort)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return idom.PageResult[idom.ImageFile]{
			Items:      []idom.ImageFile{},
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

	return idom.PageResult[idom.ImageFile]{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListImagesByCursor:
// - (inquiry_id, file_name) 昇順でソートし、CursorPage.After から先を返す
// - Cursor は "inquiryID|fileName"
func (r *InquiryImageRepositoryGCS) ListImagesByCursor(
	ctx context.Context,
	filter idom.Filter,
	_ idom.Sort,
	cpage idom.CursorPage,
) (idom.CursorPageResult[idom.ImageFile], error) {
	if r.Client == nil {
		return idom.CursorPageResult[idom.ImageFile]{}, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []idom.ImageFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return idom.CursorPageResult[idom.ImageFile]{}, err
		}
		img := buildImageFileFromAttrs(bucket, "", attrs)
		if matchImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	// (inquiryID, fileName) ASC でソート
	sortImagesByInquiryAndName(all)

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	after := strings.TrimSpace(cpage.After)
	start := 0
	if after != "" {
		aid, fn := splitCursor(after)
		for i, im := range all {
			if compareInquiryFileKey(im.InquiryID, im.FileName, aid, fn) > 0 {
				start = i
				break
			}
		}
	}

	if start >= len(all) {
		return idom.CursorPageResult[idom.ImageFile]{
			Items:      []idom.ImageFile{},
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
		last := items[len(items)-1]
		cur := makeCursor(last.InquiryID, last.FileName)
		next = &cur
	}

	return idom.CursorPageResult[idom.ImageFile]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *InquiryImageRepositoryGCS) Count(ctx context.Context, filter idom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("InquiryImageRepositoryGCS: nil storage client")
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
		img := buildImageFileFromAttrs(bucket, "", attrs)
		if matchImageFilter(img, filter) {
			total++
		}
	}
	return total, nil
}

// =======================
// Mutations
// =======================

// AddImage:
// - AddImageRequest で指定された GCS オブジェクトに inquiry 用メタデータを付与し、InquiryImage 全体を返す。
// - 実ファイルのアップロードは別レイヤーで完了している前提。
func (r *InquiryImageRepositoryGCS) AddImage(
	ctx context.Context,
	inquiryID string,
	req idom.AddImageRequest,
) (*idom.InquiryImage, error) {
	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil, fmt.Errorf("inquiryImage: empty inquiryID")
	}
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}

	// URL から bucket/object を解決。失敗したらデフォルトバケット + 慣習パス。
	var bucket, objectPath string
	if b, obj, ok := parseGCSURL(req.FileURL); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = r.bucket()
		if strings.TrimSpace(req.FileName) != "" {
			objectPath = path.Join("inquiry_images", inquiryID, strings.TrimSpace(req.FileName))
		} else {
			return nil, fmt.Errorf("inquiryImage: cannot resolve objectPath from request")
		}
	}

	_, err := r.SaveImageFromBucketObject(
		ctx,
		inquiryID,
		strings.TrimSpace(req.FileName),
		bucket,
		objectPath,
		req.FileSize,
		strings.TrimSpace(req.MimeType),
		req.Width,
		req.Height,
		time.Now().UTC(),
		"system",
	)
	if err != nil {
		return nil, err
	}
	return r.GetImagesByInquiryID(ctx, inquiryID)
}

// UpdateImages:
//   - 既存（該当 inquiryID の）画像オブジェクトを論理的に「全削除」とみなし、指定 Images に置き換えるイメージ。
//   - 実際には BuildDeleteOpsByInquiryID + DeleteObjects で GCS オブジェクトを削除し、
//     新しいオブジェクトは SaveImageFromBucketObject でメタ更新する前提。
func (r *InquiryImageRepositoryGCS) UpdateImages(
	ctx context.Context,
	inquiryID string,
	req idom.UpdateImagesRequest,
) (*idom.InquiryImage, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil, fmt.Errorf("inquiryImage: empty inquiryID")
	}

	// 既存削除ターゲットを作って削除（best-effort）
	ops, err := r.BuildDeleteOpsByInquiryID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if err := r.DeleteObjects(ctx, ops); err != nil {
		return nil, err
	}

	// 新規群を保存（各 ImageFile の FileURL からオブジェクト解決する前提）
	for _, im := range req.Images {
		fn := strings.TrimSpace(im.FileName)
		if fn == "" {
			continue
		}
		b, obj, ok := parseGCSURL(im.FileURL)
		if !ok {
			// URL から取れない場合は慣習パスを使う
			b = r.bucket()
			obj = path.Join("inquiry_images", inquiryID, fn)
		}
		_, err := r.SaveImageFromBucketObject(
			ctx,
			inquiryID,
			fn,
			b,
			obj,
			im.FileSize,
			im.MimeType,
			im.Width,
			im.Height,
			im.CreatedAt,
			im.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
	}

	return r.GetImagesByInquiryID(ctx, inquiryID)
}

// PatchImage:
// - (inquiryID, fileName) に対応するオブジェクトのメタデータをパッチ更新
func (r *InquiryImageRepositoryGCS) PatchImage(
	ctx context.Context,
	inquiryID, fileName string,
	patch idom.ImagePatch,
) (*idom.ImageFile, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	fileName = strings.TrimSpace(fileName)
	if inquiryID == "" || fileName == "" {
		return nil, idom.ErrNotFound
	}

	bucket := r.bucket()
	objName := path.Join("inquiry_images", inquiryID, fileName)
	obj := r.Client.Bucket(bucket).Object(objName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			// fallback: "<inquiryID>/<fileName>"
			objName2 := path.Join(inquiryID, fileName)
			obj2 := r.Client.Bucket(bucket).Object(objName2)
			attrs, err = obj2.Attrs(ctx)
			if err != nil {
				if errors.Is(err, storage.ErrObjectNotExist) {
					return nil, idom.ErrNotFound
				}
				return nil, err
			}
			obj = obj2
			objName = objName2
		} else {
			return nil, err
		}
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := cloneMetadata(attrs.Metadata)

	if patch.FileName != nil && strings.TrimSpace(*patch.FileName) != "" {
		meta["file_name"] = strings.TrimSpace(*patch.FileName)
	}
	if patch.FileURL != nil && strings.TrimSpace(*patch.FileURL) != "" {
		meta["file_url"] = strings.TrimSpace(*patch.FileURL)
	}
	if patch.FileSize != nil {
		meta["file_size"] = strconv.FormatInt(*patch.FileSize, 10)
	}
	if patch.MimeType != nil {
		mt := strings.TrimSpace(*patch.MimeType)
		if mt != "" {
			ua.ContentType = mt
			meta["mime_type"] = mt
		}
	}
	if patch.Width != nil {
		meta["width"] = strconv.Itoa(*patch.Width)
	}
	if patch.Height != nil {
		meta["height"] = strconv.Itoa(*patch.Height)
	}
	if patch.UpdatedBy != nil {
		if v := strings.TrimSpace(*patch.UpdatedBy); v != "" {
			meta["updated_by"] = v
		}
	}
	if patch.UpdatedAt != nil {
		meta["updated_at"] = patch.UpdatedAt.UTC().Format(time.RFC3339Nano)
	} else if len(meta) > 0 {
		meta["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			delete(meta, "deleted_at")
		} else {
			meta["deleted_at"] = patch.DeletedAt.UTC().Format(time.RFC3339Nano)
		}
	}
	if patch.DeletedBy != nil {
		if v := strings.TrimSpace(*patch.DeletedBy); v == "" {
			delete(meta, "deleted_by")
		} else {
			meta["deleted_by"] = v
		}
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	im := buildImageFileFromAttrs(bucket, "", newAttrs)

	// cursor 用 key にもなるので InquiryID / FileName を補正
	if im.InquiryID == "" {
		im.InquiryID = inquiryID
	}
	if im.FileName == "" {
		im.FileName = fileName
	}

	return &im, nil
}

func (r *InquiryImageRepositoryGCS) DeleteImage(
	ctx context.Context,
	inquiryID, fileName string,
) (*idom.InquiryImage, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	fileName = strings.TrimSpace(fileName)
	if inquiryID == "" || fileName == "" {
		return nil, idom.ErrNotFound
	}

	ops, err := r.BuildDeleteOps(ctx, []idom.ImageKey{
		{InquiryID: inquiryID, FileName: fileName},
	})
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, idom.ErrNotFound
	}
	if err := r.DeleteObjects(ctx, ops); err != nil {
		return nil, err
	}

	// 残りの画像を返す（無ければ ErrNotFound）
	return r.GetImagesByInquiryID(ctx, inquiryID)
}

func (r *InquiryImageRepositoryGCS) DeleteAllImages(ctx context.Context, inquiryID string) error {
	if r.Client == nil {
		return errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil
	}
	ops, err := r.BuildDeleteOpsByInquiryID(ctx, inquiryID)
	if err != nil {
		return err
	}
	return r.DeleteObjects(ctx, ops)
}

func (r *InquiryImageRepositoryGCS) Save(
	ctx context.Context,
	agg idom.InquiryImage,
	_ *idom.SaveOptions,
) (*idom.InquiryImage, error) {
	// シンプルに UpdateImages 相当: 全削除 -> 全追加（Metadata 更新）
	req := idom.UpdateImagesRequest{
		Images: agg.Images,
	}
	return r.UpdateImages(ctx, agg.ID, req)
}

// =======================
// GCSObjectSaver / GCSDeleteOpsProvider
// =======================

// SaveImageFromBucketObject implements idom.GCSObjectSaver.
// - bucket が空なら defaultInquiryImageBucket
// - objectPath の既存オブジェクトにメタデータを設定し、ImageFile を返す
func (r *InquiryImageRepositoryGCS) SaveImageFromBucketObject(
	ctx context.Context,
	inquiryID string,
	fileName string,
	bucket string,
	objectPath string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	createdBy string,
) (*idom.ImageFile, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}

	inquiryID = strings.TrimSpace(inquiryID)
	fileName = strings.TrimSpace(fileName)
	if inquiryID == "" || fileName == "" {
		return nil, fmt.Errorf("inquiryImage: empty inquiryID or fileName")
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.bucket()
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return nil, fmt.Errorf("inquiryImage: empty objectPath")
	}

	handle := r.Client.Bucket(b).Object(obj)
	attrs, err := handle.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, idom.ErrNotFound
		}
		return nil, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := cloneMetadata(attrs.Metadata)

	meta["inquiry_id"] = inquiryID
	meta["file_name"] = fileName
	meta["file_url"] = gcsPublicURL(b, obj)

	if fileSize > 0 {
		meta["file_size"] = strconv.FormatInt(fileSize, 10)
	}
	if mt := strings.TrimSpace(mimeType); mt != "" {
		ua.ContentType = mt
		meta["mime_type"] = mt
	}
	if width != nil {
		meta["width"] = strconv.Itoa(*width)
	}
	if height != nil {
		meta["height"] = strconv.Itoa(*height)
	}

	cb := strings.TrimSpace(createdBy)
	if cb == "" {
		cb = "system"
	}
	meta["created_by"] = cb

	ca := createdAt.UTC()
	if ca.IsZero() {
		ca = time.Now().UTC()
	}
	meta["created_at"] = ca.Format(time.RFC3339Nano)

	ua.Metadata = meta

	newAttrs, err := handle.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	im := buildImageFileFromAttrs(b, inquiryID, newAttrs)
	return &im, nil
}

// BuildDeleteOps: 指定キー群の file_url から GCS 削除ターゲットを導出。
// - URL が GCS 形式ならそこから
// - そうでなければ "inquiry_images/<inquiryID>/<fileName>" を fallback に使う
func (r *InquiryImageRepositoryGCS) BuildDeleteOps(
	ctx context.Context,
	keys []idom.ImageKey,
) ([]idom.GCSDeleteOp, error) {
	_ = ctx

	if len(keys) == 0 {
		return nil, nil
	}

	ops := make([]idom.GCSDeleteOp, 0, len(keys))
	for _, k := range keys {
		aid := strings.TrimSpace(k.InquiryID)
		fn := strings.TrimSpace(k.FileName)
		if aid == "" || fn == "" {
			continue
		}
		// file_url が直接指定されているケース向けに構造を保持している場合、
		// 呼び出し側で URL を渡してくることを想定しているが、
		// この Repository ではキーしか知らないので fallback のみ行う。
		ops = append(ops, toInquiryImageGCSDeleteOpFromURL("", aid, fn))
	}
	return ops, nil
}

// BuildDeleteOpsByInquiryID:
// - inquiryID 配下の慣習パスオブジェクトを全て削除対象とする
func (r *InquiryImageRepositoryGCS) BuildDeleteOpsByInquiryID(
	ctx context.Context,
	inquiryID string,
) ([]idom.GCSDeleteOp, error) {
	if r.Client == nil {
		return nil, errors.New("InquiryImageRepositoryGCS: nil storage client")
	}
	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil, nil
	}

	bucket := r.bucket()
	// 主に "inquiry_images/<inquiryID>/" プレフィックスを見る
	prefixes := []string{
		path.Join("inquiry_images", inquiryID) + "/",
		inquiryID + "/",
	}

	var ops []idom.GCSDeleteOp
	for _, pfx := range prefixes {
		it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{
			Prefix: pfx,
		})
		for {
			attrs, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}
			ops = append(ops, idom.GCSDeleteOp{
				Bucket:     bucket,
				ObjectPath: attrs.Name,
			})
		}
	}

	return ops, nil
}

// =======================
// DeleteObjects helper
// =======================

func (r *InquiryImageRepositoryGCS) DeleteObjects(ctx context.Context, ops []idom.GCSDeleteOp) error {
	if r.Client == nil {
		return errors.New("InquiryImageRepositoryGCS: nil storage client")
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
		err := r.Client.Bucket(b).Object(obj).Delete(ctx)
		if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
			errs = append(errs, fmt.Errorf("%s/%s: %w", b, obj, err))
		}
	}

	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}

// =======================
// Helpers
// =======================

func gcsPublicURL(bucket, objectPath string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultInquiryImageBucket
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

func belongsToInquiry(inquiryID string, attrs *storage.ObjectAttrs) bool {
	if attrs == nil {
		return false
	}
	inq := strings.TrimSpace(inquiryID)
	if inq == "" {
		return false
	}
	// metadata
	if v, ok := attrs.Metadata["inquiry_id"]; ok && strings.TrimSpace(v) == inq {
		return true
	}
	// path prefix: "inquiry_images/<inq>/" or "<inq>/"
	name := strings.TrimSpace(attrs.Name)
	if strings.HasPrefix(name, path.Join("inquiry_images", inq)+"/") {
		return true
	}
	if strings.HasPrefix(name, inq+"/") {
		return true
	}
	return false
}

// buildImageFileFromAttrs:
// inquiryID が空なら metadata / パスから推測する
func buildImageFileFromAttrs(bucket, inquiryID string, attrs *storage.ObjectAttrs) idom.ImageFile {
	if attrs == nil {
		return idom.ImageFile{}
	}

	md := attrs.Metadata
	name := strings.TrimSpace(attrs.Name)

	inq := strings.TrimSpace(inquiryID)
	if inq == "" {
		if v, ok := md["inquiry_id"]; ok && strings.TrimSpace(v) != "" {
			inq = strings.TrimSpace(v)
		} else if strings.HasPrefix(name, "inquiry_images/") {
			parts := strings.SplitN(strings.TrimPrefix(name, "inquiry_images/"), "/", 2)
			if len(parts) == 2 {
				inq = strings.TrimSpace(parts[0])
			}
		} else {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				inq = strings.TrimSpace(parts[0])
			}
		}
	}

	// fileName
	var fn string
	if v, ok := md["file_name"]; ok && strings.TrimSpace(v) != "" {
		fn = strings.TrimSpace(v)
	} else {
		if idx := strings.LastIndex(name, "/"); idx >= 0 && idx+1 < len(name) {
			fn = name[idx+1:]
		} else {
			fn = name
		}
	}

	publicURL := gcsPublicURL(bucket, name)
	if v, ok := md["file_url"]; ok && strings.TrimSpace(v) != "" {
		publicURL = strings.TrimSpace(v)
	}

	// size
	var size int64
	if sz, ok := parseInt64Meta(md, "file_size"); ok {
		size = sz
	} else if attrs.Size > 0 {
		size = attrs.Size
	}

	// mime
	mt := strings.TrimSpace(attrs.ContentType)
	if v, ok := md["mime_type"]; ok && strings.TrimSpace(v) != "" {
		mt = strings.TrimSpace(v)
	}

	// width/height
	var widthPtr, heightPtr *int
	if w, ok := parseIntMeta(md, "width"); ok {
		widthPtr = &w
	}
	if h, ok := parseIntMeta(md, "height"); ok {
		heightPtr = &h
	}

	// created/updated/deleted
	createdAt := attrs.Created
	if v, ok := md["created_at"]; ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			createdAt = t
		}
	}
	createdBy := strings.TrimSpace(md["created_by"])

	var updatedAtPtr *time.Time
	if v, ok := md["updated_at"]; ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			tu := t.UTC()
			updatedAtPtr = &tu
		}
	}
	updatedByPtr := ptrString(trimOrEmpty(md["updated_by"]))

	var deletedAtPtr *time.Time
	if v, ok := md["deleted_at"]; ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			tu := t.UTC()
			deletedAtPtr = &tu
		}
	}
	deletedByPtr := ptrString(trimOrEmpty(md["deleted_by"]))

	return idom.ImageFile{
		InquiryID: inq,
		FileName:  fn,
		FileURL:   publicURL,
		FileSize:  size,
		MimeType:  mt,
		Width:     widthPtr,
		Height:    heightPtr,
		CreatedAt: createdAt.UTC(),
		CreatedBy: createdBy,
		UpdatedAt: updatedAtPtr,
		UpdatedBy: updatedByPtr,
		DeletedAt: deletedAtPtr,
		DeletedBy: deletedByPtr,
	}
}

// matchImageFilter: GCS ベースの ImageFile に対して最低限の Filter を適用
func matchImageFilter(im idom.ImageFile, f idom.Filter) bool {
	// InquiryID
	if f.InquiryID != nil && strings.TrimSpace(*f.InquiryID) != "" {
		if im.InquiryID != strings.TrimSpace(*f.InquiryID) {
			return false
		}
	}

	// SearchQuery: FileName / FileURL / MimeType 対象
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		if !strings.Contains(strings.ToLower(im.FileName), lq) &&
			!strings.Contains(strings.ToLower(im.FileURL), lq) &&
			!strings.Contains(strings.ToLower(im.MimeType), lq) {
			return false
		}
	}

	return true
}

// applyImageSort: 最低限 idom.Sort を解釈（未指定時は created_at DESC 相当）
func applyImageSort(items []idom.ImageFile, sort idom.Sort) {
	col := strings.ToLower(string(sort.Column))
	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	switch col {
	case "inquiryid", "inquiry_id":
		sortImagesByInquiryAndName(items)
		if dir == "DESC" {
			reverseImages(items)
		}
	case "filename", "file_name":
		sortImagesByFileName(items)
		if dir == "DESC" {
			reverseImages(items)
		}
	default:
		// デフォルト: created_at DESC, inquiry_id DESC, file_name DESC
		sortImagesByCreatedAndName(items)
		if dir == "ASC" {
			reverseImages(items)
		}
	}
}

func sortImagesByInquiryAndName(items []idom.ImageFile) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if compareInquiryFileKey(items[i].InquiryID, items[i].FileName, items[j].InquiryID, items[j].FileName) > 0 {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortImagesByFileName(items []idom.ImageFile) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].FileName > items[j].FileName {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortImagesByCreatedAndName(items []idom.ImageFile) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			li, lj := items[i], items[j]
			if li.CreatedAt.After(lj.CreatedAt) {
				items[i], items[j] = items[j], items[i]
			} else if li.CreatedAt.Equal(lj.CreatedAt) {
				if compareInquiryFileKey(li.InquiryID, li.FileName, lj.InquiryID, lj.FileName) > 0 {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
}

func reverseImages(items []idom.ImageFile) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func compareInquiryFileKey(aInq, aFn, bInq, bFn string) int {
	if aInq < bInq {
		return -1
	}
	if aInq > bInq {
		return 1
	}
	if aFn < bFn {
		return -1
	}
	if aFn > bFn {
		return 1
	}
	return 0
}

func makeCursor(inquiryID, fileName string) string {
	return strings.TrimSpace(inquiryID) + "|" + strings.TrimSpace(fileName)
}

func splitCursor(cur string) (string, string) {
	parts := strings.SplitN(cur, "|", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// toInquiryImageGCSDeleteOpFromURL:
// - fileURL が GCS URL の場合はそこから
// - それ以外は "inquiry_images/<inquiryID>/<fileName>" fallback
func toInquiryImageGCSDeleteOpFromURL(fileURL, inquiryID, fileName string) idom.GCSDeleteOp {
	if b, obj, ok := parseGCSURL(fileURL); ok {
		return idom.GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	return idom.GCSDeleteOp{
		Bucket:     defaultInquiryImageBucket,
		ObjectPath: path.Join("inquiry_images", strings.TrimSpace(inquiryID), strings.TrimSpace(fileName)),
	}
}

func cloneMetadata(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func trimOrEmpty(s string) string {
	return strings.TrimSpace(s)
}
