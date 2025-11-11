// backend\internal\adapters\out\firestore\inquiryImage_repository_gcs.go
package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	idom "narratives/internal/domain/inquiryImage"
)

// InquiryImageRepositoryPG implements inquiryimage.Repository using PostgreSQL.
type InquiryImageRepositoryPG struct {
	DB *sql.DB
}

func NewInquiryImageRepositoryPG(db *sql.DB) *InquiryImageRepositoryPG {
	return &InquiryImageRepositoryPG{DB: db}
}

type runner interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
}

// =======================================
// Aggregate queries
// =======================================

func (r *InquiryImageRepositoryPG) GetImagesByInquiryID(ctx context.Context, inquiryID string) (*idom.InquiryImage, error) {
	inquiryID = strings.TrimSpace(inquiryID)
	const q = `
SELECT
  inquiry_id, file_name, file_url, file_size, mime_type,
  width, height, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM inquiry_image_files
WHERE inquiry_id = $1
ORDER BY created_at ASC, file_name ASC
`
	rows, err := r.DB.QueryContext(ctx, q, inquiryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agg := idom.InquiryImage{ID: inquiryID, Images: []idom.ImageFile{}}
	for rows.Next() {
		im, err := scanImageFile(rows)
		if err != nil {
			return nil, err
		}
		agg.Images = append(agg.Images, im)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If no files, ensure header exists; otherwise NotFound
	if len(agg.Images) == 0 {
		var one int
		err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM inquiry_images WHERE id = $1`, inquiryID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, idom.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	return &agg, nil
}

func (r *InquiryImageRepositoryPG) Exists(ctx context.Context, inquiryID, fileName string) (bool, error) {
	const q = `SELECT 1 FROM inquiry_image_files WHERE inquiry_id = $1 AND file_name = $2`
	var one int
	err := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(inquiryID), strings.TrimSpace(fileName)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// =======================================
// Listing
// =======================================

func (r *InquiryImageRepositoryPG) ListImages(ctx context.Context, filter idom.Filter, sort idom.Sort, page idom.Page) (idom.PageResult[idom.ImageFile], error) {
	where, args := buildImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	orderBy := buildImageOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY iif.created_at DESC, iif.inquiry_id DESC, iif.file_name DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM inquiry_image_files iif %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return idom.PageResult[idom.ImageFile]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  iif.inquiry_id, iif.file_name, iif.file_url, iif.file_size, iif.mime_type,
  iif.width, iif.height, iif.created_at, iif.created_by, iif.updated_at, iif.updated_by, iif.deleted_at, iif.deleted_by
FROM inquiry_image_files iif
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return idom.PageResult[idom.ImageFile]{}, err
	}
	defer rows.Close()

	var items []idom.ImageFile
	for rows.Next() {
		im, err := scanImageFile(rows)
		if err != nil {
			return idom.PageResult[idom.ImageFile]{}, err
		}
		items = append(items, im)
	}
	if err := rows.Err(); err != nil {
		return idom.PageResult[idom.ImageFile]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return idom.PageResult[idom.ImageFile]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *InquiryImageRepositoryPG) ListImagesByCursor(ctx context.Context, filter idom.Filter, _ idom.Sort, cpage idom.CursorPage) (idom.CursorPageResult[idom.ImageFile], error) {
	where, args := buildImageWhere(filter)

	// Cursor: encoded as "inquiry_id|file_name"
	if after := strings.TrimSpace(cpage.After); after != "" {
		aid, afn := splitCursor(after)
		where = append(where, fmt.Sprintf("(iif.inquiry_id, iif.file_name) > ($%d, $%d)", len(args)+1, len(args)+2))
		args = append(args, aid, afn)
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
  iif.inquiry_id, iif.file_name, iif.file_url, iif.file_size, iif.mime_type,
  iif.width, iif.height, iif.created_at, iif.created_by, iif.updated_at, iif.updated_by, iif.deleted_at, iif.deleted_by
FROM inquiry_image_files iif
%s
ORDER BY iif.inquiry_id ASC, iif.file_name ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return idom.CursorPageResult[idom.ImageFile]{}, err
	}
	defer rows.Close()

	var items []idom.ImageFile
	var lastAid, lastFn string
	for rows.Next() {
		im, err := scanImageFile(rows)
		if err != nil {
			return idom.CursorPageResult[idom.ImageFile]{}, err
		}
		items = append(items, im)
		lastAid, lastFn = im.InquiryID, im.FileName
	}
	if err := rows.Err(); err != nil {
		return idom.CursorPageResult[idom.ImageFile]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		cur := makeCursor(lastAid, lastFn)
		next = &cur
	}

	return idom.CursorPageResult[idom.ImageFile]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *InquiryImageRepositoryPG) Count(ctx context.Context, filter idom.Filter) (int, error) {
	where, args := buildImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM inquiry_image_files iif `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================================
// Mutations
// =======================================

func (r *InquiryImageRepositoryPG) AddImage(ctx context.Context, inquiryID string, req idom.AddImageRequest) (*idom.InquiryImage, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	inquiryID = strings.TrimSpace(inquiryID)
	if err := ensureInquiryHeader(ctx, tx, inquiryID); err != nil {
		return nil, err
	}

	const q = `
INSERT INTO inquiry_image_files (
  inquiry_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  NOW(), $8, NULL, NULL, NULL, NULL
)
`
	// created_by: fallback to 'system' if no actor available
	createdBy := "system"
	if _, err := tx.ExecContext(ctx, q,
		inquiryID,
		strings.TrimSpace(req.FileName),
		strings.TrimSpace(req.FileURL),
		req.FileSize,
		strings.TrimSpace(req.MimeType),
		dbcommon.ToDBInt(req.Width),
		dbcommon.ToDBInt(req.Height),
		createdBy,
	); err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, idom.ErrConflict
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetImagesByInquiryID(ctx, inquiryID)
}

func (r *InquiryImageRepositoryPG) UpdateImages(ctx context.Context, inquiryID string, req idom.UpdateImagesRequest) (*idom.InquiryImage, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	inquiryID = strings.TrimSpace(inquiryID)
	if err := ensureInquiryHeader(ctx, tx, inquiryID); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM inquiry_image_files WHERE inquiry_id = $1`, inquiryID); err != nil {
		return nil, err
	}

	if len(req.Images) > 0 {
		if err := bulkInsertImages(ctx, tx, inquiryID, req.Images); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetImagesByInquiryID(ctx, inquiryID)
}

func (r *InquiryImageRepositoryPG) PatchImage(ctx context.Context, inquiryID, fileName string, patch idom.ImagePatch) (*idom.ImageFile, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.FileName != nil {
		sets = append(sets, fmt.Sprintf("file_name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.FileName))
		i++
	}
	if patch.FileURL != nil {
		sets = append(sets, fmt.Sprintf("file_url = $%d", i))
		args = append(args, strings.TrimSpace(*patch.FileURL))
		i++
	}
	if patch.FileSize != nil {
		sets = append(sets, fmt.Sprintf("file_size = $%d", i))
		args = append(args, *patch.FileSize)
		i++
	}
	if patch.MimeType != nil {
		sets = append(sets, fmt.Sprintf("mime_type = $%d", i))
		args = append(args, strings.TrimSpace(*patch.MimeType))
		i++
	}
	if patch.Width != nil {
		sets = append(sets, fmt.Sprintf("width = $%d", i))
		args = append(args, dbcommon.ToDBInt(patch.Width))
		i++
	}
	if patch.Height != nil {
		sets = append(sets, fmt.Sprintf("height = $%d", i))
		args = append(args, dbcommon.ToDBInt(patch.Height))
		i++
	}
	if patch.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.UpdatedBy))
		i++
	}
	// updated_at explicit or NOW()
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
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

	if len(sets) == 0 {
		// Return the current row
		const sel = `
SELECT inquiry_id, file_name, file_url, file_size, mime_type,
       width, height, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM inquiry_image_files
WHERE inquiry_id = $1 AND file_name = $2
`
		row := r.DB.QueryRowContext(ctx, sel, strings.TrimSpace(inquiryID), strings.TrimSpace(fileName))
		im, err := scanImageFile(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, idom.ErrNotFound
			}
			return nil, err
		}
		return &im, nil
	}

	args = append(args, strings.TrimSpace(inquiryID), strings.TrimSpace(fileName))
	q := fmt.Sprintf(`
UPDATE inquiry_image_files
SET %s
WHERE inquiry_id = $%d AND file_name = $%d
RETURNING inquiry_id, file_name, file_url, file_size, mime_type,
          width, height, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i, i+1)

	row := r.DB.QueryRowContext(ctx, q, args...)
	im, err := scanImageFile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, idom.ErrNotFound
		}
		return nil, err
	}
	return &im, nil
}

func (r *InquiryImageRepositoryPG) DeleteImage(ctx context.Context, inquiryID, fileName string) (*idom.InquiryImage, error) {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM inquiry_image_files WHERE inquiry_id = $1 AND file_name = $2`, strings.TrimSpace(inquiryID), strings.TrimSpace(fileName))
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, idom.ErrNotFound
	}
	// return aggregate
	return r.GetImagesByInquiryID(ctx, strings.TrimSpace(inquiryID))
}

func (r *InquiryImageRepositoryPG) DeleteAllImages(ctx context.Context, inquiryID string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM inquiry_image_files WHERE inquiry_id = $1`, strings.TrimSpace(inquiryID))
	return err
}

