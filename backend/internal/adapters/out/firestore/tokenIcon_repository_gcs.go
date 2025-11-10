package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	tidom "narratives/internal/domain/tokenIcon"
)

type TokenIconRepositoryPG struct {
	DB *sql.DB
}

func NewTokenIconRepositoryPG(db *sql.DB) *TokenIconRepositoryPG {
	return &TokenIconRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *TokenIconRepositoryPG) GetByID(ctx context.Context, id string) (*tidom.TokenIcon, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_icons
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	ti, err := scanTokenIcon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}
	return &ti, nil
}

func (r *TokenIconRepositoryPG) List(ctx context.Context, filter tidom.Filter, sort tidom.Sort, page tidom.Page) (tidom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTIWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildTIOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_icons "+whereSQL, args...).Scan(&total); err != nil {
		return tidom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_icons
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return tidom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]tidom.TokenIcon, 0, perPage)
	for rows.Next() {
		ti, err := scanTokenIcon(rows)
		if err != nil {
			return tidom.PageResult{}, err
		}
		items = append(items, ti)
	}
	if err := rows.Err(); err != nil {
		return tidom.PageResult{}, err
	}

	return tidom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenIconRepositoryPG) Count(ctx context.Context, filter tidom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTIWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_icons "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TokenIconRepositoryPG) Create(ctx context.Context, in tidom.CreateTokenIconInput) (*tidom.TokenIcon, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO token_icons (
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by
) VALUES (
  gen_random_uuid()::text, $1, $2, $3,
  NOW(), 'system', NOW(), 'system'
)
RETURNING
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.URL),
		strings.TrimSpace(in.FileName),
		in.Size,
	)
	ti, err := scanTokenIcon(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, tidom.ErrConflict
		}
		return nil, err
	}
	return &ti, nil
}

func (r *TokenIconRepositoryPG) Update(ctx context.Context, id string, in tidom.UpdateTokenIconInput) (*tidom.TokenIcon, error) {
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

	setStr("file_name", in.FileName)
	setStr("url", in.URL)
	setInt64("size", in.Size)

	// Always bump updated_at
	sets = append(sets, "updated_at = NOW()")

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE token_icons
SET %s
WHERE id = $%d
RETURNING
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	ti, err := scanTokenIcon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tidom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, tidom.ErrConflict
		}
		return nil, err
	}
	return &ti, nil
}

func (r *TokenIconRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM token_icons WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return tidom.ErrNotFound
	}
	return nil
}

func (r *TokenIconRepositoryPG) UploadIcon(ctx context.Context, fileName, contentType string, _ io.Reader) (string, int64, error) {
	// Not handled by DB adapter. Implement in a storage adapter (e.g., S3/GCS).
	return "", 0, fmt.Errorf("UploadIcon: not implemented in PG repository")
}

func (r *TokenIconRepositoryPG) GetTokenIconStats(ctx context.Context) (tidom.TokenIconStats, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	var stats tidom.TokenIconStats

	// totals
	if err := run.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(size),0) FROM token_icons`).Scan(&stats.Total, &stats.TotalSize); err != nil {
		return tidom.TokenIconStats{}, err
	}
	if stats.Total > 0 {
		stats.AverageSize = float64(stats.TotalSize) / float64(stats.Total)
	}

	// largest
	{
		row := run.QueryRowContext(ctx, `
SELECT id, file_name, size
FROM token_icons
ORDER BY size DESC, id ASC
LIMIT 1`)
		var id, fn string
		var sz int64
		if err := row.Scan(&id, &fn, &sz); err == nil {
			stats.LargestIcon = &struct {
				ID       string
				FileName string
				Size     int64
			}{ID: strings.TrimSpace(id), FileName: strings.TrimSpace(fn), Size: sz}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return tidom.TokenIconStats{}, err
		}
	}

	// smallest
	{
		row := run.QueryRowContext(ctx, `
SELECT id, file_name, size
FROM token_icons
WHERE size IS NOT NULL
ORDER BY size ASC, id ASC
LIMIT 1`)
		var id, fn string
		var sz int64
		if err := row.Scan(&id, &fn, &sz); err == nil {
			stats.SmallestIcon = &struct {
				ID       string
				FileName string
				Size     int64
			}{ID: strings.TrimSpace(id), FileName: strings.TrimSpace(fn), Size: sz}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return tidom.TokenIconStats{}, err
		}
	}
	return stats, nil
}

func (r *TokenIconRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

func (r *TokenIconRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM token_icons`)
	return err
}

// ========================
// Compatibility methods
// ========================

func (r *TokenIconRepositoryPG) FetchAllTokenIcons(ctx context.Context) ([]*tidom.TokenIcon, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, url, file_name, size,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_icons
ORDER BY updated_at DESC, id DESC`
	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tidom.TokenIcon
	for rows.Next() {
		ti, err := scanTokenIcon(rows)
		if err != nil {
			return nil, err
		}
		tt := ti
		out = append(out, &tt)
	}
	return out, rows.Err()
}

func (r *TokenIconRepositoryPG) FetchTokenIconByID(ctx context.Context, iconID string) (*tidom.TokenIcon, error) {
	return r.GetByID(ctx, iconID)
}

func (r *TokenIconRepositoryPG) FetchTokenIconByBlueprintID(ctx context.Context, tokenBlueprintID string) (*tidom.TokenIcon, error) {
	// Not supported by current schema (token_icons has no token_blueprint_id).
	return nil, tidom.ErrNotFound
}

func (r *TokenIconRepositoryPG) CreateTokenIcon(ctx context.Context, in tidom.CreateTokenIconInput) (*tidom.TokenIcon, error) {
	return r.Create(ctx, in)
}

func (r *TokenIconRepositoryPG) UpdateTokenIcon(ctx context.Context, iconID string, updates tidom.UpdateTokenIconInput) (*tidom.TokenIcon, error) {
	return r.Update(ctx, iconID, updates)
}

// ========================
// Helpers
// ========================

func scanTokenIcon(s dbcommon.RowScanner) (tidom.TokenIcon, error) {
	var (
		id, url, fileName, createdBy, updatedBy string
		size                                    int64
		createdAt, updatedAt                    time.Time
		deletedAtNS                             sql.NullTime
		deletedByNS                             sql.NullString
	)
	if err := s.Scan(
		&id, &url, &fileName, &size,
		&createdAt, &createdBy, &updatedAt, &updatedBy, &deletedAtNS, &deletedByNS,
	); err != nil {
		return tidom.TokenIcon{}, err
	}

	// ドメインの TokenIcon は ID/URL/FileName/Size のみを持つため、
	// 監査系カラムは読み取りつつも返却値には含めません。
	return tidom.TokenIcon{
		ID:       strings.TrimSpace(id),
		URL:      strings.TrimSpace(url),
		FileName: strings.TrimSpace(fileName),
		Size:     size,
	}, nil
}

func buildTIWhere(f tidom.Filter) ([]string, []any) {
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

	if v := strings.TrimSpace(f.FileNameLike); v != "" {
		where = append(where, fmt.Sprintf("file_name ILIKE $%d", len(args)+1))
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

func buildTIOrderBy(s tidom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "size":
		col = "size"
	case "filename", "file_name":
		col = "file_name"
	default:
		// default (no explicit sort) -> rely on caller's fallback
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
