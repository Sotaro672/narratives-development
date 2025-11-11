// backend\internal\adapters\out\firestore\listImage_repository_pg.go
package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	listimagedom "narratives/internal/domain/listImage"
)

// ListImageRepositoryPG is the Postgres adapter for list_images.
// This struct MUST be distinct from ListRepositoryPG (lists table).
type ListImageRepositoryPG struct {
	DB *sql.DB
}

func NewListImageRepositoryPG(db *sql.DB) *ListImageRepositoryPG {
	return &ListImageRepositoryPG{DB: db}
}

// ─────────────────────────────────
// Queries
// ─────────────────────────────────

// GetByID satisfies usecase.ListImageByIDReader.
// (usecase側インターフェース名は GetByID なので合わせる)
func (r *ListImageRepositoryPG) GetByID(ctx context.Context, imageID string) (listimagedom.ListImage, error) {
	const q = `
SELECT
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
FROM list_images
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(imageID))
	img, err := scanListImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}
	return img, nil
}

// ListByListID satisfies usecase.ListImageReader.
// (usecase側インターフェース名は ListByListID なので合わせる)
func (r *ListImageRepositoryPG) ListByListID(ctx context.Context, listID string) ([]listimagedom.ListImage, error) {
	const q = `
SELECT
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
FROM list_images
WHERE list_id = $1
ORDER BY display_order ASC, created_at ASC, id ASC
`
	rows, err := r.DB.QueryContext(ctx, q, strings.TrimSpace(listID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []listimagedom.ListImage
	for rows.Next() {
		img, err := scanListImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ListImageRepositoryPG) Exists(ctx context.Context, imageID string) (bool, error) {
	const q = `SELECT 1 FROM list_images WHERE id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(imageID)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ListImageRepositoryPG) Count(ctx context.Context, filter listimagedom.Filter) (int, error) {
	where, args := buildListImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM list_images li `+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ListImageRepositoryPG) List(
	ctx context.Context,
	filter listimagedom.Filter,
	sort listimagedom.Sort,
	page listimagedom.Page,
) (listimagedom.PageResult[listimagedom.ListImage], error) {

	where, args := buildListImageWhere(filter)

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildListImageOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY li.display_order ASC, li.created_at ASC, li.id ASC"
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM list_images li %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return listimagedom.PageResult[listimagedom.ListImage]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
FROM list_images li
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return listimagedom.PageResult[listimagedom.ListImage]{}, err
	}
	defer rows.Close()

	var items []listimagedom.ListImage
	for rows.Next() {
		img, err := scanListImage(rows)
		if err != nil {
			return listimagedom.PageResult[listimagedom.ListImage]{}, err
		}
		items = append(items, img)
	}
	if err := rows.Err(); err != nil {
		return listimagedom.PageResult[listimagedom.ListImage]{}, err
	}

	totalPages := (total + perPage - 1) / perPage

	return listimagedom.PageResult[listimagedom.ListImage]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *ListImageRepositoryPG) ListByCursor(
	ctx context.Context,
	filter listimagedom.Filter,
	_ listimagedom.Sort,
	cpage listimagedom.CursorPage,
) (listimagedom.CursorPageResult[listimagedom.ListImage], error) {

	where, args := buildListImageWhere(filter)

	// simple cursor on id ASC
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("li.id > $%d", len(args)+1))
		args = append(args, after)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := fmt.Sprintf(`
SELECT
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
FROM list_images li
%s
ORDER BY li.id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return listimagedom.CursorPageResult[listimagedom.ListImage]{}, err
	}
	defer rows.Close()

	var items []listimagedom.ListImage
	var lastID string
	for rows.Next() {
		img, err := scanListImage(rows)
		if err != nil {
			return listimagedom.CursorPageResult[listimagedom.ListImage]{}, err
		}
		items = append(items, img)
		lastID = img.ID
	}
	if err := rows.Err(); err != nil {
		return listimagedom.CursorPageResult[listimagedom.ListImage]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
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

// SaveFromBucketObject satisfies usecase.ListImageObjectSaver.
// GCS等からオブジェクトを登録する用途。
func (r *ListImageRepositoryPG) SaveFromBucketObject(
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

	// URL の生成ポリシー: とりあえず "gs://bucket/objectPath" 的な形にしておく。
	// あとで公開URLにするならここを差し替えればいい。
	urlVal := buildGCSURL(bucket, objectPath)

	const q = `
INSERT INTO list_images (
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,$11,$12
)
ON CONFLICT (id) DO UPDATE SET
  list_id       = EXCLUDED.list_id,
  url           = EXCLUDED.url,
  file_name     = EXCLUDED.file_name,
  size          = EXCLUDED.size,
  display_order = EXCLUDED.display_order,
  created_at    = LEAST(list_images.created_at, EXCLUDED.created_at),
  created_by    = LEAST(list_images.created_by, EXCLUDED.created_by),
  updated_at    = COALESCE(EXCLUDED.updated_at, list_images.updated_at),
  updated_by    = EXCLUDED.updated_by,
  deleted_at    = EXCLUDED.deleted_at,
  deleted_by    = EXCLUDED.deleted_by
RETURNING
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
`

	nowUTC := time.Now().UTC()

	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(id),
		strings.TrimSpace(listID),
		strings.TrimSpace(urlVal),
		strings.TrimSpace(objectPath), // file_nameはobjectPathをそのまま使う想定
		size,
		displayOrder,
		createdAt.UTC(),
		strings.TrimSpace(createdBy),
		nowUTC,
		strings.TrimSpace(createdBy),
		nil,
		nil,
	)

	out, err := scanListImage(row)
	if err != nil {
		return listimagedom.ListImage{}, err
	}
	return out, nil
}

