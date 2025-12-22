// backend/internal/adapters/out/gcs/listImage_repository_gcs.go
package gcs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	listimagedom "narratives/internal/domain/listImage"
)

// Env var key (Cloud Run / local .env) for list image bucket.
const EnvListImageBucket = "LIST_IMAGE_BUCKET"

// Fallback bucket name (期待値: narratives-development-list).
const FallbackListImageBucket = "narratives-development-list"

// ListImageRepositoryGCS is a small GCS adapter for List images.
// - 「listId/」配下に複数画像を保存できる設計
// - objectPath: {listId}/{imageId}/{fileName}
//
// NOTE:
// - gcs/tokenIcon_repository_gcs.go に sanitizeFileName が既にあるため、こちらでは定義しない。
// - ファイル名の正規化は tokenIcon 側 sanitizeFileName を再利用する（同一 package gcs）。
type ListImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string // optional (if empty, use env or fallback)
}

func NewListImageRepositoryGCS(client *storage.Client, bucket string) *ListImageRepositoryGCS {
	return &ListImageRepositoryGCS{
		Client: client,
		Bucket: strings.TrimSpace(bucket),
	}
}

// ResolveBucket decides bucket by:
// 1) repository.Bucket
// 2) env LIST_IMAGE_BUCKET
// 3) FallbackListImageBucket
func (r *ListImageRepositoryGCS) ResolveBucket() string {
	if r != nil {
		if b := strings.TrimSpace(r.Bucket); b != "" {
			return b
		}
	}
	if b := strings.TrimSpace(os.Getenv(EnvListImageBucket)); b != "" {
		return b
	}
	return FallbackListImageBucket
}

// UploadResult is a pure GCS result (record creation is handled elsewhere).
type UploadResult struct {
	Bucket      string
	ObjectPath  string
	PublicURL   string
	Size        int64
	ContentType string
	FileName    string // normalized filename actually used
	ImageID     string // resolved (generated if empty)
}

// UploadDataURL uploads a data URL image to GCS.
// It validates payload using domain validator and stores it under:
//
//	{listId}/{imageId}/{fileName}
//
// - listID: required（期待値: list_create 時の id をディレクトリ名に）
// - imageID: optional（空なら生成。複数画像対応のため imageId で衝突回避）
// - fileName: optional（空なら "image" + 推定拡張子）
func (r *ListImageRepositoryGCS) UploadDataURL(
	ctx context.Context,
	listID string,
	imageID string,
	fileName string,
	dataURL string,
) (UploadResult, error) {
	if r == nil || r.Client == nil {
		return UploadResult{}, fmt.Errorf("ListImageRepositoryGCS.UploadDataURL: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return UploadResult{}, fmt.Errorf("ListImageRepositoryGCS.UploadDataURL: listID is empty")
	}

	// Validate and decode (domain policy)
	mime, payload, err := listimagedom.ValidateDataURL(
		strings.TrimSpace(dataURL),
		int(listimagedom.DefaultMaxImageSizeBytes),
		listimagedom.SupportedImageMIMEs,
	)
	if err != nil {
		return UploadResult{}, err
	}

	// Normalize fileName (reuse sanitizeFileName from tokenIcon_repository_gcs.go)
	normName := strings.TrimSpace(fileName)
	if normName == "" {
		normName = "image"
	}
	normName = sanitizeFileName(normName)
	normName = ensureExtensionByMIME(normName, mime)

	// Resolve imageID
	imgID := strings.TrimSpace(imageID)
	if imgID == "" {
		imgID = newObjectID()
	}

	objPath, err := buildListImageObjectPath(listID, imgID, normName)
	if err != nil {
		return UploadResult{}, err
	}

	bucket := r.ResolveBucket()

	w := r.Client.Bucket(bucket).Object(objPath).NewWriter(ctx)
	w.ContentType = mime
	w.Metadata = map[string]string{
		"listId":     listID,
		"imageId":    imgID,
		"fileName":   normName,
		"uploadedAt": time.Now().UTC().Format(time.RFC3339Nano),
	}

	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return UploadResult{}, fmt.Errorf("ListImageRepositoryGCS.UploadDataURL: write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return UploadResult{}, fmt.Errorf("ListImageRepositoryGCS.UploadDataURL: close failed: %w", err)
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objPath)

	return UploadResult{
		Bucket:      bucket,
		ObjectPath:  objPath,
		PublicURL:   publicURL,
		Size:        int64(len(payload)),
		ContentType: mime,
		FileName:    normName,
		ImageID:     imgID,
	}, nil
}

// DeleteObject deletes a GCS object by bucket/objectPath.
func (r *ListImageRepositoryGCS) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("ListImageRepositoryGCS.DeleteObject: storage client is nil")
	}
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.ResolveBucket()
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return fmt.Errorf("ListImageRepositoryGCS.DeleteObject: objectPath is empty")
	}

	if err := r.Client.Bucket(b).Object(obj).Delete(ctx); err != nil {
		return fmt.Errorf("ListImageRepositoryGCS.DeleteObject: delete failed: %w", err)
	}
	return nil
}

