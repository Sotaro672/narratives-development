// backend\internal\adapters\out\firestore\tokenBlueprint_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lib/pq"

	dbcommon "narratives/internal/adapters/out/db/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintRepositoryPG struct {
	DB *sql.DB
}

func NewTokenBlueprintRepositoryPG(db *sql.DB) *TokenBlueprintRepositoryPG {
	return &TokenBlueprintRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *TokenBlueprintRepositoryPG) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, name, symbol, brand_id, description, icon_id, content_files,
  assignee_id, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_blueprints
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	tb, err := scanTokenBlueprint(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tbdom.ErrNotFound
		}
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBlueprintRepositoryPG) List(ctx context.Context, filter tbdom.Filter, sort tbdom.Sort, page tbdom.Page) (tbdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTBWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildTBOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_blueprints "+whereSQL, args...).Scan(&total); err != nil {
		return tbdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, name, symbol, brand_id, description, icon_id, content_files,
  assignee_id, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM token_blueprints
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return tbdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]tbdom.TokenBlueprint, 0, perPage)
	for rows.Next() {
		tb, err := scanTokenBlueprint(rows)
		if err != nil {
			return tbdom.PageResult{}, err
		}
		items = append(items, tb)
	}
	if err := rows.Err(); err != nil {
		return tbdom.PageResult{}, err
	}

	return tbdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryPG) Count(ctx context.Context, filter tbdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTBWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_blueprints "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TokenBlueprintRepositoryPG) Create(ctx context.Context, in tbdom.CreateTokenBlueprintInput) (*tbdom.TokenBlueprint, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	files := sanitizeStrings(in.ContentFiles)

	// icon_id: nil または空なら NULL
	var iconArg any
	if in.IconID != nil {
		if v := strings.TrimSpace(*in.IconID); v != "" {
			iconArg = v
		} else {
			iconArg = nil
		}
	}

	const q = `
INSERT INTO token_blueprints (
  id, name, symbol, brand_id, description, icon_id, content_files,
  assignee_id, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5, $6,
  $7, COALESCE($8, NOW()), $9, COALESCE($10, NOW()), $11, NULL, NULL
)
RETURNING
  id, name, symbol, brand_id, description, icon_id, content_files,
  assignee_id, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Symbol),
		strings.TrimSpace(in.BrandID),
		strings.TrimSpace(in.Description),
		iconArg,
		pq.Array(files),
		strings.TrimSpace(in.AssigneeID),
		dbcommon.ToDBTime(in.CreatedAt),
		strings.TrimSpace(in.CreatedBy),
		dbcommon.ToDBTime(in.UpdatedAt),
		strings.TrimSpace(in.UpdatedBy),
	)
	tb, err := scanTokenBlueprint(row)
	if err != nil {
		// Note: ErrConflict will surface only if a UNIQUE constraint exists (e.g., on symbol)
		if dbcommon.IsUniqueViolation(err) {
			return nil, tbdom.ErrConflict
		}
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBlueprintRepositoryPG) Update(ctx context.Context, id string, in tbdom.UpdateTokenBlueprintInput) (*tbdom.TokenBlueprint, error) {
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

	setStr("name", in.Name)
	setStr("symbol", in.Symbol)
	setStr("brand_id", in.BrandID)
	setStr("description", in.Description)
	setStr("assignee_id", in.AssigneeID)

	// icon_id（空文字なら NULL へ）
	if in.IconID != nil {
		v := strings.TrimSpace(*in.IconID)
		if v == "" {
			sets = append(sets, "icon_id = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("icon_id = $%d", i))
			args = append(args, v)
			i++
		}
	}

	// content_files
	if in.ContentFiles != nil {
		files := sanitizeStrings(*in.ContentFiles)
		sets = append(sets, fmt.Sprintf("content_files = $%d", i))
		args = append(args, pq.Array(files))
		i++
	}

	// updated_at
	if in.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, in.UpdatedAt.UTC())
		i++
	} else {
		sets = append(sets, "updated_at = NOW()")
	}

	// updated_by
	if in.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, strings.TrimSpace(*in.UpdatedBy))
		i++
	}

	// deleted_at / deleted_by（NULL 設定可）
	if in.DeletedAt != nil {
		sets = append(sets, fmt.Sprintf("deleted_at = $%d", i))
		args = append(args, in.DeletedAt.UTC())
		i++
	}
	if in.DeletedBy != nil {
		v := strings.TrimSpace(*in.DeletedBy)
		if v == "" {
			sets = append(sets, "deleted_by = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("deleted_by = $%d", i))
			args = append(args, v)
			i++
		}
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE token_blueprints
SET %s
WHERE id = $%d
RETURNING
  id, name, symbol, brand_id, description, icon_id, content_files,
  assignee_id, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	tb, err := scanTokenBlueprint(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tbdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, tbdom.ErrConflict
		}
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBlueprintRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM token_blueprints WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return tbdom.ErrNotFound
	}
	return nil
}

func (r *TokenBlueprintRepositoryPG) IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT 1
FROM token_blueprints
WHERE symbol = $1 AND ($2 = '' OR id <> $2)
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(symbol), strings.TrimSpace(excludeID))
	var dummy int
	err := row.Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *TokenBlueprintRepositoryPG) IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT 1
FROM token_blueprints
WHERE name = $1 AND ($2 = '' OR id <> $2)
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(name), strings.TrimSpace(excludeID))
	var dummy int
	err := row.Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *TokenBlueprintRepositoryPG) UploadIcon(ctx context.Context, fileName, contentType string, _ io.Reader) (string, error) {
	// Not handled by DB adapter. Implement in a storage adapter (e.g., S3/GCS).
	return "", fmt.Errorf("UploadIcon: not implemented in PG repository")
}

func (r *TokenBlueprintRepositoryPG) UploadContentFile(ctx context.Context, fileName, contentType string, _ io.Reader) (string, error) {
	// Not handled by DB adapter. Implement in a storage adapter (e.g., S3/GCS).
	return "", fmt.Errorf("UploadContentFile: not implemented in PG repository")
}

func (r *TokenBlueprintRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

func (r *TokenBlueprintRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM token_blueprints`)
	return err
}