func (r *ListImageRepositoryPG) Create(ctx context.Context, img listimagedom.ListImage) (listimagedom.ListImage, error) {
	const q = `
INSERT INTO list_images (
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,$11,$12
)
RETURNING
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(img.ID),
		strings.TrimSpace(img.ListID),
		strings.TrimSpace(img.URL),
		strings.TrimSpace(img.FileName),
		img.Size,
		img.DisplayOrder,
		img.CreatedAt.UTC(),
		strings.TrimSpace(img.CreatedBy),
		dbcommon.ToDBTime(img.UpdatedAt),
		dbcommon.ToDBText(img.UpdatedBy),
		dbcommon.ToDBTime(img.DeletedAt),
		dbcommon.ToDBText(img.DeletedBy),
	)
	out, err := scanListImage(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return listimagedom.ListImage{}, listimagedom.ErrConflict
		}
		return listimagedom.ListImage{}, err
	}
	return out, nil
}

func (r *ListImageRepositoryPG) Update(ctx context.Context, imageID string, patch listimagedom.ListImagePatch) (listimagedom.ListImage, error) {
	sets := []string{}
	args := []any{}
	i := 1

	setStr := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(*p))
			i++
		}
	}
	setInt64 := func(col string, p *int64) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, *p)
			i++
		}
	}
	setInt := func(col string, p *int) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, *p)
			i++
		}
	}

	setStr("url", patch.URL)
	setStr("file_name", patch.FileName)
	setInt64("size", patch.Size)
	setInt("display_order", patch.DisplayOrder)

	// optional audit fields
	if patch.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.UpdatedBy))
		i++
	}
	if patch.DeletedAt != nil {
		sets = append(sets, fmt.Sprintf("deleted_at = $%d", i))
		args = append(args, dbcommon.ToDBTime(patch.DeletedAt))
		i++
	}
	if patch.DeletedBy != nil {
		sets = append(sets, fmt.Sprintf("deleted_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.DeletedBy))
		i++
	}

	// updated_at explicit or NOW() if any field changed
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	if len(sets) == 0 {
		// nothing to change → return current row
		current, err := r.GetByID(ctx, imageID)
		if err != nil {
			return listimagedom.ListImage{}, err
		}
		return current, nil
	}

	args = append(args, strings.TrimSpace(imageID))

	q := fmt.Sprintf(`
UPDATE list_images
SET %s
WHERE id = $%d
RETURNING
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanListImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, err
	}
	return out, nil
}