// ============================================================
// usecase required methods (方針A)
// - ListByListID(ctx, listID) ([]ListImage, error)
// - GetByID(ctx, id) (ListImage, error)
// - SaveFromBucketObject(ctx, bucket, objectPath, listID, fileName, size, displayOrder, createdBy, createdAt) (ListImage, error)
// ============================================================

// ListByListID lists images under "{listId}/" prefix.
// It returns domain ListImage items (best-effort).
func (r *ListImageRepositoryGCS) ListByListID(ctx context.Context, listID string) ([]listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("ListImageRepositoryGCS.ListByListID: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listimagedom.ListImage{}, nil
	}

	bucket := r.ResolveBucket()
	prefix := sanitizePathSegment(listID)
	if prefix == "" {
		return []listimagedom.ListImage{}, nil
	}
	prefix = prefix + "/"

	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})

	out := make([]listimagedom.ListImage, 0, 16)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ListImageRepositoryGCS.ListByListID: iterate failed: %w", err)
		}
		if attrs == nil {
			continue
		}

		// ".keep" は除外（tokenIcon 側の helper を再利用）
		if isKeepObject(attrs.Name) {
			continue
		}

		li, ok := buildListImageFromAttrs(bucket, attrs)
		if !ok {
			continue
		}

		// listID で絞り込み（メタ不正/パス不正の保険）
		if strings.TrimSpace(li.ListID) != listID {
			continue
		}

		out = append(out, li)
	}

	return out, nil
}

// GetByID gets a ListImage by id.
// id can be:
// - objectPath within the bucket (e.g. "{listId}/{imageId}/{fileName}")
// - https://storage.googleapis.com/{bucket}/{objectPath}
func (r *ListImageRepositoryGCS) GetByID(ctx context.Context, id string) (listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: storage client is nil")
	}

	bucket, objectPath, err := resolveBucketObjectForListImage(strings.TrimSpace(id), r.ResolveBucket())
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: attrs failed: %w", err)
	}

	li, ok := buildListImageFromAttrs(bucket, attrs)
	if !ok {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: failed to build list image from attrs")
	}
	return li, nil
}

// SaveFromBucketObject:
// - already-uploaded object を前提に、GCS metadata を整えて domain ListImage を返す。
// - objectPath policy は "{listId}/{imageId}/{fileName}" を推奨。
// - id は objectPath を採用（= GetByID で一意に引ける）
//
// NOTE: usecase 側 interface に合わせたシグネチャ
func (r *ListImageRepositoryGCS) SaveFromBucketObject(
	ctx context.Context,
	bucket string,
	objectPath string,
	listID string,
	fileName string,
	size int64,
	displayOrder int,
	createdBy string,
	createdAt time.Time,
) (listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidListID
	}

	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidCreatedBy
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.ResolveBucket()
	}

	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: objectPath is empty")
	}

	// object exists?
	o := r.Client.Bucket(b).Object(obj)
	attrs, err := o.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: attrs failed: %w", err)
	}

	// imageId は objectPath から推定して metadata に入れる（複数画像対応）
	_, imageID, ok := splitListImageObjectPath(obj)
	if !ok || strings.TrimSpace(imageID) == "" {
		imageID = newObjectID()
	}

	fn := strings.TrimSpace(fileName)
	if fn == "" {
		fn = path.Base(obj)
	}
	fn = sanitizeFileName(fn)
	if fn == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidFileName
	}

	// size reconcile
	finalSize := size
	if finalSize <= 0 {
		finalSize = attrs.Size
	}
	if finalSize < 0 {
		finalSize = 0
	}

	// createdAt default
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	} else {
		createdAt = createdAt.UTC()
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)

	// merge metadata
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	meta["listId"] = listID
	meta["imageId"] = imageID
	meta["fileName"] = fn
	meta["url"] = publicURL
	meta["size"] = fmt.Sprint(finalSize)
	meta["displayOrder"] = fmt.Sprint(displayOrder)
	meta["createdAt"] = createdAt.Format(time.RFC3339Nano)
	meta["createdBy"] = createdBy

	// metadata update
	newAttrs, err := o.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: update metadata failed: %w", err)
	}

	// domain object (id = objectPath)
	li, derr := listimagedom.NewFromGCSObject(
		strings.TrimSpace(newAttrs.Name), // id
		listID,
		fn,
		finalSize,
		displayOrder,
		b,
		newAttrs.Name,
		createdAt,
		createdBy,
		nil, nil, nil, nil,
	)
	if derr != nil {
		// best-effort fallback（domain validate が落ちても UI を止めない）
		tmp := listimagedom.ListImage{
			ID:           strings.TrimSpace(newAttrs.Name),
			ListID:       listID,
			URL:          publicURL,
			FileName:     fn,
			Size:         finalSize,
			DisplayOrder: displayOrder,
			CreatedAt:    createdAt,
			CreatedBy:    createdBy,
		}
		return tmp, nil
	}

	// URL はメタの url を優先
	if strings.TrimSpace(publicURL) != "" {
		_ = li.UpdateURL(publicURL)
	}

	return li, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func buildListImageObjectPath(listID, imageID, fileName string) (string, error) {
	lid := sanitizePathSegment(listID)
	iid := sanitizePathSegment(imageID)

	// fileName は sanitizeFileName() 済みで入ってくる想定（同一 package の関数を再利用）
	fn := strings.TrimSpace(fileName)

	if lid == "" {
		return "", fmt.Errorf("listImage: invalid listID for object path")
	}
	if iid == "" {
		return "", fmt.Errorf("listImage: invalid imageID for object path")
	}
	if fn == "" {
		return "", fmt.Errorf("listImage: invalid fileName for object path")
	}

	// {listId}/{imageId}/{fileName}
	return path.Join(lid, iid, fn), nil
}

