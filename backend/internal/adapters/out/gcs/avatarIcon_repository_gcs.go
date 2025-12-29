// backend/internal/adapters/out/gcs/avatarIcon_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	avicon "narratives/internal/domain/avatarIcon"
)

// AvatarIconRepositoryGCS implements avatarIcon.Repository backed by Google Cloud Storage.
//
// ✅ 追加方針（docId フォルダ作成）:
//   - GCS は「フォルダ/バケット」を階層として作れない（実体は object の prefix）ため、
//     "docId/.keep" のような 0byte(or小さな)オブジェクトを作成して Console 上で prefix を見せる。
//   - List/GetByAvatarID などで ".keep" はアイコンとして扱わないように除外する。
type AvatarIconRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// NewAvatarIconRepositoryGCS creates a new GCS-based avatar icon repository.
// bucket が空の場合はそのまま保持し、利用時にエラーとします。
// （必要なら設定側で必須指定してください）
func NewAvatarIconRepositoryGCS(client *storage.Client, bucket string) *AvatarIconRepositoryGCS {
	return &AvatarIconRepositoryGCS{
		Client: client,
		Bucket: strings.TrimSpace(bucket),
	}
}

// effectiveBucket resolves the bucket name or returns error if empty.
func (r *AvatarIconRepositoryGCS) effectiveBucket() (string, error) {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		// ドメイン側に DefaultBucket がある場合はそれをフォールバックにする
		if strings.TrimSpace(avicon.DefaultBucket) != "" {
			return strings.TrimSpace(avicon.DefaultBucket), nil
		}
		return "", errors.New("AvatarIconRepositoryGCS: bucket is empty")
	}
	return b, nil
}

// EnsureAvatarFolder ensures "<avatarID>/" prefix is visible on GCS console.
//
// GCS はフォルダを作れないため "avatarID/.keep" を作成する。
// 既に存在する場合は no-op 扱い。
func (r *AvatarIconRepositoryGCS) EnsureAvatarFolder(ctx context.Context, avatarID string) error {
	if r.Client == nil {
		return errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return err
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return errors.New("AvatarIconRepositoryGCS: avatarID is empty")
	}

	objName := strings.TrimLeft(avatarID, "/") + "/.keep"

	oh := r.Client.Bucket(bucketName).Object(objName).If(storage.Conditions{DoesNotExist: true})
	w := oh.NewWriter(ctx)
	w.ContentType = "text/plain; charset=utf-8"

	// 0byteでもOKだが、ツールやUI都合で「空」を嫌うことがあるので 1行だけ入れておく
	_, _ = w.Write([]byte("keep"))
	if err := w.Close(); err != nil {
		// すでに存在する場合は 412 になるので無視
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 412 {
			return nil
		}
		return err
	}
	return nil
}

// ✅ NEW: AvatarUsecase 側から呼ぶ「prefix 確保」用メソッド
// - interface(AvatarIconObjectStoragePort) に合わせて bucket/prefix を受け取る
// - prefix は "docId/" を想定（末尾 "/" はあってもなくてもOK）
func (r *AvatarIconRepositoryGCS) EnsurePrefix(ctx context.Context, bucket, prefix string) error {
	if r.Client == nil {
		return errors.New("AvatarIconRepositoryGCS: nil storage client")
	}

	bucketName := strings.TrimSpace(bucket)
	if bucketName == "" {
		// 明示 bucket が無い場合は repository の bucket を使う
		b, err := r.effectiveBucket()
		if err != nil {
			return err
		}
		bucketName = b
	}

	prefix = strings.TrimSpace(prefix)
	prefix = strings.TrimLeft(prefix, "/")
	if prefix == "" {
		return errors.New("AvatarIconRepositoryGCS: prefix is empty")
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objName := prefix + ".keep"

	oh := r.Client.Bucket(bucketName).Object(objName).If(storage.Conditions{DoesNotExist: true})
	w := oh.NewWriter(ctx)
	w.ContentType = "text/plain; charset=utf-8"

	_, _ = w.Write([]byte("keep"))
	if err := w.Close(); err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 412 {
			return nil
		}
		return err
	}
	return nil
}

