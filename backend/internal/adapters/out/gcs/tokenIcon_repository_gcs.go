// backend/internal/adapters/out/gcs/tokenIcon_repository_gcs.go
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
	tidom "narratives/internal/domain/tokenIcon"
)

// GCS-based implementation of TokenIcon repository.
// - Icons are stored as GCS objects.
// - One object = one TokenIcon.
// - Object name is used as ID (id == objectPath).
// - Metadata (file_name, size, etc.) is stored in GCS object metadata.
type TokenIconRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// Default bucket for token icons in this adapter layer.
const defaultTokenIconBucket = "narratives_development_token_icon"

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

// ========================
// RepositoryPort impl
// ========================

// GetByID:
// - id is either:
//   - a GCS URL (gs:// or https://storage.googleapis.com/...),
//   - or an object path within the default bucket.
func (r *TokenIconRepositoryGCS) GetByID(ctx context.Context, id string) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tidom.ErrNotFound
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
		return nil, tidom.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	icon := buildTokenIconFromAttrs(bucket, attrs)
	return &icon, nil
}

// List:
// - Scans all objects in bucket.
// - Maps them to TokenIcon.
// - Applies Filter / Sort / Page in memory.
func (r *TokenIconRepositoryGCS) List(
	ctx context.Context,
	filter tidom.Filter,
	sort tidom.Sort,
	page tidom.Page,
) (tidom.PageResult, error) {
	if r.Client == nil {
		return tidom.PageResult{}, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket := r.bucket()
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{})

	var all []tidom.TokenIcon
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tidom.PageResult{}, err
		}
		icon := buildTokenIconFromAttrs(bucket, attrs)
		if matchTokenIconFilter(icon, filter) {
			all = append(all, icon)
		}
	}

	applyTokenIconSort(all, sort)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
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

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return tidom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count: scan and count those that match filter.
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
		icon := buildTokenIconFromAttrs(bucket, attrs)
		if matchTokenIconFilter(icon, filter) {
			total++
		}
	}
	return total, nil
}

// Create:
// - Assumes GCS object already exists.
// - in.URL should be a GCS URL or object path.
// - Updates GCS metadata (file_name, size) and returns TokenIcon.
func (r *TokenIconRepositoryGCS) Create(ctx context.Context, in tidom.CreateTokenIconInput) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	url := strings.TrimSpace(in.URL)
	if url == "" {
		return nil, fmt.Errorf("TokenIconRepositoryGCS.Create: empty URL")
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
		return nil, fmt.Errorf("TokenIconRepositoryGCS.Create: empty objectPath")
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

	if fn := strings.TrimSpace(in.FileName); fn != "" {
		meta["file_name"] = fn
	}
	if in.Size > 0 {
		meta["size"] = fmt.Sprint(in.Size)
	}

	ua := storage.ObjectAttrsToUpdate{}
	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, newAttrs)
	return &ti, nil
}

// Update:
// - Updates GCS metadata for the icon identified by id.
func (r *TokenIconRepositoryGCS) Update(ctx context.Context, id string, in tidom.UpdateTokenIconInput) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tidom.ErrNotFound
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
		return nil, tidom.ErrNotFound
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

	if len(meta) == 0 {
		// nothing to update; return current
		ti := buildTokenIconFromAttrs(bucket, attrs)
		return &ti, nil
	}

	ua := storage.ObjectAttrsToUpdate{Metadata: meta}
	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, newAttrs)
	return &ti, nil
}

// Delete:
// - Deletes the underlying GCS object.
// - id can be a GCS URL or object path.
func (r *TokenIconRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tidom.ErrNotFound
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
		return tidom.ErrNotFound
	}

	err := r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return tidom.ErrNotFound
	}
	return err
}

// UploadIcon:
// - Uploads icon file to GCS and returns (publicURL, size).
// - Object name policy: "<unix_nano>_<fileName>".
func (r *TokenIconRepositoryGCS) UploadIcon(ctx context.Context, fileName, contentType string, body io.Reader) (string, int64, error) {
	if r.Client == nil {
		return "", 0, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "", 0, fmt.Errorf("UploadIcon: empty fileName")
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
		// best-effort cleanup
		_ = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
		return "", 0, err
	}

	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenIconBucket)
	return publicURL, n, nil
}

