// backend/internal/adapters/out/gcs/tokenContents_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	gcscommon "narratives/internal/adapters/out/gcs/common"
	tcdom "narratives/internal/domain/tokenContents"
)

// GCS-based implementation of token content repository.
// - Binary contents are stored in GCS.
// - Metadata is derived from GCS Object attributes/metadata.
// - One GCS object represents one TokenContent.
// - Object name is used as ID (id == objectPath).
type TokenContentsRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// Default bucket for token contents in this adapter layer.
const defaultTokenContentsBucket = "narratives_development_token_contents"

func NewTokenContentsRepositoryGCS(client *storage.Client, bucket string) *TokenContentsRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultTokenContentsBucket
	}
	return &TokenContentsRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *TokenContentsRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultTokenContentsBucket
	}
	return b
}

// ========================
// RepositoryPort impl
// ========================

// GetByID:
// - id is treated as either:
//   - GCS URL (gs:// or https://storage.googleapis.com/...), or
//   - object path within default bucket.
func (r *TokenContentsRepositoryGCS) GetByID(ctx context.Context, id string) (*tcdom.TokenContent, error) {
	if r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tcdom.ErrNotFound
	}

	var bucket, objectPath string
	if b, obj, ok := gcscommon.ParseGCSURL(id); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = r.bucket()
		objectPath = id
	}

	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objectPath == "" {
		return nil, tcdom.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, err
	}

	tc := buildTokenContentFromAttrs(bucket, attrs)
	return &tc, nil
}

// List:
// - scans all objects in the bucket
// - maps them to TokenContent
// - applies Filter / Sort / Page in memory
func (r *TokenContentsRepositoryGCS) List(
	ctx context.Context,
	filter tcdom.Filter,
	sort tcdom.Sort,
	page tcdom.Page,
) (tcdom.PageResult, error) {
	if r.Client == nil {
		return tcdom.PageResult{}, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	all := make([]tcdom.TokenContent, 0)

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tcdom.PageResult{}, err
		}
		tc := buildTokenContentFromAttrs(bucket, attrs)
		if matchTokenContentFilter(tc, filter) {
			all = append(all, tc)
		}
	}

	applyTokenContentSort(all, sort)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return tcdom.PageResult{
			Items:      []tcdom.TokenContent{},
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

	return tcdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count: same scan as List but only increments when filter matches.
func (r *TokenContentsRepositoryGCS) Count(ctx context.Context, filter tcdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("TokenContentsRepositoryGCS: nil storage client")
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
		tc := buildTokenContentFromAttrs(bucket, attrs)
		if matchTokenContentFilter(tc, filter) {
			total++
		}
	}
	return total, nil
}

// Create:
// - Assumes the object is already uploaded to GCS.
// - in.URL should point to that object (GCS URL or path).
// - Updates object metadata (name/type/size) and returns the TokenContent.
func (r *TokenContentsRepositoryGCS) Create(ctx context.Context, in tcdom.CreateTokenContentInput) (*tcdom.TokenContent, error) {
	if r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	url := strings.TrimSpace(in.URL)
	if url == "" {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: empty URL (expected GCS object URL or path)")
	}

	var bucket, objectPath string
	if b, obj, ok := gcscommon.ParseGCSURL(url); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = r.bucket()
		objectPath = url
	}

	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objectPath == "" {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: empty objectPath")
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, err
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if n := strings.TrimSpace(in.Name); n != "" {
		meta["name"] = n
	}
	if t := strings.TrimSpace(string(in.Type)); t != "" {
		meta["type"] = t
	}
	if in.Size > 0 {
		meta["size"] = fmt.Sprint(in.Size)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	tc := buildTokenContentFromAttrs(bucket, newAttrs)
	return &tc, nil
}

// Update:
// - Updates metadata on the corresponding GCS object for given id.
func (r *TokenContentsRepositoryGCS) Update(ctx context.Context, id string, in tcdom.UpdateTokenContentInput) (*tcdom.TokenContent, error) {
	if r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tcdom.ErrNotFound
	}

	var bucket, objectPath string
	if b, obj, ok := gcscommon.ParseGCSURL(id); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = r.bucket()
		objectPath = id
	}
	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objectPath == "" {
		return nil, tcdom.ErrNotFound
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, err
	}

	ua := storage.ObjectAttrsToUpdate{}
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
	setType := func(p *tcdom.ContentType) {
		if p == nil {
			return
		}
		v := strings.TrimSpace(string(*p))
		if v == "" {
			delete(meta, "type")
		} else {
			meta["type"] = v
		}
	}
	setInt64 := func(key string, p *int64) {
		if p == nil {
			return
		}
		meta[key] = fmt.Sprint(*p)
	}

	setStr("name", in.Name)
	setType(in.Type)
	setStr("url", in.URL)
	setInt64("size", in.Size)

	if len(meta) == 0 {
		// nothing to update -> return current
		tc := buildTokenContentFromAttrs(bucket, attrs)
		return &tc, nil
	}

	ua.Metadata = meta

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	tc := buildTokenContentFromAttrs(bucket, newAttrs)
	return &tc, nil
}

// Delete removes the underlying GCS object.
// id can be a GCS URL or object path.
func (r *TokenContentsRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tcdom.ErrNotFound
	}

	var bucket, objectPath string
	if b, obj, ok := gcscommon.ParseGCSURL(id); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = r.bucket()
		objectPath = id
	}
	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if objectPath == "" {
		return tcdom.ErrNotFound
	}

	err := r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return tcdom.ErrNotFound
	}
	return err
}

