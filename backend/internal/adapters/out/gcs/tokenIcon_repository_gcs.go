// backend/internal/adapters/out/gcs/tokenIcon_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	gcscommon "narratives/internal/adapters/out/gcs/common"
	tidom "narratives/internal/domain/tokenIcon"
)

// TokenIconRepositoryGCS
//   - Token Icon の実体（バイナリ）を GCS に保存するためのアダプタ。
//   - tokenIcon.RepositoryPort を満たすため、GCS 走査で GetByID/List/Count/Stats/Reset も実装します。
//     ※ 大量オブジェクトになるなら、将来的に別永続層へ分離推奨。
type TokenIconRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// Default bucket for token icons (public).
// env TOKEN_ICON_BUCKET が空のときのフォールバック。
const defaultTokenIconBucket = "narratives-development_token_icon"

func NewTokenIconRepositoryGCS(client *storage.Client, bucket string) *TokenIconRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultTokenIconBucket
	}
	return &TokenIconRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *TokenIconRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultTokenIconBucket
	}
	return b
}

// ============================================================
// GetByID
// ============================================================

// GetByID:
// - id is either:
//   - a GCS URL (gs:// or https://storage.googleapis.com/...),
//   - or an object path within the default bucket.
func (r *TokenIconRepositoryGCS) GetByID(ctx context.Context, id string) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return nil, err
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, attrs)
	return &ti, nil
}

// ============================================================
// List / Count
// ============================================================

func (r *TokenIconRepositoryGCS) List(
	ctx context.Context,
	filter tidom.Filter,
	sortSpec tidom.Sort,
	page tidom.Page,
) (tidom.PageResult, error) {
	if r.Client == nil {
		return tidom.PageResult{}, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	all := make([]tidom.TokenIcon, 0, 64)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tidom.PageResult{}, err
		}

		ti := buildTokenIconFromAttrs(bucket, attrs)
		if matchTokenIconFilter(ti, filter) {
			all = append(all, ti)
		}
	}

	applyTokenIconSort(all, sortSpec)

	pageNum := page.Number
	if pageNum <= 0 {
		pageNum = 1
	}
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	total := len(all)
	if total == 0 {
		return tidom.PageResult{
			Items:      []tidom.TokenIcon{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	totalPages := (total + perPage - 1) / perPage
	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	return tidom.PageResult{
		Items:      all[offset:end],
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenIconRepositoryGCS) Count(ctx context.Context, filter tidom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("TokenIconRepositoryGCS: nil storage client")
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

		ti := buildTokenIconFromAttrs(bucket, attrs)
		if matchTokenIconFilter(ti, filter) {
			total++
		}
	}

	return total, nil
}

// ============================================================
// Upload
// ============================================================

// UploadIcon uploads icon file to GCS and returns (publicURL, size).
// Object name policy: "<unix_nano>_<fileName>" in the configured bucket.
func (r *TokenIconRepositoryGCS) UploadIcon(
	ctx context.Context,
	fileName string,
	contentType string,
	body io.Reader,
) (string, int64, error) {
	if r.Client == nil {
		return "", 0, errors.New("TokenIconRepositoryGCS: nil storage client")
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		return "", 0, fmt.Errorf("UploadIcon: empty fileName")
	}

	objectPath := fmt.Sprintf("%d_%s", time.Now().UTC().UnixNano(), fileName)
	return r.UploadIconTo(ctx, objectPath, contentType, body)
}

// UploadIconTo uploads icon bytes to the specified objectPath in the configured bucket.
func (r *TokenIconRepositoryGCS) UploadIconTo(
	ctx context.Context,
	objectPath string,
	contentType string,
	body io.Reader,
) (string, int64, error) {
	if r.Client == nil {
		return "", 0, errors.New("TokenIconRepositoryGCS: nil storage client")
	}
	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objectPath == "" {
		return "", 0, fmt.Errorf("UploadIconTo: empty objectPath")
	}

	bucket := r.bucket()

	w := r.Client.Bucket(bucket).Object(objectPath).NewWriter(ctx)
	if ct := strings.TrimSpace(contentType); ct != "" {
		w.ContentType = ct
	}

	n, err := io.Copy(w, body)
	if cerr := w.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		_ = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
		return "", 0, err
	}

	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenIconBucket)
	return publicURL, n, nil
}

