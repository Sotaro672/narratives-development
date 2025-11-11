package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	cimgdom "narratives/internal/domain/campaignImage"
)

// Repository implementation for CampaignImage (PostgreSQL)
type CampaignImageRepositoryPG struct {
	DB *sql.DB
}

func NewCampaignImageRepositoryPG(db *sql.DB) *CampaignImageRepositoryPG {
	return &CampaignImageRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

func (r *CampaignImageRepositoryPG) List(ctx context.Context, filter cimgdom.Filter, sort cimgdom.Sort, page cimgdom.Page) (cimgdom.PageResult[cimgdom.CampaignImage], error) {
	where, args := buildCampaignImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildCampaignImageOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM campaign_images %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return cimgdom.PageResult[cimgdom.CampaignImage]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, campaign_id, image_url, width, height, file_size, mime_type
FROM campaign_images
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return cimgdom.PageResult[cimgdom.CampaignImage]{}, err
	}
	defer rows.Close()

	var items []cimgdom.CampaignImage
	for rows.Next() {
		item, err := scanCampaignImage(rows)
		if err != nil {
			return cimgdom.PageResult[cimgdom.CampaignImage]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return cimgdom.PageResult[cimgdom.CampaignImage]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return cimgdom.PageResult[cimgdom.CampaignImage]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *CampaignImageRepositoryPG) ListByCursor(ctx context.Context, filter cimgdom.Filter, _ cimgdom.Sort, cpage cimgdom.CursorPage) (cimgdom.CursorPageResult[cimgdom.CampaignImage], error) {
	where, args := buildCampaignImageWhere(filter)
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("id > $%d", len(args)+1))
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
  id, campaign_id, image_url, width, height, file_size, mime_type
FROM campaign_images
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return cimgdom.CursorPageResult[cimgdom.CampaignImage]{}, err
	}
	defer rows.Close()

	var items []cimgdom.CampaignImage
	var lastID string
	for rows.Next() {
		item, err := scanCampaignImage(rows)
		if err != nil {
			return cimgdom.CursorPageResult[cimgdom.CampaignImage]{}, err
		}
		items = append(items, item)
		lastID = item.ID
	}
	if err := rows.Err(); err != nil {
		return cimgdom.CursorPageResult[cimgdom.CampaignImage]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return cimgdom.CursorPageResult[cimgdom.CampaignImage]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *CampaignImageRepositoryPG) GetByID(ctx context.Context, id string) (cimgdom.CampaignImage, error) {
	const q = `
SELECT
  id, campaign_id, image_url, width, height, file_size, mime_type
FROM campaign_images
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	item, err := scanCampaignImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}
	return item, nil
}

func (r *CampaignImageRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM campaign_images WHERE id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *CampaignImageRepositoryPG) Count(ctx context.Context, filter cimgdom.Filter) (int, error) {
	where, args := buildCampaignImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaign_images `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *CampaignImageRepositoryPG) Create(ctx context.Context, img cimgdom.CampaignImage) (cimgdom.CampaignImage, error) {
	const q = `
INSERT INTO campaign_images (
  id, campaign_id, image_url, width, height, file_size, mime_type
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
RETURNING
  id, campaign_id, image_url, width, height, file_size, mime_type
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(img.ID),
		strings.TrimSpace(img.CampaignID),
		strings.TrimSpace(img.ImageURL),
		dbcommon.ToDBInt(img.Width),
		dbcommon.ToDBInt(img.Height),
		dbcommon.ToDBInt64(img.FileSize),
		dbcommon.ToDBText(img.MimeType),
	)
	out, err := scanCampaignImage(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return cimgdom.CampaignImage{}, cimgdom.ErrConflict
		}
		return cimgdom.CampaignImage{}, err
	}
	return out, nil
}

func (r *CampaignImageRepositoryPG) Update(ctx context.Context, id string, patch cimgdom.CampaignImagePatch) (cimgdom.CampaignImage, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.CampaignID != nil {
		sets = append(sets, fmt.Sprintf("campaign_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.CampaignID))
		i++
	}
	if patch.ImageURL != nil {
		sets = append(sets, fmt.Sprintf("image_url = $%d", i))
		args = append(args, strings.TrimSpace(*patch.ImageURL))
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
	if patch.FileSize != nil {
		sets = append(sets, fmt.Sprintf("file_size = $%d", i))
		args = append(args, dbcommon.ToDBInt64(patch.FileSize))
		i++
	}
	if patch.MimeType != nil {
		sets = append(sets, fmt.Sprintf("mime_type = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.MimeType))
		i++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
UPDATE campaign_images
SET %s
WHERE id = $%d
RETURNING
  id, campaign_id, image_url, width, height, file_size, mime_type
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanCampaignImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cimgdom.CampaignImage{}, cimgdom.ErrNotFound
		}
		return cimgdom.CampaignImage{}, err
	}
	return out, nil
}

func (r *CampaignImageRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM campaign_images WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return cimgdom.ErrNotFound
	}
	return nil
}

func (r *CampaignImageRepositoryPG) Save(ctx context.Context, img cimgdom.CampaignImage, _ *cimgdom.SaveOptions) (cimgdom.CampaignImage, error) {
	const q = `
INSERT INTO campaign_images (
  id, campaign_id, image_url, width, height, file_size, mime_type
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
ON CONFLICT (id) DO UPDATE SET
  campaign_id = EXCLUDED.campaign_id,
  image_url   = EXCLUDED.image_url,
  width       = EXCLUDED.width,
  height      = EXCLUDED.height,
  file_size   = EXCLUDED.file_size,
  mime_type   = EXCLUDED.mime_type
RETURNING
  id, campaign_id, image_url, width, height, file_size, mime_type
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(img.ID),
		strings.TrimSpace(img.CampaignID),
		strings.TrimSpace(img.ImageURL),
		dbcommon.ToDBInt(img.Width),
		dbcommon.ToDBInt(img.Height),
		dbcommon.ToDBInt64(img.FileSize),
		dbcommon.ToDBText(img.MimeType),
	)
	out, err := scanCampaignImage(row)
	if err != nil {
		return cimgdom.CampaignImage{}, err
	}
	return out, nil
}

// =======================
// GCS-friendly helpers
// =======================

// GCS default bucket for campaign images in this repository layer.
const defaultCampaignImageBucket = "narratives_development_campaign_image"

// SaveFromBucketObject builds a public URL from bucket/object and saves.
func (r *CampaignImageRepositoryPG) SaveFromBucketObject(
	ctx context.Context,
	id string,
	campaignID string,
	bucket string,
	objectPath string,
	width, height *int,
	fileSize *int64,
	mimeType *string,
	// kept for backward compatibility, but not used anymore
	_ *string,
	_ time.Time,
) (cimgdom.CampaignImage, error) {
	// Build public URL locally
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultCampaignImageBucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return cimgdom.CampaignImage{}, fmt.Errorf("campaignImage: empty objectPath")
	}
	publicURL := gcsPublicURL(b, obj)

	img := cimgdom.CampaignImage{
		ID:         strings.TrimSpace(id),
		CampaignID: strings.TrimSpace(campaignID),
		ImageURL:   publicURL,
		Width:      width,
		Height:     height,
		FileSize:   fileSize,
		MimeType:   mimeType,
	}
	return r.Save(ctx, img, nil)
}