// UploadContent:
// - Uploads content to GCS and returns (publicURL, size).
// - ID generation policy: "<unix_nano>_<fileName>"
func (r *TokenContentsRepositoryGCS) UploadContent(ctx context.Context, fileName, contentType string, body io.Reader) (string, int64, error) {
	if r.Client == nil {
		return "", 0, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "", 0, fmt.Errorf("UploadContent: empty fileName")
	}

	bucket := r.bucket()
	objectPath := fmt.Sprintf("%d_%s", time.Now().UTC().UnixNano(), fileName)

	w := r.Client.Bucket(bucket).Object(objectPath).NewWriter(ctx)
	if ct := strings.TrimSpace(contentType); ct != "" {
		w.ContentType = ct
	}

	n, err := io.Copy(w, body)
	if cerr := w.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		_ = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx) // best-effort cleanup
		return "", 0, err
	}

	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenContentsBucket)
	return publicURL, n, nil
}

// GetStats:
// - Aggregates over all objects in the bucket.
func (r *TokenContentsRepositoryGCS) GetStats(ctx context.Context) (tcdom.TokenContentStats, error) {
	if r.Client == nil {
		return tcdom.TokenContentStats{}, errors.New("TokenContentsRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var stats tcdom.TokenContentStats

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tcdom.TokenContentStats{}, err
		}

		tc := buildTokenContentFromAttrs(bucket, attrs)

		// Deleted 無視（仕様により必要なら判定追加）
		stats.TotalCount++
		stats.TotalSize += tc.Size

		typ := strings.ToLower(strings.TrimSpace(string(tc.Type)))
		switch typ {
		case "image":
			stats.CountByType.Image++
		case "video":
			stats.CountByType.Video++
		case "pdf":
			stats.CountByType.PDF++
		case "document":
			stats.CountByType.Document++
		}
	}

	stats.TotalSizeFormatted = humanBytes(stats.TotalSize)
	return stats, nil
}

// Reset:
//   - Deletes all objects in the configured bucket.
//     (Use carefully; mainly for testing parity with PG implementation.)
func (r *TokenContentsRepositoryGCS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("TokenContentsRepositoryGCS: nil storage client")
	}
	bucket := r.bucket()

	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})
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

// WithTx:
// - GCS にはトランザクションがないため、そのまま fn を呼び出す。
// - usecase 側との互換性のためのダミー実装。
func (r *TokenContentsRepositoryGCS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

// ========================
// Helpers
// ========================

// buildTokenContentFromAttrs converts GCS ObjectAttrs → TokenContent.
func buildTokenContentFromAttrs(bucket string, attrs *storage.ObjectAttrs) tcdom.TokenContent {
	name := strings.TrimSpace(attrs.Name)

	meta := attrs.Metadata
	getMeta := func(key string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[key])
	}

	// ID: object path as-is
	id := name

	// Name: metadata.name or fallback to object base name
	tcName := getMeta("name")
	if tcName == "" {
		tcName = name
		if idx := strings.LastIndex(tcName, "/"); idx >= 0 && idx < len(tcName)-1 {
			tcName = tcName[idx+1:]
		}
	}

	// Type: metadata.type
	var ctype tcdom.ContentType
	if t := getMeta("type"); t != "" {
		ctype = tcdom.ContentType(t)
	}

	// URL: metadata.url or public URL from bucket/object
	url := getMeta("url")
	if url == "" {
		url = gcscommon.GCSPublicURL(bucket, name, defaultTokenContentsBucket)
	}

	// Size: metadata.size or attrs.Size
	var size int64
	if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
		size = sz
	} else if attrs.Size > 0 {
		size = attrs.Size
	}

	return tcdom.TokenContent{
		ID:   id,
		Name: tcName,
		Type: ctype,
		URL:  url,
		Size: size,
	}
}

// matchTokenContentFilter applies tcdom.Filter to a TokenContent.
func matchTokenContentFilter(tc tcdom.TokenContent, f tcdom.Filter) bool {
	// IDs
	if len(f.IDs) > 0 {
		ok := false
		for _, id := range f.IDs {
			if strings.TrimSpace(id) == tc.ID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Types
	if len(f.Types) > 0 {
		ok := false
		for _, t := range f.Types {
			if tc.Type == t {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// NameLike
	if v := strings.TrimSpace(f.NameLike); v != "" {
		lv := strings.ToLower(v)
		if !strings.Contains(strings.ToLower(tc.Name), lv) {
			return false
		}
	}

	// Size range
	if f.SizeMin != nil && tc.Size < *f.SizeMin {
		return false
	}
	if f.SizeMax != nil && tc.Size > *f.SizeMax {
		return false
	}

	return true
}

// applyTokenContentSort: in-memory sort equivalent of buildTCOrderBy.
func applyTokenContentSort(items []tcdom.TokenContent, s tcdom.Sort) {
	if len(items) <= 1 {
		return
	}

	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		// Default (PG互換): updated_at DESC, id DESC
		dir = "DESC"
		col = "id"
	}

	// SA4006 fix: declare and assign in switch
	var less func(i, j int) bool

	switch col {
	case "size":
		less = func(i, j int) bool { return items[i].Size < items[j].Size }
	case "name":
		less = func(i, j int) bool { return items[i].Name < items[j].Name }
	case "type":
		less = func(i, j int) bool { return string(items[i].Type) < string(items[j].Type) }
	case "id":
		fallthrough
	default:
		col = "id"
		less = func(i, j int) bool { return items[i].ID < items[j].ID }
	}

	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			swap := less(j, i) // ASC: swap if items[j] < items[i]
			if dir == "DESC" { // DESC: swap if items[i] < items[j]
				swap = less(i, j)
			}
			if swap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// humanBytes: same helper as PG版 for stats formatting.
func humanBytes(b int64) string {
	const unit = int64(1024)
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
