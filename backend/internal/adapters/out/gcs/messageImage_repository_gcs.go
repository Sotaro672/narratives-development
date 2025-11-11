// backend/internal/adapters/out/gcs/messageImage_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	gcscommon "narratives/internal/adapters/out/gcs/common"
	midom "narratives/internal/domain/messageImage"
)

// GCS ベースの MessageImage リポジトリ。
// - 画像本体は GCS に保存されている前提
// - メタ情報は GCS ObjectAttrs / Metadata から構成
// - オブジェクト名の規約: 通常は "<messageID>/<fileName>" を想定
type MessageImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// デフォルトバケット（midom.DefaultBucket があればそれを優先）
const defaultMessageImageBucket = "narratives_development_message_image"

func NewMessageImageRepositoryGCS(client *storage.Client, bucket string) *MessageImageRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		if strings.TrimSpace(midom.DefaultBucket) != "" {
			b = midom.DefaultBucket
		} else {
			b = defaultMessageImageBucket
		}
	}
	return &MessageImageRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *MessageImageRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		if strings.TrimSpace(midom.DefaultBucket) != "" {
			return midom.DefaultBucket
		}
		return defaultMessageImageBucket
	}
	return b
}

// ========================================
// RepositoryPort impl (GCS版)
// ========================================

// ListByMessageID:
// - messageID に紐づくオブジェクトを列挙（Prefix ベース）
// - deleted_at があるものは除外
// - created_at ASC, file_name ASC 相当でソート
func (r *MessageImageRepositoryGCS) ListByMessageID(ctx context.Context, messageID string) ([]midom.ImageFile, error) {
	if r.Client == nil {
		return nil, errors.New("MessageImageRepositoryGCS: nil storage client")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return []midom.ImageFile{}, nil
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: messageID + "/",
	})

	var out []midom.ImageFile
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		img := buildMessageImageFromAttrs(bucket, attrs)
		if img.MessageID == messageID && img.DeletedAt == nil {
			out = append(out, img)
		}
	}

	// created_at ASC, file_name ASC
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			swap := false
			if out[i].CreatedAt.After(out[j].CreatedAt) {
				swap = true
			} else if out[i].CreatedAt.Equal(out[j].CreatedAt) &&
				out[i].FileName > out[j].FileName {
				swap = true
			}
			if swap {
				out[i], out[j] = out[j], out[i]
			}
		}
	}

	return out, nil
}

// Get:
// - messageID / fileName から単一画像を取得
// - 規約 "<messageID>/<fileName>" のオブジェクトを参照
// - 見つからない場合は ErrNotFound
func (r *MessageImageRepositoryGCS) Get(ctx context.Context, messageID, fileName string) (*midom.ImageFile, error) {
	if r.Client == nil {
		return nil, errors.New("MessageImageRepositoryGCS: nil storage client")
	}
	messageID = strings.TrimSpace(messageID)
	fileName = strings.TrimSpace(fileName)
	if messageID == "" || fileName == "" {
		return nil, midom.ErrNotFound
	}

	bucket := r.bucket()
	objName := messageID + "/" + fileName

	attrs, err := r.Client.Bucket(bucket).Object(objName).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			// 念のため metadata ベースの fallback も試す
			it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})
			for {
				oa, e := it.Next()
				if errors.Is(e, iterator.Done) {
					break
				}
				if e != nil {
					return nil, e
				}
				img := buildMessageImageFromAttrs(bucket, oa)
				if img.MessageID == messageID && img.FileName == fileName {
					return &img, nil
				}
			}
			return nil, midom.ErrNotFound
		}
		return nil, err
	}

	img := buildMessageImageFromAttrs(bucket, attrs)
	if img.MessageID != messageID || img.FileName != fileName {
		return nil, midom.ErrNotFound
	}
	return &img, nil
}

// List:
// - 全オブジェクトを列挙して Filter/Sort/Page をメモリ上で適用
func (r *MessageImageRepositoryGCS) List(
	ctx context.Context,
	filter midom.Filter,
	sort midom.Sort,
	page midom.Page,
) (midom.PageResult, error) {
	if r.Client == nil {
		return midom.PageResult{}, errors.New("MessageImageRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	all := make([]midom.ImageFile, 0)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return midom.PageResult{}, err
		}
		img := buildMessageImageFromAttrs(bucket, attrs)
		if matchMessageImageFilter(img, filter) {
			all = append(all, img)
		}
	}

	applyMessageImageSort(all, sort)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return midom.PageResult{
			Items:      []midom.ImageFile{},
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

	return midom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *MessageImageRepositoryGCS) Count(ctx context.Context, filter midom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("MessageImageRepositoryGCS: nil storage client")
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
		img := buildMessageImageFromAttrs(bucket, attrs)
		if matchMessageImageFilter(img, filter) {
			total++
		}
	}
	return total, nil
}