// GetTokenIconStats:
// - Scans all objects and computes stats.
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

		icon := buildTokenIconFromAttrs(bucket, attrs)
		if icon.Size <= 0 {
			continue
		}

		stats.Total++
		stats.TotalSize += icon.Size

		if largest == nil || icon.Size > largest.Size || (icon.Size == largest.Size && icon.ID < largest.ID) {
			tmp := icon
			largest = &tmp
		}
		if smallest == nil || icon.Size < smallest.Size || (icon.Size == smallest.Size && icon.ID < smallest.ID) {
			tmp := icon
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

// WithTx:
// - No real transactions in GCS; call fn directly.
// - Provided only for interface compatibility.
func (r *TokenIconRepositoryGCS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

// Reset:
// - Deletes all objects in the configured bucket.
// - Mainly for tests; use carefully.
func (r *TokenIconRepositoryGCS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("TokenIconRepositoryGCS: nil storage client")
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

// ========================
// Compatibility methods
// (kept for callers using old names; now backed by GCS)
// ========================

func (r *TokenIconRepositoryGCS) FetchAllTokenIcons(ctx context.Context) ([]*tidom.TokenIcon, error) {
	res, err := r.List(ctx, tidom.Filter{}, tidom.Sort{}, tidom.Page{
		Number:  1,
		PerPage: 10000, // large enough; adjust if needed
	})
	if err != nil {
		return nil, err
	}
	out := make([]*tidom.TokenIcon, 0, len(res.Items))
	for i := range res.Items {
		ti := res.Items[i]
		out = append(out, &ti)
	}
	return out, nil
}

func (r *TokenIconRepositoryGCS) FetchTokenIconByID(ctx context.Context, iconID string) (*tidom.TokenIcon, error) {
	return r.GetByID(ctx, iconID)
}

func (r *TokenIconRepositoryGCS) FetchTokenIconByBlueprintID(ctx context.Context, _ string) (*tidom.TokenIcon, error) {
	// Not supported (schema doesn't relate icons and blueprints directly).
	return nil, tidom.ErrNotFound
}

func (r *TokenIconRepositoryGCS) CreateTokenIcon(ctx context.Context, in tidom.CreateTokenIconInput) (*tidom.TokenIcon, error) {
	return r.Create(ctx, in)
}

func (r *TokenIconRepositoryGCS) UpdateTokenIcon(ctx context.Context, iconID string, updates tidom.UpdateTokenIconInput) (*tidom.TokenIcon, error) {
	return r.Update(ctx, iconID, updates)
}

// ========================
// Helpers
// ========================

func buildTokenIconFromAttrs(bucket string, attrs *storage.ObjectAttrs) tidom.TokenIcon {
	name := strings.TrimSpace(attrs.Name)
	meta := attrs.Metadata

	getMeta := func(key string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[key])
	}

	// ID: object name as-is
	id := name

	// URL: metadata.url or public URL
	url := getMeta("url")
	if url == "" {
		url = gcscommon.GCSPublicURL(bucket, name, defaultTokenIconBucket)
	}

	// FileName: metadata.file_name or last segment of object name
	fileName := getMeta("file_name")
	if fileName == "" {
		fileName = name
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 && idx < len(fileName)-1 {
			fileName = fileName[idx+1:]
		}
	}

	// Size: metadata.size or attrs.Size
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

// matchTokenIconFilter applies tidom.Filter conditions to a TokenIcon.
func matchTokenIconFilter(ti tidom.TokenIcon, f tidom.Filter) bool {
	// IDs
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

	// FileNameLike
	if v := strings.TrimSpace(f.FileNameLike); v != "" {
		lv := strings.ToLower(v)
		if !strings.Contains(strings.ToLower(ti.FileName), lv) {
			return false
		}
	}

	// Size range
	if f.SizeMin != nil && ti.Size < *f.SizeMin {
		return false
	}
	if f.SizeMax != nil && ti.Size > *f.SizeMax {
		return false
	}

	return true
}

// applyTokenIconSort: in-memory sorting analogous to buildTIOrderBy.
func applyTokenIconSort(items []tidom.TokenIcon, s tidom.Sort) {
	if len(items) <= 1 {
		return
	}

	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	var less func(i, j int) bool

	switch col {
	case "size":
		less = func(i, j int) bool { return items[i].Size < items[j].Size }
	case "filename", "file_name":
		less = func(i, j int) bool { return items[i].FileName < items[j].FileName }
	default:
		// default: id DESC (ASC comparator then invert if dir == DESC)
		less = func(i, j int) bool { return items[i].ID < items[j].ID }
		col = "id"
	}

	// simple O(n^2) sort; dataset is expected to be small
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
