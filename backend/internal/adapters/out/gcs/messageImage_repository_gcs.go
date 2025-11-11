// backend\internal\adapters\out\firestore\messageImage_repository_gcs.go
package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	midom "narratives/internal/domain/messageImage"

	"cloud.google.com/go/storage"
)

type MessageImageRepositoryPG struct {
	DB *sql.DB
}

func NewMessageImageRepositoryPG(db *sql.DB) *MessageImageRepositoryPG {
	return &MessageImageRepositoryPG{DB: db}
}

// ========================================
// RepositoryPort impl
// ========================================

func (r *MessageImageRepositoryPG) ListByMessageID(ctx context.Context, messageID string) ([]midom.ImageFile, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
FROM message_images
WHERE message_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC, file_name ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(messageID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []midom.ImageFile
	for rows.Next() {
		img, err := scanMessageImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

func (r *MessageImageRepositoryPG) Get(ctx context.Context, messageID, fileName string) (*midom.ImageFile, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
FROM message_images
WHERE message_id = $1 AND file_name = $2`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(messageID), strings.TrimSpace(fileName))
	img, err := scanMessageImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, midom.ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (r *MessageImageRepositoryPG) List(ctx context.Context, filter midom.Filter, sort midom.Sort, page midom.Page) (midom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildMessageImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildMessageImageOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at ASC, file_name ASC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_images "+whereSQL, args...).Scan(&total); err != nil {
		return midom.PageResult{}, err
	}

	q := fmt.Sprintf(`
SELECT
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
FROM message_images
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return midom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]midom.ImageFile, 0, perPage)
	for rows.Next() {
		img, err := scanMessageImage(rows)
		if err != nil {
			return midom.PageResult{}, err
		}
		items = append(items, img)
	}
	if err := rows.Err(); err != nil {
		return midom.PageResult{}, err
	}

	return midom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *MessageImageRepositoryPG) Count(ctx context.Context, filter midom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildMessageImageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_images "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *MessageImageRepositoryPG) Add(ctx context.Context, img midom.ImageFile) (midom.ImageFile, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
INSERT INTO message_images (
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8,$9,$10
)
RETURNING
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(img.MessageID),
		strings.TrimSpace(img.FileName),
		strings.TrimSpace(img.FileURL),
		img.FileSize,
		strings.TrimSpace(img.MimeType),
		dbcommon.ToDBInt(img.Width),
		dbcommon.ToDBInt(img.Height),
		img.CreatedAt.UTC(),
		dbcommon.ToDBTime(img.UpdatedAt),
		dbcommon.ToDBTime(img.DeletedAt),
	)
	out, err := scanMessageImage(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return midom.ImageFile{}, midom.ErrConflict
		}
		return midom.ImageFile{}, err
	}
	return out, nil
}

func (r *MessageImageRepositoryPG) ReplaceAll(ctx context.Context, messageID string, images []midom.ImageFile) ([]midom.ImageFile, error) {
	// Use TX for atomic replace
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	ctxTx := dbcommon.CtxWithTx(ctx, tx)

	if err := r.DeleteAll(ctxTx, messageID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	out := make([]midom.ImageFile, 0, len(images))
	for _, img := range images {
		img.MessageID = strings.TrimSpace(messageID)
		saved, err := r.Add(ctxTx, img)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		out = append(out, saved)
	}

	return out, tx.Commit()
}

func (r *MessageImageRepositoryPG) Update(ctx context.Context, messageID, fileName string, patch midom.ImageFilePatch) (midom.ImageFile, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

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
	setTime := func(col string, p *time.Time) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, p.UTC())
			i++
		}
	}

	// Updatable columns
	setStr("file_name", patch.FileName)
	setStr("file_url", patch.FileURL)
	setInt64("file_size", patch.FileSize)
	setStr("mime_type", patch.MimeType)
	setInt("width", patch.Width)
	setInt("height", patch.Height)
	setTime("updated_at", patch.UpdatedAt)
	setTime("deleted_at", patch.DeletedAt)

	// Auto touch updated_at if anything changed and UpdatedAt not explicitly set
	if patch.UpdatedAt == nil && len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	if len(sets) == 0 {
		// nothing to update; return current
		got, err := r.Get(ctx, messageID, fileName)
		if err != nil {
			return midom.ImageFile{}, err
		}
		return *got, nil
	}

	args = append(args, strings.TrimSpace(messageID), strings.TrimSpace(fileName))
	q := fmt.Sprintf(`
UPDATE message_images
SET %s
WHERE message_id = $%d AND file_name = $%d
RETURNING
  message_id, file_name, file_url, file_size, mime_type, width, height,
  created_at, updated_at, deleted_at
`, strings.Join(sets, ", "), i, i+1)

	row := run.QueryRowContext(ctx, q, args...)
	out, err := scanMessageImage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return midom.ImageFile{}, midom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return midom.ImageFile{}, midom.ErrConflict
		}
		return midom.ImageFile{}, err
	}
	return out, nil
}