func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// prohibit separators
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "/", "_")
	// trim dots/spaces to avoid weird paths
	s = strings.Trim(s, ". ")
	return s
}

func ensureExtensionByMIME(fileName string, mime string) string {
	lower := strings.ToLower(strings.TrimSpace(fileName))

	// If already has an extension, keep it
	if strings.Contains(path.Base(lower), ".") {
		return fileName
	}

	ext := ""
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		ext = ""
	}

	if ext == "" {
		return fileName
	}
	return fileName + ext
}

func newObjectID() string {
	// 12 bytes random => 24 hex chars
	b := make([]byte, 12)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// fallback
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

func resolveBucketObjectForListImage(id string, fallbackBucket string) (bucket string, objectPath string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", listimagedom.ErrNotFound
	}

	// domain helper: ParseGCSURL supports storage.googleapis.com / storage.cloud.google.com
	if b, obj, ok := listimagedom.ParseGCSURL(id); ok {
		bucket, objectPath = strings.TrimSpace(b), strings.TrimLeft(strings.TrimSpace(obj), "/")
	} else {
		bucket = strings.TrimSpace(fallbackBucket)
		objectPath = strings.TrimLeft(strings.TrimSpace(id), "/")
	}

	if bucket == "" || objectPath == "" {
		return "", "", listimagedom.ErrNotFound
	}
	return bucket, objectPath, nil
}

// buildListImageFromAttrs converts GCS attrs to domain ListImage (best-effort).
func buildListImageFromAttrs(bucket string, attrs *storage.ObjectAttrs) (listimagedom.ListImage, bool) {
	if attrs == nil {
		return listimagedom.ListImage{}, false
	}

	obj := strings.TrimSpace(attrs.Name)
	if obj == "" {
		return listimagedom.ListImage{}, false
	}

	meta := attrs.Metadata
	getMeta := func(k string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[k])
	}

	listID, _, ok := splitListImageObjectPath(obj)
	if !ok {
		// 期待ポリシー外の object は除外（安全側）
		return listimagedom.ListImage{}, false
	}

	// fileName
	fileName := getMeta("fileName")
	if fileName == "" {
		fileName = path.Base(obj)
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		return listimagedom.ListImage{}, false
	}

	// url
	url := getMeta("url")
	if url == "" {
		url = fmt.Sprintf("https://storage.googleapis.com/%s/%s", strings.TrimSpace(bucket), obj)
	}

	// size
	size := attrs.Size
	if v := getMeta("size"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			size = n
		}
	}

	// displayOrder
	displayOrder := 0
	if v := getMeta("displayOrder"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			displayOrder = n
		}
	}

	// createdAt / createdBy（domain は必須なので best-effort）
	createdAt := attrs.Created
	if v := getMeta("createdAt"); v != "" {
		if t, e := time.Parse(time.RFC3339, v); e == nil {
			createdAt = t.UTC()
		}
	}
	createdBy := getMeta("createdBy")
	if createdBy == "" {
		createdBy = "system"
	}

	// id は objectPath を採用（= GetByID で一意）
	id := obj

	li, err := listimagedom.NewFromGCSObject(
		id,
		listID,
		fileName,
		size,
		displayOrder,
		strings.TrimSpace(bucket),
		obj,
		createdAt,
		createdBy,
		nil, nil, nil, nil,
	)
	if err != nil {
		// best-effort fallback
		tmp := listimagedom.ListImage{
			ID:           id,
			ListID:       listID,
			URL:          url,
			FileName:     fileName,
			Size:         size,
			DisplayOrder: displayOrder,
			CreatedAt:    createdAt.UTC(),
			CreatedBy:    createdBy,
		}
		return tmp, true
	}

	// constructor の URL は PublicURL(bucket,obj) になるので、メタの url を優先したい場合は差し替え
	if strings.TrimSpace(url) != "" {
		_ = li.UpdateURL(url)
	}
	return li, true
}

// splitListImageObjectPath expects "{listId}/{imageId}/{fileName}".
func splitListImageObjectPath(objectPath string) (listID string, imageID string, ok bool) {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		return "", "", false
	}
	listID = strings.TrimSpace(parts[0])
	imageID = strings.TrimSpace(parts[1])
	if listID == "" || imageID == "" {
		return "", "", false
	}
	return listID, imageID, true
}
