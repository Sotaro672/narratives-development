// backend\internal\adapters\out\gcs\tokenContents_repository_gcs.go
package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io" // added
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	tcdom "narratives/internal/domain/tokenContents"
)

type TokenContentsRepositoryPG struct {
	DB *sql.DB
}

func NewTokenContentsRepositoryPG(db *sql.DB) *TokenContentsRepositoryPG {
	return &TokenContentsRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *TokenContentsRepositoryPG) GetByID(ctx context.Context, id string) (*tcdom.TokenContent, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, name, type, url, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_contents
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	tc, err := scanTokenContent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tcdom.ErrNotFound
		}
		return nil, err
	}
	return &tc, nil
}

func (r *TokenContentsRepositoryPG) List(ctx context.Context, filter tcdom.Filter, sort tcdom.Sort, page tcdom.Page) (tcdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTCWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildTCOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_contents "+whereSQL, args...).Scan(&total); err != nil {
		return tcdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, name, type, url, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_contents
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return tcdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]tcdom.TokenContent, 0, perPage)
	for rows.Next() {
		tc, err := scanTokenContent(rows)
		if err != nil {
			return tcdom.PageResult{}, err
		}
		items = append(items, tc)
	}
	if err := rows.Err(); err != nil {
		return tcdom.PageResult{}, err
	}

	return tcdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenContentsRepositoryPG) Count(ctx context.Context, filter tcdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTCWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_contents "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TokenContentsRepositoryPG) Create(ctx context.Context, in tcdom.CreateTokenContentInput) (*tcdom.TokenContent, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO token_contents (
  id, name, type, url, size,
  created_at, created_by, updated_at, updated_by
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4,
  NOW(), 'system', NOW(), 'system'
)
RETURNING
  id, name, type, url, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(string(in.Type)),
		strings.TrimSpace(in.URL),
		in.Size,
	)
	tc, err := scanTokenContent(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, tcdom.ErrConflict
		}
		return nil, err
	}
	return &tc, nil
}

func (r *TokenContentsRepositoryPG) Update(ctx context.Context, id string, in tcdom.UpdateTokenContentInput) (*tcdom.TokenContent, error) {
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

	setStr("name", in.Name)
	if in.Type != nil {
		sets = append(sets, fmt.Sprintf(`type = $%d`, i))
		args = append(args, strings.TrimSpace(string(*in.Type)))
		i++
	}
	setStr("url", in.URL)
	setInt64("size", in.Size)

	// Always bump updated_at
	sets = append(sets, "updated_at = NOW()")

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE token_contents
SET %s
WHERE id = $%d
RETURNING
  id, name, type, url, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	tc, err := scanTokenContent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tcdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, tcdom.ErrConflict
		}
		return nil, err
	}
	return &tc, nil
}

func (r *TokenContentsRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM token_contents WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return tcdom.ErrNotFound
	}
	return nil
}

func (r *TokenContentsRepositoryPG) UploadContent(ctx context.Context, fileName, contentType string, _ io.Reader) (string, int64, error) {
	// Not handled by DB adapter. Implement in a storage adapter (e.g., S3/GCS).
	return "", 0, fmt.Errorf("UploadContent: not implemented in PG repository")
}

func (r *TokenContentsRepositoryPG) GetStats(ctx context.Context) (tcdom.TokenContentStats, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	var stats tcdom.TokenContentStats

	// totals
	if err := run.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(size),0) FROM token_contents`).Scan(&stats.TotalCount, &stats.TotalSize); err != nil {
		return tcdom.TokenContentStats{}, err
	}
	stats.TotalSizeFormatted = humanBytes(stats.TotalSize)

	// count by type
	rows, err := run.QueryContext(ctx, `SELECT type, COUNT(*) FROM token_contents GROUP BY type`)
	if err != nil {
		return tcdom.TokenContentStats{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var t string
		var c int
		if err := rows.Scan(&t, &c); err != nil {
			return tcdom.TokenContentStats{}, err
		}
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "image":
			stats.CountByType.Image += c
		case "video":
			stats.CountByType.Video += c
		case "pdf":
			stats.CountByType.PDF += c
		case "document":
			stats.CountByType.Document += c
			// "audio" not in enum; leave zero
		}
	}
	if err := rows.Err(); err != nil {
		return tcdom.TokenContentStats{}, err
	}

	return stats, nil
}

func (r *TokenContentsRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM token_contents`)
	return err
}

func (r *TokenContentsRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := dbcommon.CtxWithTx(ctx, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ========================
// Helpers
// ========================

func scanTokenContent(s dbcommon.RowScanner) (tcdom.TokenContent, error) {
	var (
		id, name, typ, url, createdBy, updatedBy string
		size                                     int64
		createdAt, updatedAt                     time.Time
		deletedAtNS                              sql.NullTime
		deletedByNS                              sql.NullString
	)
	if err := s.Scan(
		&id, &name, &typ, &url, &size,
		&createdAt, &createdBy, &updatedAt, &updatedBy, &deletedAtNS, &deletedByNS,
	); err != nil {
		return tcdom.TokenContent{}, err
	}

	return tcdom.TokenContent{
		ID:   strings.TrimSpace(id),
		Name: strings.TrimSpace(name),
		Type: tcdom.ContentType(strings.TrimSpace(typ)),
		URL:  strings.TrimSpace(url),
		Size: size,
	}, nil
}

func buildTCWhere(f tcdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addIn := func(col string, vals []string) {
		clean := make([]string, 0, len(vals))
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				clean = append(clean, v)
			}
		}
		if len(clean) == 0 {
			return
		}
		base := len(args)
		ph := make([]string, len(clean))
		for i, v := range clean {
			args = append(args, v)
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
	}

	addIn("id", f.IDs)

	if len(f.Types) > 0 {
		vs := make([]string, 0, len(f.Types))
		for _, t := range f.Types {
			if v := strings.TrimSpace(string(t)); v != "" {
				vs = append(vs, v)
			}
		}
		addIn("type", vs)
	}

	if v := strings.TrimSpace(f.NameLike); v != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}

	if f.SizeMin != nil {
		where = append(where, fmt.Sprintf("size >= $%d", len(args)+1))
		args = append(args, *f.SizeMin)
	}
	if f.SizeMax != nil {
		where = append(where, fmt.Sprintf("size <= $%d", len(args)+1))
		args = append(args, *f.SizeMax)
	}

	return where, args
}

func buildTCOrderBy(s tcdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "size":
		col = "size"
	case "name":
		col = "name"
	case "type":
		col = "type"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

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