func (r *InquiryImageRepositoryPG) Save(ctx context.Context, agg idom.InquiryImage, _ *idom.SaveOptions) (*idom.InquiryImage, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id := strings.TrimSpace(agg.ID)
	if err := ensureInquiryHeader(ctx, tx, id); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM inquiry_image_files WHERE inquiry_id = $1`, id); err != nil {
		return nil, err
	}
	if len(agg.Images) > 0 {
		if err := bulkInsertImages(ctx, tx, id, agg.Images); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetImagesByInquiryID(ctx, id)
}

// =======================================
// GCS-friendly helpers (implements inquiryimage.GCSObjectSaver, GCSDeleteOpsProvider)
// =======================================

// SaveImageFromBucketObject implements idom.GCSObjectSaver.
// - bucket が空なら idom.DefaultBucket (narratives_development_inquiry_image) を使用
// - objectPath から公開URLを構築して保存（存在すれば upsert）
func (r *InquiryImageRepositoryPG) SaveImageFromBucketObject(
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
	inquiryID = strings.TrimSpace(inquiryID)
	fileName = strings.TrimSpace(fileName)

	if inquiryID == "" || fileName == "" {
		return nil, fmt.Errorf("inquiryImage: empty inquiryID or fileName")
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = idom.DefaultBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return nil, fmt.Errorf("inquiryImage: empty objectPath")
	}

	// 公開URLを組み立て（entity 側の PublicURL があれば利用）
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)
	if fn := getPublicURLFunc(); fn != nil {
		publicURL = fn(b, obj)
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// ヘッダ確保
	if err := ensureInquiryHeader(ctx, tx, inquiryID); err != nil {
		return nil, err
	}

	// upsert（inquiry_id, file_name に一意制約想定）
	const q = `
INSERT INTO inquiry_image_files (
  inquiry_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8,$9, NULL, NULL, NULL, NULL
)
ON CONFLICT (inquiry_id, file_name) DO UPDATE SET
  file_url  = EXCLUDED.file_url,
  file_size = EXCLUDED.file_size,
  mime_type = EXCLUDED.mime_type,
  width     = EXCLUDED.width,
  height    = EXCLUDED.height,
  updated_at= GREATEST(COALESCE(inquiry_image_files.updated_at, EXCLUDED.created_at), EXCLUDED.created_at),
  updated_by= EXCLUDED.created_by
RETURNING inquiry_id, file_name, file_url, file_size, mime_type,
          width, height, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`

	// created_by のフォールバック
	cb := strings.TrimSpace(createdBy)
	if cb == "" {
		cb = "system"
	}
	ca := createdAt.UTC()
	if ca.IsZero() {
		ca = time.Now().UTC()
	}

	row := tx.QueryRowContext(ctx, q,
		inquiryID,
		fileName,
		publicURL,
		fileSize,
		strings.TrimSpace(mimeType),
		dbcommon.ToDBInt(width),
		dbcommon.ToDBInt(height),
		ca,
		cb,
	)
	im, err := scanImageFile(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &im, nil
}

// BuildDeleteOps implements idom.GCSDeleteOpsProvider.
// 指定キー群のファイルURLから GCS 削除ターゲットを導出します。
func (r *InquiryImageRepositoryPG) BuildDeleteOps(ctx context.Context, keys []idom.ImageKey) ([]idom.GCSDeleteOp, error) {
	cleaned := make([]idom.ImageKey, 0, len(keys))
	for _, k := range keys {
		aid := strings.TrimSpace(k.InquiryID)
		fn := strings.TrimSpace(k.FileName)
		if aid != "" && fn != "" {
			cleaned = append(cleaned, idom.ImageKey{InquiryID: aid, FileName: fn})
		}
	}
	if len(cleaned) == 0 {
		return nil, nil
	}

	// WHERE (inquiry_id, file_name) IN ((...),(...),...)
	pairs := make([]string, 0, len(cleaned))
	args := make([]any, 0, len(cleaned)*2)
	for _, k := range cleaned {
		pairs = append(pairs, fmt.Sprintf("($%d,$%d)", len(args)+1, len(args)+2))
		args = append(args, k.InquiryID, k.FileName)
	}

	q := fmt.Sprintf(`
SELECT inquiry_id, file_name, file_url
FROM inquiry_image_files
WHERE (inquiry_id, file_name) IN (%s)
`, strings.Join(pairs, ","))

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ops := make([]idom.GCSDeleteOp, 0, len(cleaned))
	for rows.Next() {
		var aid, fn, urlStr string
		if err := rows.Scan(&aid, &fn, &urlStr); err != nil {
			return nil, err
		}
		// was: ops = append(ops, toGCSDeleteOpFromURL(urlStr, aid, fn))
		ops = append(ops, toInquiryImageGCSDeleteOpFromURL(urlStr, aid, fn))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ops, nil
}

// BuildDeleteOpsByInquiryID implements idom.GCSDeleteOpsProvider.
// inquiryID 配下のファイルURLから GCS 削除ターゲットを導出します。
func (r *InquiryImageRepositoryPG) BuildDeleteOpsByInquiryID(ctx context.Context, inquiryID string) ([]idom.GCSDeleteOp, error) {
	aid := strings.TrimSpace(inquiryID)
	if aid == "" {
		return nil, nil
	}
	const q = `SELECT file_name, file_url FROM inquiry_image_files WHERE inquiry_id = $1`
	rows, err := r.DB.QueryContext(ctx, q, aid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []idom.GCSDeleteOp
	for rows.Next() {
		var fn, urlStr string
		if err := rows.Scan(&fn, &urlStr); err != nil {
			return nil, err
		}
		// was: ops = append(ops, toGCSDeleteOpFromURL(urlStr, aid, fn))
		ops = append(ops, toInquiryImageGCSDeleteOpFromURL(urlStr, aid, fn))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ops, nil
}

// =======================================
// Helpers
// =======================================

func ensureInquiryHeader(ctx context.Context, ex runner, inquiryID string) error {
	_, err := ex.ExecContext(ctx, `INSERT INTO inquiry_images (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, inquiryID)
	return err
}