// ========================
// Helpers
// ========================

func scanTokenBlueprint(s dbcommon.RowScanner) (tbdom.TokenBlueprint, error) {
	var (
		id, name, symbol, brandID, description string
		iconNS                                 sql.NullString
		contentFiles                           []string
		assigneeID                             string
		createdBy, updatedByNS                 sql.NullString
		createdAt, updatedAt                   time.Time
		deletedAtNS                            sql.NullTime
		deletedByNS                            sql.NullString
	)
	if err := s.Scan(
		&id, &name, &symbol, &brandID, &description, &iconNS, pq.Array(&contentFiles),
		&assigneeID, &createdAt, &createdBy, &updatedAt, &updatedByNS, &deletedAtNS, &deletedByNS,
	); err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	var icon *string
	if iconNS.Valid {
		v := strings.TrimSpace(iconNS.String)
		if v != "" {
			icon = &v
		}
	}
	var delAt *time.Time
	if deletedAtNS.Valid {
		t := deletedAtNS.Time.UTC()
		delAt = &t
	}
	var delBy *string
	if deletedByNS.Valid {
		v := strings.TrimSpace(deletedByNS.String)
		if v != "" {
			delBy = &v
		}
	}
	updatedBy := ""
	if updatedByNS.Valid {
		updatedBy = strings.TrimSpace(updatedByNS.String)
	}
	return tbdom.TokenBlueprint{
		ID:           strings.TrimSpace(id),
		Name:         strings.TrimSpace(name),
		Symbol:       strings.TrimSpace(symbol),
		BrandID:      strings.TrimSpace(brandID),
		Description:  strings.TrimSpace(description),
		IconID:       icon,
		ContentFiles: contentFiles,
		AssigneeID:   strings.TrimSpace(assigneeID),
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    strings.TrimSpace(createdBy.String),
		UpdatedAt:    updatedAt.UTC(),
		UpdatedBy:    updatedBy,
		DeletedAt:    delAt,
		DeletedBy:    delBy,
	}, nil
}

func buildTBWhere(f tbdom.Filter) ([]string, []any) {
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
	addIn("brand_id", f.BrandIDs)
	addIn("assignee_id", f.AssigneeIDs)
	addIn("symbol", f.Symbols)

	if v := strings.TrimSpace(f.NameLike); v != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}
	if v := strings.TrimSpace(f.SymbolLike); v != "" {
		where = append(where, fmt.Sprintf("symbol ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}
	if f.HasIcon != nil {
		if *f.HasIcon {
			where = append(where, "(icon_id IS NOT NULL AND btrim(icon_id) <> '')")
		} else {
			where = append(where, "(icon_id IS NULL OR btrim(icon_id) = '')")
		}
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
		where = append(where, fmt.Sprintf("updated_at >= $%d", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("updated_at < $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}

	return where, args
}

func buildTBOrderBy(s tbdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "name":
		col = "name"
	case "symbol":
		col = "symbol"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

func sanitizeStrings(xs []string) []string {
	out := make([]string, 0, len(xs))
	seen := make(map[string]struct{}, len(xs))
	for _, v := range xs {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