func isFolderMarkerObject(name string) bool {
	n := strings.TrimSpace(name)
	if n == "" {
		return false
	}
	// "folder/" 形式や ".keep" を常に除外
	if strings.HasSuffix(n, "/") {
		return true
	}
	if strings.HasSuffix(n, "/.keep") {
		return true
	}
	// EnsurePrefix が ".keep" を作るので、これも除外
	if strings.HasSuffix(n, "/.keep") || strings.HasSuffix(n, ".keep") {
		return true
	}
	return false
}

// ==============================
// List (page-based)
// ==============================

func (r *AvatarIconRepositoryGCS) List(
	ctx context.Context,
	filter avicon.Filter,
	sortCfg avicon.Sort,
	page avicon.Page,
) (avicon.PageResult[avicon.AvatarIcon], error) {
	if r.Client == nil {
		return avicon.PageResult[avicon.AvatarIcon]{}, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return avicon.PageResult[avicon.AvatarIcon]{}, err
	}

	q := &storage.Query{}
	// AvatarID が指定されている場合は prefix を絞る
	if filter.AvatarID != nil {
		if v := strings.TrimSpace(*filter.AvatarID); v != "" {
			q.Prefix = v + "/"
		}
	}

	it := r.Client.Bucket(bucketName).Objects(ctx, q)

	var all []avicon.AvatarIcon
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return avicon.PageResult[avicon.AvatarIcon]{}, err
		}
		if isFolderMarkerObject(attrs.Name) {
			continue
		}
		icon := buildAvatarIconFromAttrs(bucketName, attrs)
		if matchAvatarIconFilter(icon, filter) {
			all = append(all, icon)
		}
	}

	sortAvatarIcons(all, sortCfg)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return avicon.PageResult[avicon.AvatarIcon]{
			Items:      []avicon.AvatarIcon{},
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
	totalPages := dbcommon.ComputeTotalPages(total, perPage)

	return avicon.PageResult[avicon.AvatarIcon]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ==============================
// ListByCursor (cursor-based)
// ==============================

func (r *AvatarIconRepositoryGCS) ListByCursor(
	ctx context.Context,
	filter avicon.Filter,
	_ avicon.Sort, // for now we fix ordering by ID ASC (object name)
	cpage avicon.CursorPage,
) (avicon.CursorPageResult[avicon.AvatarIcon], error) {
	if r.Client == nil {
		return avicon.CursorPageResult[avicon.AvatarIcon]{}, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return avicon.CursorPageResult[avicon.AvatarIcon]{}, err
	}

	q := &storage.Query{}
	if filter.AvatarID != nil {
		if v := strings.TrimSpace(*filter.AvatarID); v != "" {
			q.Prefix = v + "/"
		}
	}

	it := r.Client.Bucket(bucketName).Objects(ctx, q)

	var all []avicon.AvatarIcon
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return avicon.CursorPageResult[avicon.AvatarIcon]{}, err
		}
		if isFolderMarkerObject(attrs.Name) {
			continue
		}
		icon := buildAvatarIconFromAttrs(bucketName, attrs)
		if matchAvatarIconFilter(icon, filter) {
			all = append(all, icon)
		}
	}

	// 並び順は ID (object 名) 昇順
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	after := strings.TrimSpace(cpage.After)
	start := 0
	if after != "" {
		for i, v := range all {
			if v.ID > after {
				start = i
				break
			}
		}
	}

	if start > len(all) {
		start = len(all)
	}

	end := start + limit
	if end > len(all) {
		end = len(all)
	}

	items := all[start:end]

	var next *string
	if end < len(all) && len(items) > 0 {
		lastID := items[len(items)-1].ID
		next = &lastID
	}

	return avicon.CursorPageResult[avicon.AvatarIcon]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ==============================
// Getters
// ==============================

// GetByID fetches a single avatar icon by its ID (object name).
func (r *AvatarIconRepositoryGCS) GetByID(ctx context.Context, id string) (avicon.AvatarIcon, error) {
	if r.Client == nil {
		return avicon.AvatarIcon{}, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return avicon.AvatarIcon{}, err
	}

	id = strings.TrimSpace(id)
	if id == "" || isFolderMarkerObject(id) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucketName).Object(id).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, err
	}

	if isFolderMarkerObject(attrs.Name) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}
	return buildAvatarIconFromAttrs(bucketName, attrs), nil
}