// Add:
// - 実際の GCS アップロードが完了している前提で、該当オブジェクトのメタデータを更新して ImageFile を返す。
func (r *MessageImageRepositoryGCS) Add(ctx context.Context, img midom.ImageFile) (midom.ImageFile, error) {
	if r.Client == nil {
		return midom.ImageFile{}, errors.New("MessageImageRepositoryGCS: nil storage client")
	}

	defaultBucket := r.bucket()
	var bucket, objectPath string

	// 1) FileURL が GCS URL なら最優先
	if u := strings.TrimSpace(img.FileURL); u != "" {
		if b, obj, ok := gcscommon.ParseGCSURL(u); ok {
			bucket = b
			objectPath = obj
		}
	}

	// 2) messageID / fileName から推測
	if bucket == "" || objectPath == "" {
		mid := strings.TrimSpace(img.MessageID)
		fn := strings.TrimSpace(img.FileName)
		if mid != "" && fn != "" {
			bucket = defaultBucket
			objectPath = mid + "/" + fn
		}
	}

	if bucket == "" || strings.TrimSpace(objectPath) == "" {
		return midom.ImageFile{}, fmt.Errorf("messageImage: cannot resolve object path from input")
	}

	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	obj := r.Client.Bucket(bucket).Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return midom.ImageFile{}, midom.ErrNotFound
		}
		return midom.ImageFile{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	// 基本メタ設定
	if m := strings.TrimSpace(img.MessageID); m != "" {
		meta["message_id"] = m
	}
	if fn := strings.TrimSpace(img.FileName); fn != "" {
		meta["file_name"] = fn
	}
	if u := strings.TrimSpace(img.FileURL); u != "" {
		meta["file_url"] = u
	}
	if img.FileSize > 0 {
		meta["file_size"] = fmt.Sprint(img.FileSize)
	}
	if mt := strings.TrimSpace(img.MimeType); mt != "" {
		ua.ContentType = mt
		meta["mime_type"] = mt
	}
	if img.Width != nil {
		meta["width"] = fmt.Sprint(*img.Width)
	}
	if img.Height != nil {
		meta["height"] = fmt.Sprint(*img.Height)
	}

	// DeletedAt
	if img.DeletedAt != nil {
		meta["deleted_at"] = img.DeletedAt.UTC().Format(time.RFC3339Nano)
	} else {
		delete(meta, "deleted_at")
	}

	// UpdatedAt
	if img.UpdatedAt != nil {
		meta["updated_at"] = img.UpdatedAt.UTC().Format(time.RFC3339Nano)
	} else {
		meta["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return midom.ImageFile{}, err
	}

	return buildMessageImageFromAttrs(bucket, newAttrs), nil
}

// ReplaceAll:
// - 該当 messageID の prefix オブジェクトを全削除し、渡された一覧を Add して差し替える
func (r *MessageImageRepositoryGCS) ReplaceAll(ctx context.Context, messageID string, images []midom.ImageFile) ([]midom.ImageFile, error) {
	if r.Client == nil {
		return nil, errors.New("MessageImageRepositoryGCS: nil storage client")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, fmt.Errorf("messageImage: empty messageID")
	}

	if err := r.DeleteAll(ctx, messageID); err != nil {
		return nil, err
	}

	out := make([]midom.ImageFile, 0, len(images))
	for _, img := range images {
		img.MessageID = messageID
		saved, err := r.Add(ctx, img)
		if err != nil {
			return nil, err
		}
		out = append(out, saved)
	}
	return out, nil
}

// Update:
// - "<messageID>/<fileName>" オブジェクトのメタデータを patch で更新
func (r *MessageImageRepositoryGCS) Update(ctx context.Context, messageID, fileName string, patch midom.ImageFilePatch) (midom.ImageFile, error) {
	if r.Client == nil {
		return midom.ImageFile{}, errors.New("MessageImageRepositoryGCS: nil storage client")
	}

	messageID = strings.TrimSpace(messageID)
	fileName = strings.TrimSpace(fileName)
	if messageID == "" || fileName == "" {
		return midom.ImageFile{}, midom.ErrNotFound
	}

	bucket := r.bucket()
	objName := messageID + "/" + fileName
	obj := r.Client.Bucket(bucket).Object(objName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return midom.ImageFile{}, midom.ErrNotFound
		}
		return midom.ImageFile{}, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	setStr := func(key string, p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v != "" {
				meta[key] = v
			} else {
				delete(meta, key)
			}
		}
	}
	setInt := func(key string, p *int) {
		if p != nil {
			meta[key] = fmt.Sprint(*p)
		}
	}
	setInt64 := func(key string, p *int64) {
		if p != nil {
			meta[key] = fmt.Sprint(*p)
		}
	}

	// Updatable fields
	setStr("file_name", patch.FileName)
	setStr("file_url", patch.FileURL)
	setInt64("file_size", patch.FileSize)
	setStr("mime_type", patch.MimeType)
	setInt("width", patch.Width)
	setInt("height", patch.Height)

	if patch.MimeType != nil {
		mt := strings.TrimSpace(*patch.MimeType)
		if mt != "" {
			ua.ContentType = mt
		}
	}

	// DeletedAt
	if patch.DeletedAt != nil {
		t := patch.DeletedAt.UTC()
		meta["deleted_at"] = t.Format(time.RFC3339Nano)
	}

	// UpdatedAt 明示 or 自動
	if patch.UpdatedAt != nil {
		if !patch.UpdatedAt.IsZero() {
			meta["updated_at"] = patch.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
	} else if patch.FileName != nil || patch.FileURL != nil ||
		patch.FileSize != nil || patch.MimeType != nil ||
		patch.Width != nil || patch.Height != nil || patch.DeletedAt != nil {
		meta["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return midom.ImageFile{}, err
	}

	return buildMessageImageFromAttrs(bucket, newAttrs), nil
}

func (r *MessageImageRepositoryGCS) Delete(ctx context.Context, messageID, fileName string) error {
	if r.Client == nil {
		return errors.New("MessageImageRepositoryGCS: nil storage client")
	}
	messageID = strings.TrimSpace(messageID)
	fileName = strings.TrimSpace(fileName)
	if messageID == "" || fileName == "" {
		return midom.ErrNotFound
	}

	bucket := r.bucket()
	objName := messageID + "/" + fileName
	err := r.Client.Bucket(bucket).Object(objName).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return midom.ErrNotFound
	}
	return err
}

// DeleteAll:
// - messageID/ prefix の全オブジェクトを削除
func (r *MessageImageRepositoryGCS) DeleteAll(ctx context.Context, messageID string) error {
	if r.Client == nil {
		return errors.New("MessageImageRepositoryGCS: nil storage client")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil
	}

	bucket := r.bucket()
	prefix := messageID + "/"

	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	var errs []error
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if err := r.Client.Bucket(bucket).Object(attrs.Name).Delete(ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
			errs = append(errs, fmt.Errorf("%s: %w", attrs.Name, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}

// ========================================
// Helpers (GCS ObjectAttrs -> midom.ImageFile)
// ========================================

func buildMessageImageFromAttrs(bucket string, attrs *storage.ObjectAttrs) midom.ImageFile {
	name := strings.TrimSpace(attrs.Name)

	// MessageID:
	// 1. Metadata "message_id"
	// 2. "<messageID>/..." のパス先頭
	var messageID string
	if v, ok := attrs.Metadata["message_id"]; ok && strings.TrimSpace(v) != "" {
		messageID = strings.TrimSpace(v)
	} else if name != "" {
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			messageID = strings.TrimSpace(parts[0])
		}
	}

	// FileName: パス末尾
	fileName := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 && idx < len(name)-1 {
		fileName = name[idx+1:]
	}

	// 公開 URL（共通ユーティリティ利用）
	publicURL := gcscommon.GCSPublicURL(bucket, name, defaultMessageImageBucket)

	// file_size
	var fileSize int64
	if sz, ok := gcscommon.ParseInt64Meta(attrs.Metadata, "file_size"); ok {
		fileSize = sz
	} else if attrs.Size > 0 {
		fileSize = attrs.Size
	}

	// mime_type
	mimeType := strings.TrimSpace(attrs.ContentType)
	if mimeType == "" {
		if mt, ok := attrs.Metadata["mime_type"]; ok {
			mimeType = strings.TrimSpace(mt)
		}
	}

	// width / height
	var widthPtr, heightPtr *int
	if w, ok := gcscommon.ParseIntMeta(attrs.Metadata, "width"); ok {
		widthPtr = &w
	}
	if h, ok := gcscommon.ParseIntMeta(attrs.Metadata, "height"); ok {
		heightPtr = &h
	}

	// created / updated
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

	// deleted_at (metadata)
	var deletedAtPtr *time.Time
	if v := strings.TrimSpace(attrs.Metadata["deleted_at"]); v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			tu := t.UTC()
			deletedAtPtr = &tu
		}
	}

	return midom.ImageFile{
		MessageID: messageID,
		FileName:  fileName,
		FileURL:   publicURL,
		FileSize:  fileSize,
		MimeType:  mimeType,
		Width:     widthPtr,
		Height:    heightPtr,
		CreatedAt: createdAt,
		UpdatedAt: updatedAtPtr,
		DeletedAt: deletedAtPtr,
	}
}

// Filter: GCS ベース ImageFile に対して midom.Filter を適用
func matchMessageImageFilter(img midom.ImageFile, f midom.Filter) bool {
	if v := strings.TrimSpace(f.MessageID); v != "" && img.MessageID != v {
		return false
	}
	if v := strings.TrimSpace(f.FileNameLike); v != "" {
		lv := strings.ToLower(v)
		if !strings.Contains(strings.ToLower(img.FileName), lv) {
			return false
		}
	}
	if f.MimeType != nil {
		mt := strings.TrimSpace(*f.MimeType)
		if mt != "" && !strings.EqualFold(img.MimeType, mt) {
			return false
		}
	}
	if f.MinSize != nil && img.FileSize < *f.MinSize {
		return false
	}
	if f.MaxSize != nil && img.FileSize > *f.MaxSize {
		return false
	}

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

// ソート: DB版 buildMessageImageOrderBy と同等の意味を in-memory で再現
func applyMessageImageSort(items []midom.ImageFile, sort midom.Sort) {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}

	// デフォルト: created_at ASC, file_name ASC
	if col == "" {
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				swap := false
				if items[i].CreatedAt.After(items[j].CreatedAt) {
					swap = true
				} else if items[i].CreatedAt.Equal(items[j].CreatedAt) &&
					items[i].FileName > items[j].FileName {
					swap = true
				}
				if swap {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
		return
	}

	var less func(i, j int) bool

	switch col {
	case "createdat", "created_at":
		less = func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) }
	case "filename", "file_name":
		less = func(i, j int) bool { return items[i].FileName < items[j].FileName }
	case "filesize", "file_size":
		less = func(i, j int) bool { return items[i].FileSize < items[j].FileSize }
	case "updatedat", "updated_at":
		less = func(i, j int) bool {
			a, b := items[i].UpdatedAt, items[j].UpdatedAt
			switch {
			case a == nil && b == nil:
				return items[i].FileName < items[j].FileName
			case a == nil:
				return true
			case b == nil:
				return false
			default:
				return a.Before(*b)
			}
		}
	default:
		// 不明カラム -> デフォルトへフォールバック
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				swap := false
				if items[i].CreatedAt.After(items[j].CreatedAt) {
					swap = true
				} else if items[i].CreatedAt.Equal(items[j].CreatedAt) &&
					items[i].FileName > items[j].FileName {
					swap = true
				}
				if swap {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
		return
	}

	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			swap := less(j, i)
			if dir == "DESC" {
				swap = less(i, j)
			}
			if swap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// ========================================
// MessageImageStorageGCS (削除用アダプタ)
// ========================================

type MessageImageStorageGCS struct {
	Client          *storage.Client
	Bucket          string
	SignedURLExpiry time.Duration
}

// NewMessageImageStorageGCS creates a storage adapter with the provided client.
// If bucket is empty, it falls back to midom.DefaultBucket / defaultMessageImageBucket.
func NewMessageImageStorageGCS(client *storage.Client, bucket string) *MessageImageStorageGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		if strings.TrimSpace(midom.DefaultBucket) != "" {
			b = midom.DefaultBucket
		} else {
			b = defaultMessageImageBucket
		}
	}
	return &MessageImageStorageGCS{
		Client:          client,
		Bucket:          b,
		SignedURLExpiry: 15 * time.Minute,
	}
}