func (r *MessageImageRepositoryPG) Delete(ctx context.Context, messageID, fileName string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM message_images WHERE message_id = $1 AND file_name = $2`, strings.TrimSpace(messageID), strings.TrimSpace(fileName))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return midom.ErrNotFound
	}
	return nil
}

func (r *MessageImageRepositoryPG) DeleteAll(ctx context.Context, messageID string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM message_images WHERE message_id = $1`, strings.TrimSpace(messageID))
	return err
}

// ========================================
// Helpers
// ========================================

func scanMessageImage(s dbcommon.RowScanner) (midom.ImageFile, error) {
	var (
		messageID, fileName, fileURL, mimeType string
		fileSize                               int64
		widthNS, heightNS                      sql.NullInt64
		createdAt                              time.Time
		updatedAtNS, deletedAtNS               sql.NullTime
	)
	if err := s.Scan(
		&messageID, &fileName, &fileURL, &fileSize, &mimeType, &widthNS, &heightNS,
		&createdAt, &updatedAtNS, &deletedAtNS,
	); err != nil {
		return midom.ImageFile{}, err
	}

	var widthPtr, heightPtr *int
	if widthNS.Valid {
		w := int(widthNS.Int64)
		widthPtr = &w
	}
	if heightNS.Valid {
		h := int(heightNS.Int64)
		heightPtr = &h
	}

	toPtrTime := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}

	return midom.ImageFile{
		MessageID: strings.TrimSpace(messageID),
		FileName:  strings.TrimSpace(fileName),
		FileURL:   strings.TrimSpace(fileURL),
		FileSize:  fileSize,
		MimeType:  strings.TrimSpace(mimeType),
		Width:     widthPtr,
		Height:    heightPtr,

		CreatedAt: createdAt.UTC(),
		UpdatedAt: toPtrTime(updatedAtNS),
		DeletedAt: toPtrTime(deletedAtNS),
	}, nil
}

func buildMessageImageWhere(f midom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	if v := strings.TrimSpace(f.MessageID); v != "" {
		where = append(where, fmt.Sprintf("message_id = $%d", len(args)+1))
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.FileNameLike); v != "" {
		where = append(where, fmt.Sprintf("file_name ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}
	if f.MimeType != nil {
		mt := strings.TrimSpace(*f.MimeType)
		if mt != "" {
			where = append(where, fmt.Sprintf("mime_type = $%d", len(args)+1))
			args = append(args, mt)
		}
	}
	if f.MinSize != nil {
		where = append(where, fmt.Sprintf("file_size >= $%d", len(args)+1))
		args = append(args, *f.MinSize)
	}
	if f.MaxSize != nil {
		where = append(where, fmt.Sprintf("file_size <= $%d", len(args)+1))
		args = append(args, *f.MaxSize)
	}

	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at >= $%d)", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at < $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}

	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "deleted_at IS NOT NULL")
		} else {
			where = append(where, "deleted_at IS NULL")
		}
	}

	return where, args
}

func buildMessageImageOrderBy(sort midom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "filename", "file_name":
		col = "file_name"
	case "filesize", "file_size":
		col = "file_size"
	case "updatedat", "updated_at":
		col = "updated_at"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s, file_name %s", col, dir, dir)
}

// MessageImageStorageGCS implements messageImage.ObjectStoragePort using Google Cloud Storage.
type MessageImageStorageGCS struct {
	Client          *storage.Client
	Bucket          string
	SignedURLExpiry time.Duration
}

// NewMessageImageStorageGCS creates a storage adapter with the provided client.
// If bucket is empty, it falls back to midom.DefaultBucket ("narratives_development_message_image").
func NewMessageImageStorageGCS(client *storage.Client, bucket string) *MessageImageStorageGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = midom.DefaultBucket
	}
	return &MessageImageStorageGCS{
		Client:          client,
		Bucket:          b,
		SignedURLExpiry: 15 * time.Minute,
	}
}

// DeleteObject deletes a single GCS object.
// bucket or objectPath can be empty/relative; bucket falls back to adapter's default, objectPath gets trimmed.
func (s *MessageImageStorageGCS) DeleteObject(ctx context.Context, bucket, objectPath string) error {
	if s.Client == nil {
		return errors.New("MessageImageStorageGCS: nil storage client")
	}
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = s.Bucket
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if b == "" || obj == "" {
		return fmt.Errorf("invalid bucket/objectPath: bucket=%q, objectPath=%q", b, objectPath)
	}
	err := s.Client.Bucket(b).Object(obj).Delete(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			// Treat as success (idempotent delete)
			return nil
		}
		return err
	}
	return nil
}

// DeleteObjects deletes multiple GCS objects best-effort.
// It continues on errors and returns a combined error if any failures occurred.
func (s *MessageImageStorageGCS) DeleteObjects(ctx context.Context, ops []midom.GCSDeleteOp) error {
	if len(ops) == 0 {
		return nil
	}
	var errs []error
	for _, op := range ops {
		b := strings.TrimSpace(op.Bucket)
		if b == "" {
			b = s.Bucket
		}
		if err := s.DeleteObject(ctx, b, op.ObjectPath); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", b, op.ObjectPath, err))
		}
	}
	if len(errs) > 0 {
		return dbcommon.JoinErrors(errs)
	}
	return nil
}