// BuildDeleteOps implements cimgdom.GCSDeleteOpsProvider.
func (r *CampaignImageRepositoryPG) BuildDeleteOps(ctx context.Context, ids []string) ([]cimgdom.GCSDeleteOp, error) {
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			cleaned = append(cleaned, id)
		}
	}
	if len(cleaned) == 0 {
		return nil, nil
	}

	ph := make([]string, 0, len(cleaned))
	args := make([]any, 0, len(cleaned))
	for _, id := range cleaned {
		ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, id)
	}

	q := fmt.Sprintf(`SELECT id, image_url FROM campaign_images WHERE id IN (%s)`, strings.Join(ph, ","))
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ops := make([]cimgdom.GCSDeleteOp, 0, len(cleaned))
	for rows.Next() {
		var id, urlStr string
		if err := rows.Scan(&id, &urlStr); err != nil {
			return nil, err
		}
		ops = append(ops, toGCSDeleteOpFromURL(strings.TrimSpace(urlStr), strings.TrimSpace(id)))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ops, nil
}

// BuildDeleteOpsByCampaignID implements cimgdom.GCSDeleteOpsProvider.
func (r *CampaignImageRepositoryPG) BuildDeleteOpsByCampaignID(ctx context.Context, campaignID string) ([]cimgdom.GCSDeleteOp, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, nil
	}
	const q = `SELECT id, image_url FROM campaign_images WHERE campaign_id = $1`
	rows, err := r.DB.QueryContext(ctx, q, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []cimgdom.GCSDeleteOp
	for rows.Next() {
		var id, urlStr string
		if err := rows.Scan(&id, &urlStr); err != nil {
			return nil, err
		}
		ops = append(ops, toGCSDeleteOpFromURL(strings.TrimSpace(urlStr), strings.TrimSpace(id)))
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func toGCSDeleteOpFromURL(imageURL string, id string) cimgdom.GCSDeleteOp {
	if b, obj, ok := parseGCSURL(imageURL); ok {
		return cimgdom.GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	// Fallback: delete by synthesized path under default bucket
	return cimgdom.GCSDeleteOp{
		Bucket:     defaultCampaignImageBucket,
		ObjectPath: path.Join("campaign_images", strings.TrimSpace(id)),
	}
}

// scanCampaignImage scans only the needed columns.
func scanCampaignImage(s dbcommon.RowScanner) (cimgdom.CampaignImage, error) {
	var (
		id, campaignID, imageURL string
		widthN                   sql.NullInt32
		heightN                  sql.NullInt32
		fileSizeN                sql.NullInt64
		mimeTypeN                sql.NullString
	)
	if err := s.Scan(
		&id, &campaignID, &imageURL,
		&widthN, &heightN, &fileSizeN, &mimeTypeN,
	); err != nil {
		return cimgdom.CampaignImage{}, err
	}

	return cimgdom.CampaignImage{
		ID:         strings.TrimSpace(id),
		CampaignID: strings.TrimSpace(campaignID),
		ImageURL:   strings.TrimSpace(imageURL),
		Width:      ptrIntFromNull(widthN),
		Height:     ptrIntFromNull(heightN),
		FileSize:   ptrInt64FromNull(fileSizeN),
		MimeType:   ptrStringFromNull(mimeTypeN),
	}, nil
}

func ptrStringFromNull(n sql.NullString) *string {
	if n.Valid {
		s := strings.TrimSpace(n.String)
		return &s
	}
	return nil
}
func ptrIntFromNull(n sql.NullInt32) *int {
	if n.Valid {
		v := int(n.Int32)
		return &v
	}
	return nil
}
func ptrInt64FromNull(n sql.NullInt64) *int64 {
	if n.Valid {
		v := n.Int64
		return &v
	}
	return nil
}

// buildCampaignImageWhere builds WHERE clauses and args from Filter.
func buildCampaignImageWhere(f cimgdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}
	next := func(val any, cond string) {
		where = append(where, fmt.Sprintf(cond, len(args)+1))
		args = append(args, val)
	}

	// Search text (ILIKE) -> image_url, mime_type のみ対象
	if s := strings.TrimSpace(f.SearchQuery); s != "" {
		like := "%" + s + "%"
		where = append(where, fmt.Sprintf("(image_url ILIKE $%d OR mime_type ILIKE $%d)", len(args)+1, len(args)+2))
		args = append(args, like, like)
	}

	// Campaign filters
	if f.CampaignID != nil && strings.TrimSpace(*f.CampaignID) != "" {
		next(strings.TrimSpace(*f.CampaignID), "campaign_id = $%d")
	}
	if len(f.CampaignIDs) > 0 {
		ids := make([]string, 0, len(f.CampaignIDs))
		for _, id := range f.CampaignIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			ph := make([]string, 0, len(ids))
			for range ids {
				ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, ids[len(ph)-1])
			}
			where = append(where, "campaign_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// MimeType filters
	if len(f.MimeTypes) > 0 {
		mts := make([]string, 0, len(f.MimeTypes))
		for _, m := range f.MimeTypes {
			m = strings.TrimSpace(m)
			if m != "" {
				mts = append(mts, m)
			}
		}
		if len(mts) > 0 {
			ph := make([]string, 0, len(mts))
			for range mts {
				ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, mts[len(ph)-1])
			}
			where = append(where, "mime_type IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Size/Dimension ranges
	if f.WidthMin != nil {
		next(*f.WidthMin, "width >= $%d")
	}
	if f.WidthMax != nil {
		next(*f.WidthMax, "width <= $%d")
	}
	if f.HeightMin != nil {
		next(*f.HeightMin, "height >= $%d")
	}
	if f.HeightMax != nil {
		next(*f.HeightMax, "height <= $%d")
	}
	if f.FileSizeMin != nil {
		next(*f.FileSizeMin, "file_size >= $%d")
	}
	if f.FileSizeMax != nil {
		next(*f.FileSizeMax, "file_size <= $%d")
	}

	return where, args
}

// buildCampaignImageOrderBy returns ORDER BY clause or "" (fallback to default in callers).
func buildCampaignImageOrderBy(_ cimgdom.Sort) string {
	// 拡張の余地を残し、現状は呼び出し側デフォルト（id DESC）に委ねます。
	return ""
}