func (r *ListImageRepositoryPG) Save(ctx context.Context, img listimagedom.ListImage, _ *listimagedom.SaveOptions) (listimagedom.ListImage, error) {
	const q = `
INSERT INTO list_images (
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,$11,$12
)
ON CONFLICT (id) DO UPDATE SET
  list_id       = EXCLUDED.list_id,
  url           = EXCLUDED.url,
  file_name     = EXCLUDED.file_name,
  size          = EXCLUDED.size,
  display_order = EXCLUDED.display_order,
  created_at    = LEAST(list_images.created_at, EXCLUDED.created_at),
  created_by    = LEAST(list_images.created_by, EXCLUDED.created_by),
  updated_at    = COALESCE(EXCLUDED.updated_at, list_images.updated_at),
  updated_by    = EXCLUDED.updated_by,
  deleted_at    = EXCLUDED.deleted_at,
  deleted_by    = EXCLUDED.deleted_by
RETURNING
  id, list_id, url, file_name, size, display_order,
  created_at, created_by, updated_at, updated_by,
  deleted_at, deleted_by
`

	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(img.ID),
		strings.TrimSpace(img.ListID),
		strings.TrimSpace(img.URL),
		strings.TrimSpace(img.FileName),
		img.Size,
		img.DisplayOrder,
		img.CreatedAt.UTC(),
		strings.TrimSpace(img.CreatedBy),
		dbcommon.ToDBTime(img.UpdatedAt),
		dbcommon.ToDBText(img.UpdatedBy),
		dbcommon.ToDBTime(img.DeletedAt),
		dbcommon.ToDBText(img.DeletedBy),
	)

	out, err := scanListImage(row)
	if err != nil {
		return listimagedom.ListImage{}, err
	}
	return out, nil
}

// Upload is not implemented at DB layer (actual blob upload should happen in object storage adapter).
func (r *ListImageRepositoryPG) Upload(ctx context.Context, _ listimagedom.UploadImageInput) (*listimagedom.ListImage, error) {
	return nil, listimagedom.ErrUploadFailed
}