func bulkInsertImages(ctx context.Context, ex runner, inquiryID string, items []idom.ImageFile) error {
	if len(items) == 0 {
		return nil
	}
	sb := strings.Builder{}
	sb.WriteString(`INSERT INTO inquiry_image_files (
  inquiry_id, file_name, file_url, file_size, mime_type,
  width, height, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES `)
	args := make([]any, 0, len(items)*13)
	for i, it := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5,
			len(args)+6, len(args)+7, len(args)+8, len(args)+9, len(args)+10,
			len(args)+11, len(args)+12, len(args)+13,
		))
		args = append(args,
			inquiryID,
			strings.TrimSpace(it.FileName),
			strings.TrimSpace(it.FileURL),
			it.FileSize,
			strings.TrimSpace(it.MimeType),
			dbcommon.ToDBInt(it.Width),
			dbcommon.ToDBInt(it.Height),
			it.CreatedAt.UTC(),
			strings.TrimSpace(it.CreatedBy),
			dbcommon.ToDBTime(it.UpdatedAt),
			dbcommon.ToDBText(it.UpdatedBy),
			dbcommon.ToDBTime(it.DeletedAt),
			dbcommon.ToDBText(it.DeletedBy),
		)
	}
	_, err := ex.ExecContext(ctx, sb.String(), args...)
	if dbcommon.IsUniqueViolation(err) {
		return idom.ErrConflict
	}
	return err
}