// GetByAvatarID lists icons under "<avatarID>/" prefix.
func (r *AvatarIconRepositoryGCS) GetByAvatarID(ctx context.Context, avatarID string) ([]avicon.AvatarIcon, error) {
	if r.Client == nil {
		return nil, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return nil, err
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return []avicon.AvatarIcon{}, nil
	}

	q := &storage.Query{Prefix: avatarID + "/"}
	it := r.Client.Bucket(bucketName).Objects(ctx, q)

	var items []avicon.AvatarIcon
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if isFolderMarkerObject(attrs.Name) {
			continue
		}
		icon := buildAvatarIconFromAttrs(bucketName, attrs)
		if icon.AvatarID != nil && *icon.AvatarID == avatarID {
			items = append(items, icon)
		}
	}
	return items, nil
}

// ==============================
// Mutations
// ==============================

// Create "registers" an avatar icon based on an existing GCS object.
//
// 実ファイルのアップロードは別途行われている前提で、ここでは指定情報から
// 対応するオブジェクトが存在するか確認し、存在すれば AvatarIcon を返します。
func (r *AvatarIconRepositoryGCS) Create(ctx context.Context, a avicon.AvatarIcon) (avicon.AvatarIcon, error) {
	if r.Client == nil {
		return avicon.AvatarIcon{}, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return avicon.AvatarIcon{}, err
	}

	path, err := r.objectPathFromAvatarIcon(a)
	if err != nil {
		return avicon.AvatarIcon{}, err
	}
	if isFolderMarkerObject(path) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucketName).Object(path).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, err
	}
	if isFolderMarkerObject(attrs.Name) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	return buildAvatarIconFromAttrs(bucketName, attrs), nil
}

// Update currently does not mutate GCS objects; it just returns the latest state.
// （アイコンメタ情報は GCS オブジェクト名/サイズから導出するのみとする簡易実装）
func (r *AvatarIconRepositoryGCS) Update(ctx context.Context, id string, _ avicon.AvatarIconPatch) (avicon.AvatarIcon, error) {
	return r.GetByID(ctx, id)
}

// Delete removes the underlying GCS object by id (object name).
func (r *AvatarIconRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return err
	}

	id = strings.TrimSpace(id)
	if id == "" || isFolderMarkerObject(id) {
		return avicon.ErrNotFound
	}

	err = r.Client.Bucket(bucketName).Object(id).Delete(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return avicon.ErrNotFound
		}
		return err
	}
	return nil
}

// Count counts icons matching the filter by scanning objects.
func (r *AvatarIconRepositoryGCS) Count(ctx context.Context, filter avicon.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return 0, err
	}

	q := &storage.Query{}
	if filter.AvatarID != nil {
		if v := strings.TrimSpace(*filter.AvatarID); v != "" {
			q.Prefix = v + "/"
		}
	}

	it := r.Client.Bucket(bucketName).Objects(ctx, q)

	count := 0
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		if isFolderMarkerObject(attrs.Name) {
			continue
		}
		icon := buildAvatarIconFromAttrs(bucketName, attrs)
		if matchAvatarIconFilter(icon, filter) {
			count++
		}
	}
	return count, nil
}

// Save behaves as an upsert-like "refresh" based on existing GCS object.
// opts は現状未使用。
func (r *AvatarIconRepositoryGCS) Save(
	ctx context.Context,
	a avicon.AvatarIcon,
	_ *avicon.SaveOptions,
) (avicon.AvatarIcon, error) {
	if r.Client == nil {
		return avicon.AvatarIcon{}, errors.New("AvatarIconRepositoryGCS: nil storage client")
	}
	bucketName, err := r.effectiveBucket()
	if err != nil {
		return avicon.AvatarIcon{}, err
	}

	path, err := r.objectPathFromAvatarIcon(a)
	if err != nil {
		return avicon.AvatarIcon{}, err
	}
	if isFolderMarkerObject(path) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucketName).Object(path).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, err
	}
	if isFolderMarkerObject(attrs.Name) {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	return buildAvatarIconFromAttrs(bucketName, attrs), nil
}

// ==============================
// Helpers
// ==============================