func (r *ListImageRepositoryPG) Delete(ctx context.Context, imageID string) error {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM list_images WHERE id = $1`,
		strings.TrimSpace(imageID),
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return listimagedom.ErrNotFound
	}
	return nil
}

// ─────────────────────────────────
// Helpers
// ─────────────────────────────────

func scanListImage(s dbcommon.RowScanner) (listimagedom.ListImage, error) {
	var (
		idNS, listIDNS, urlNS, fileNameNS     sql.NullString
		createdByNS, updatedByNS, deletedByNS sql.NullString
		size                                  int64
		displayOrder                          int
		createdAt                             time.Time
		updatedAtNS                           sql.NullTime
		deletedAtNS                           sql.NullTime
	)

	if err := s.Scan(
		&idNS, &listIDNS, &urlNS, &fileNameNS, &size, &displayOrder,
		&createdAt, &createdByNS, &updatedAtNS, &updatedByNS,
		&deletedAtNS, &deletedByNS,
	); err != nil {
		return listimagedom.ListImage{}, err
	}

	toPtrStr := func(ns sql.NullString) *string {
		if ns.Valid {
			v := strings.TrimSpace(ns.String)
			if v != "" {
				return &v
			}
		}
		return nil
	}
	toPtrTime := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}

	return listimagedom.ListImage{
		ID:           strings.TrimSpace(idNS.String),
		ListID:       strings.TrimSpace(listIDNS.String),
		URL:          strings.TrimSpace(urlNS.String),
		FileName:     strings.TrimSpace(fileNameNS.String),
		Size:         size,
		DisplayOrder: displayOrder,
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    strings.TrimSpace(createdByNS.String),
		UpdatedAt:    toPtrTime(updatedAtNS),
		UpdatedBy:    toPtrStr(updatedByNS),
		DeletedAt:    toPtrTime(deletedAtNS),
		DeletedBy:    toPtrStr(deletedByNS),
	}, nil
}

func buildListImageWhere(f listimagedom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// free text search
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where,
			fmt.Sprintf("(li.file_name ILIKE $%d OR li.url ILIKE $%d)",
				len(args)+1, len(args)+1,
			),
		)
		args = append(args, "%"+sq+"%")
	}

	// IDs IN (...)
	if len(f.IDs) > 0 {
		ph := []string{}
		for _, v := range f.IDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "li.id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// listID equals / IN
	if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
		where = append(where, fmt.Sprintf("li.list_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.ListID))
	}
	if len(f.ListIDs) > 0 {
		ph := []string{}
		for _, v := range f.ListIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "li.list_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// fileNameLike
	if f.FileNameLike != nil && strings.TrimSpace(*f.FileNameLike) != "" {
		where = append(where, fmt.Sprintf("li.file_name ILIKE $%d", len(args)+1))
		args = append(args, "%"+strings.TrimSpace(*f.FileNameLike)+"%")
	}

	// size range
	if f.MinSize != nil {
		where = append(where, fmt.Sprintf("li.size >= $%d", len(args)+1))
		args = append(args, *f.MinSize)
	}
	if f.MaxSize != nil {
		where = append(where, fmt.Sprintf("li.size <= $%d", len(args)+1))
		args = append(args, *f.MaxSize)
	}

	// display_order range
	if f.MinDisplayOrd != nil {
		where = append(where, fmt.Sprintf("li.display_order >= $%d", len(args)+1))
		args = append(args, *f.MinDisplayOrd)
	}
	if f.MaxDisplayOrd != nil {
		where = append(where, fmt.Sprintf("li.display_order <= $%d", len(args)+1))
		args = append(args, *f.MaxDisplayOrd)
	}

	// audit by user
	if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
		where = append(where, fmt.Sprintf("li.created_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.CreatedBy))
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		where = append(where, fmt.Sprintf("li.updated_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.UpdatedBy))
	}
	if f.DeletedBy != nil && strings.TrimSpace(*f.DeletedBy) != "" {
		where = append(where, fmt.Sprintf("li.deleted_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.DeletedBy))
	}

	// date ranges
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("li.created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("li.created_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where,
			fmt.Sprintf("(li.updated_at IS NOT NULL AND li.updated_at >= $%d)", len(args)+1),
		)
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where,
			fmt.Sprintf("(li.updated_at IS NOT NULL AND li.updated_at < $%d)", len(args)+1),
		)
		args = append(args, f.UpdatedTo.UTC())
	}
	if f.DeletedFrom != nil {
		where = append(where,
			fmt.Sprintf("(li.deleted_at IS NOT NULL AND li.deleted_at >= $%d)", len(args)+1),
		)
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where,
			fmt.Sprintf("(li.deleted_at IS NOT NULL AND li.deleted_at < $%d)", len(args)+1),
		)
		args = append(args, f.DeletedTo.UTC())
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "li.deleted_at IS NOT NULL")
		} else {
			where = append(where, "li.deleted_at IS NULL")
		}
	}

	return where, args
}

func buildListImageOrderBy(sort listimagedom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "li.id"
	case "listid", "list_id":
		col = "li.list_id"
	case "url":
		col = "li.url"
	case "filename", "file_name":
		col = "li.file_name"
	case "size":
		col = "li.size"
	case "displayorder", "display_order":
		col = "li.display_order"
	case "createdat", "created_at":
		col = "li.created_at"
	case "updatedat", "updated_at":
		col = "li.updated_at"
	case "deletedat", "deleted_at":
		col = "li.deleted_at"
	case "createdby", "created_by":
		col = "li.created_by"
	case "updatedby", "updated_by":
		col = "li.updated_by"
	case "deletedby", "deleted_by":
		col = "li.deleted_by"
	default:
		return ""
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}

// buildGCSURL is a tiny helper to form a URL string from bucket/object.
// tailor this for your public URL scheme later.
func buildGCSURL(bucket, objectPath string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		// もし listimagedom で DefaultBucket とか持ってるならここで fallback してOK
		// b = listimagedom.DefaultBucket
	}
	return "gs://" + b + "/" + strings.TrimSpace(objectPath)
}