func scanImageFile(s dbcommon.RowScanner) (idom.ImageFile, error) {
	var (
		inquiryIDNS, fileNameNS, fileURLNS, mimeNS, createdByNS, updatedByNS, deletedByNS sql.NullString
		fileSize                                                                          int64
		widthNS, heightNS                                                                 sql.NullInt64
		createdAt                                                                         time.Time
		updatedAtNS, deletedAtNS                                                          sql.NullTime
	)
	if err := s.Scan(
		&inquiryIDNS, &fileNameNS, &fileURLNS, &fileSize, &mimeNS,
		&widthNS, &heightNS, &createdAt, &createdByNS, &updatedAtNS, &updatedByNS, &deletedAtNS, &deletedByNS,
	); err != nil {
		return idom.ImageFile{}, err
	}

	toPtrInt := func(ns sql.NullInt64) *int {
		if ns.Valid {
			v := int(ns.Int64)
			if v > 0 {
				return &v
			}
		}
		return nil
	}
	toPtrTime := func(ns sql.NullTime) *time.Time {
		if ns.Valid {
			t := ns.Time.UTC()
			return &t
		}
		return nil
	}
	toPtrStr := func(ns sql.NullString) *string {
		if ns.Valid {
			v := strings.TrimSpace(ns.String)
			return &v
		}
		return nil
	}

	return idom.ImageFile{
		InquiryID: strings.TrimSpace(inquiryIDNS.String),
		FileName:  strings.TrimSpace(fileNameNS.String),
		FileURL:   strings.TrimSpace(fileURLNS.String),
		FileSize:  fileSize,
		MimeType:  strings.TrimSpace(mimeNS.String),
		Width:     toPtrInt(widthNS),
		Height:    toPtrInt(heightNS),
		CreatedAt: createdAt.UTC(),
		CreatedBy: strings.TrimSpace(createdByNS.String),
		UpdatedAt: toPtrTime(updatedAtNS),
		UpdatedBy: toPtrStr(updatedByNS), // fix: was toPtrTime(updatedByNS)
		DeletedAt: toPtrTime(deletedAtNS),
		DeletedBy: toPtrStr(deletedByNS), // fix: was toPtrTime(deletedByNS)
	}, nil
}