// objectPathFromAvatarIcon 推論:
// - a.ID が非空ならそれを object path とみなす
// - そうでなければ AvatarID/FileName から "<avatarID>/<fileName>" を構築
func (r *AvatarIconRepositoryGCS) objectPathFromAvatarIcon(a avicon.AvatarIcon) (string, error) {
	if id := strings.TrimSpace(a.ID); id != "" {
		return strings.TrimLeft(id, "/"), nil
	}
	if a.AvatarID != nil && a.FileName != nil {
		aid := strings.TrimSpace(*a.AvatarID)
		fn := strings.TrimSpace(*a.FileName)
		if aid != "" && fn != "" {
			return fmt.Sprintf("%s/%s", aid, fn), nil
		}
	}
	return "", errors.New("avatar icon: missing id or (avatarID, fileName)")
}

// buildAvatarIconFromAttrs converts GCS ObjectAttrs into AvatarIcon.
func buildAvatarIconFromAttrs(bucket string, attrs *storage.ObjectAttrs) avicon.AvatarIcon {
	name := strings.TrimSpace(attrs.Name)

	var avatarIDPtr *string
	var fileNamePtr *string

	if name != "" {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			if v := strings.TrimSpace(parts[0]); v != "" {
				vCopy := v
				avatarIDPtr = &vCopy
			}
			if v := strings.TrimSpace(parts[1]); v != "" {
				vCopy := v
				fileNamePtr = &vCopy
			}
		} else {
			// パスに / が無い場合は fileName のみとみなす
			if v := strings.TrimSpace(name); v != "" {
				vCopy := v
				fileNamePtr = &vCopy
			}
		}
	}

	var sizePtr *int64
	if attrs.Size > 0 {
		s := attrs.Size
		sizePtr = &s
	}

	url := fmt.Sprintf("gs://%s/%s", bucket, attrs.Name)

	return avicon.AvatarIcon{
		ID:       name,
		AvatarID: avatarIDPtr,
		URL:      url,
		FileName: fileNamePtr,
		Size:     sizePtr,
	}
}

// matchAvatarIconFilter applies avicon.Filter in-memory.
func matchAvatarIconFilter(a avicon.AvatarIcon, f avicon.Filter) bool {
	// SearchQuery: ID / URL / FileName 部分一致
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		fn := ""
		if a.FileName != nil {
			fn = *a.FileName
		}
		haystack := strings.ToLower(a.ID + " " + a.URL + " " + fn)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// AvatarID
	if f.AvatarID != nil {
		want := strings.TrimSpace(*f.AvatarID)
		got := ""
		if a.AvatarID != nil {
			got = strings.TrimSpace(*a.AvatarID)
		}
		if want != "" && want != got {
			return false
		}
	}

	// HasAvatarID
	if f.HasAvatarID != nil {
		has := a.AvatarID != nil && strings.TrimSpace(*a.AvatarID) != ""
		if *f.HasAvatarID && !has {
			return false
		}
		if !*f.HasAvatarID && has {
			return false
		}
	}

	// Size range
	if f.SizeMin != nil {
		if a.Size == nil || *a.Size < *f.SizeMin {
			return false
		}
	}
	if f.SizeMax != nil {
		if a.Size == nil || *a.Size > *f.SizeMax {
			return false
		}
	}

	return true
}

// sortAvatarIcons orders icons based on avicon.Sort (id/size/file_name/url).
func sortAvatarIcons(items []avicon.AvatarIcon, sortCfg avicon.Sort) {
	if len(items) == 0 {
		return
	}

	col := strings.ToLower(string(sortCfg.Column))
	dir := strings.ToUpper(string(sortCfg.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	asc := dir == "ASC"

	less := func(i, j int) bool {
		a, b := items[i], items[j]

		switch col {
		case "id":
			if asc {
				return a.ID < b.ID
			}
			return a.ID > b.ID

		case "size":
			var sa, sb int64
			if a.Size != nil {
				sa = *a.Size
			}
			if b.Size != nil {
				sb = *b.Size
			}
			if sa == sb {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return sa < sb
			}
			return sa > sb

		case "filename", "file_name":
			var fa, fb string
			if a.FileName != nil {
				fa = *a.FileName
			}
			if b.FileName != nil {
				fb = *b.FileName
			}
			if fa == fb {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return fa < fb
			}
			return fa > fb

		case "url":
			if a.URL == b.URL {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.URL < b.URL
			}
			return a.URL > b.URL

		default:
			// デフォルトは ID ASC
			if asc {
				return a.ID < b.ID
			}
			return a.ID > b.ID
		}
	}

	sort.SliceStable(items, less)
}