func (s *MessageImageStorageGCS) bucket() string {
	b := strings.TrimSpace(s.Bucket)
	if b == "" {
		if strings.TrimSpace(midom.DefaultBucket) != "" {
			return midom.DefaultBucket
		}
		return defaultMessageImageBucket
	}
	return b
}

// DeleteObject deletes a single GCS object.
func (s *MessageImageStorageGCS) DeleteObject(ctx context.Context, bucket, objectPath string) error {
	if s.Client == nil {
		return errors.New("MessageImageStorageGCS: nil storage client")
	}
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = s.bucket()
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if b == "" || obj == "" {
		return fmt.Errorf("invalid bucket/objectPath: bucket=%q, objectPath=%q", b, objectPath)
	}
	err := s.Client.Bucket(b).Object(obj).Delete(ctx)
	if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return err
	}
	return nil
}

// DeleteObjects deletes multiple GCS objects best-effort.
func (s *MessageImageStorageGCS) DeleteObjects(ctx context.Context, ops []midom.GCSDeleteOp) error {
	if s.Client == nil {
		return errors.New("MessageImageStorageGCS: nil storage client")
	}
	if len(ops) == 0 {
		return nil
	}
	var errs []error
	for _, op := range ops {
		b := strings.TrimSpace(op.Bucket)
		if b == "" {
			b = s.bucket()
		}
		obj := strings.TrimLeft(strings.TrimSpace(op.ObjectPath), "/")
		if obj == "" {
			continue
		}
		if err := s.Client.Bucket(b).Object(obj).Delete(ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
			errs = append(errs, fmt.Errorf("%s/%s: %w", b, obj, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}