// ============================================================
// Create / Update / Delete
// ============================================================

// Create:
// - 「Upload 済みの GCS オブジェクト」に対してメタデータ（file_name/size/url）を付与します。
// - in.URL は publicURL / gs:// / https://storage.googleapis.com/... / objectPath のいずれでもOK。
// - 戻り値 TokenIcon.ID は objectPath（= newAttrs.Name）です。
func (r *TokenIconRepositoryGCS) Create(
	ctx context.Context,
	in tidom.CreateTokenIconInput,
) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return nil, fmt.Errorf("TokenIconRepositoryGCS.Create: empty URL")
	}

	bucket, objectPath, err := resolveBucketObject(raw, r.bucket())
	if err != nil {
		return nil, err
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	// merge metadata
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if fn := strings.TrimSpace(in.FileName); fn != "" {
		meta["file_name"] = fn
	}
	if in.Size > 0 {
		meta["size"] = fmt.Sprint(in.Size)
	}

	// URL は “公開 URL” を優先して入れる（Create入力がgs://等でも閲覧側はこれを使える）
	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenIconBucket)
	meta["url"] = publicURL

	newAttrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return nil, err
	}

	ti := tidom.TokenIcon{
		ID:       newAttrs.Name,
		URL:      strings.TrimSpace(meta["url"]),
		FileName: strings.TrimSpace(meta["file_name"]),
		Size:     newAttrs.Size,
	}
	if ti.URL == "" {
		ti.URL = publicURL
	}
	if ti.FileName == "" {
		ti.FileName = lastSegment(newAttrs.Name)
	}
	if v := strings.TrimSpace(meta["size"]); v != "" {
		if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
			ti.Size = sz
		}
	}

	return &ti, nil
}

// Update updates GCS object metadata for the icon identified by id.
func (r *TokenIconRepositoryGCS) Update(
	ctx context.Context,
	id string,
	in tidom.UpdateTokenIconInput,
) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return nil, err
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	setStr := func(key string, p *string) {
		if p == nil {
			return
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			delete(meta, key)
		} else {
			meta[key] = v
		}
	}
	setInt64 := func(key string, p *int64) {
		if p == nil {
			return
		}
		meta[key] = fmt.Sprint(*p)
	}

	setStr("file_name", in.FileName)
	setStr("url", in.URL)
	setInt64("size", in.Size)

	newAttrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, newAttrs)
	return &ti, nil
}

// Delete deletes the underlying GCS object.
func (r *TokenIconRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return err
	}

	err = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return tidom.ErrNotFound
	}
	return err
}

// ============================================================
// Stats / Tx / Reset (RepositoryPort compatibility)
// ============================================================

func (r *TokenIconRepositoryGCS) GetTokenIconStats(ctx context.Context) (tidom.TokenIconStats, error) {
	if r.Client == nil {
		return tidom.TokenIconStats{}, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var stats tidom.TokenIconStats
	var largest *tidom.TokenIcon
	var smallest *tidom.TokenIcon

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tidom.TokenIconStats{}, err
		}

		ti := buildTokenIconFromAttrs(bucket, attrs)
		if ti.Size <= 0 {
			continue
		}

		stats.Total++
		stats.TotalSize += ti.Size

		if largest == nil || ti.Size > largest.Size || (ti.Size == largest.Size && ti.ID < largest.ID) {
			tmp := ti
			largest = &tmp
		}
		if smallest == nil || ti.Size < smallest.Size || (ti.Size == smallest.Size && ti.ID < smallest.ID) {
			tmp := ti
			smallest = &tmp
		}
	}

	if stats.Total > 0 {
		stats.AverageSize = float64(stats.TotalSize) / float64(stats.Total)
	}
	if largest != nil {
		stats.LargestIcon = &struct {
			ID       string
			FileName string
			Size     int64
		}{ID: largest.ID, FileName: largest.FileName, Size: largest.Size}
	}
	if smallest != nil {
		stats.SmallestIcon = &struct {
			ID       string
			FileName string
			Size     int64
		}{ID: smallest.ID, FileName: smallest.FileName, Size: smallest.Size}
	}

	return stats, nil
}