func buildImageWhere(_ idom.Filter) ([]string, []any) {
	// 最小実装: フィルタ未使用（安全に空条件を返す）
	return []string{}, []any{}
}

func buildImageOrderBy(_ idom.Sort) string {
	// 呼び出し側のデフォルト ORDER BY を利用
	return ""
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

// ===== GCS helpers for delete ops =====

// rename to avoid collision with campaignImage's helper
func toInquiryImageGCSDeleteOpFromURL(fileURL, inquiryID, fileName string) idom.GCSDeleteOp {
	if parse := getParseGCSURLFunc(); parse != nil {
		if b, obj, ok := parse(fileURL); ok {
			return idom.GCSDeleteOp{Bucket: b, ObjectPath: obj}
		}
	}
	return idom.GCSDeleteOp{
		Bucket:     idom.DefaultBucket,
		ObjectPath: path.Join("inquiry_images", strings.TrimSpace(inquiryID), strings.TrimSpace(fileName)),
	}
}

// オプショナル: entity 側の関数が未実装でもビルド通るように間接参照する
func getPublicURLFunc() func(bucket, objectPath string) string {
	// entity に PublicURL がない場合は nil を返し、呼び出し側でフォールバック実装を使用
	return nil
}
func getParseGCSURLFunc() func(u string) (string, string, bool) {
	// entity に ParseGCSURL がない場合は nil を返し、呼び出し側でフォールバック実装を使用
	return nil
}