// WithTx: GCS にトランザクションは無いので、そのまま実行
func (r *TokenIconRepositoryGCS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

// Reset: 開発/テスト用。バケット内の全オブジェクトを削除。
func (r *TokenIconRepositoryGCS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		_ = r.Client.Bucket(bucket).Object(attrs.Name).Delete(ctx)
	}
	return nil
}

// ============================================================
// Helpers
// ============================================================

func buildTokenIconFromAttrs(bucket string, attrs *storage.ObjectAttrs) tidom.TokenIcon {
	name := strings.TrimSpace(attrs.Name)
	meta := attrs.Metadata

	getMeta := func(key string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[key])
	}

	id := name

	url := getMeta("url")
	if url == "" {
		url = gcscommon.GCSPublicURL(bucket, name, defaultTokenIconBucket)
	}

	fileName := getMeta("file_name")
	if fileName == "" {
		fileName = lastSegment(name)
	}

	size := attrs.Size
	if meta != nil {
		if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
			size = sz
		}
	}

	return tidom.TokenIcon{
		ID:       id,
		URL:      url,
		FileName: fileName,
		Size:     size,
	}
}

func matchTokenIconFilter(ti tidom.TokenIcon, f tidom.Filter) bool {
	if len(f.IDs) > 0 {
		ok := false
		for _, id := range f.IDs {
			if strings.TrimSpace(id) == ti.ID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if v := strings.TrimSpace(f.FileNameLike); v != "" {
		lv := strings.ToLower(v)
		if !strings.Contains(strings.ToLower(ti.FileName), lv) {
			return false
		}
	}

	if f.SizeMin != nil && ti.Size < *f.SizeMin {
		return false
	}
	if f.SizeMax != nil && ti.Size > *f.SizeMax {
		return false
	}

	return true
}

func applyTokenIconSort(items []tidom.TokenIcon, s tidom.Sort) {
	if len(items) <= 1 {
		return
	}

	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToLower(strings.TrimSpace(string(s.Order)))
	if dir != "asc" && dir != "desc" {
		dir = "desc"
	}

	less := func(i, j int) bool { return items[i].ID < items[j].ID }

	switch col {
	case "size":
		less = func(i, j int) bool { return items[i].Size < items[j].Size }
	case "filename", "file_name", "filename_like", "fileName":
		less = func(i, j int) bool { return items[i].FileName < items[j].FileName }
	default:
		less = func(i, j int) bool { return items[i].ID < items[j].ID }
	}

	sort.Slice(items, func(i, j int) bool {
		if dir == "asc" {
			return less(i, j)
		}
		return less(j, i)
	})
}

func resolveBucketObject(id string, fallbackBucket string) (bucket string, objectPath string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", tidom.ErrNotFound
	}

	if b, obj, ok := gcscommon.ParseGCSURL(id); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = strings.TrimSpace(fallbackBucket)
		objectPath = id
	}

	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if bucket == "" || objectPath == "" {
		return "", "", tidom.ErrNotFound
	}
	return bucket, objectPath, nil
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func lastSegment(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if i := strings.LastIndex(p, "/"); i >= 0 && i < len(p)-1 {
		return p[i+1:]
	}
	return p
}
