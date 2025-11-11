package gcs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	avicon "narratives/internal/domain/avatarIcon"
)

// AvatarIconRepositoryPG implements avatarIcon.Repository with PostgreSQL.
type AvatarIconRepositoryPG struct {
	DB *sql.DB
}

func NewAvatarIconRepositoryPG(db *sql.DB) *AvatarIconRepositoryPG {
	return &AvatarIconRepositoryPG{DB: db}
}

// List returns paginated results with filter and sort.
func (r *AvatarIconRepositoryPG) List(ctx context.Context, filter avicon.Filter, sort avicon.Sort, page avicon.Page) (avicon.PageResult[avicon.AvatarIcon], error) {
	where, args := buildAvatarIconWhere(filter)

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildAvatarIconOrderBy(sort)
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM avatar_icons %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return avicon.PageResult[avicon.AvatarIcon]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, avatar_id, url, file_name, size
FROM avatar_icons
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avicon.PageResult[avicon.AvatarIcon]{}, err
	}
	defer rows.Close()

	var items []avicon.AvatarIcon
	for rows.Next() {
		a, err := scanAvatarIcon(rows)
		if err != nil {
			return avicon.PageResult[avicon.AvatarIcon]{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return avicon.PageResult[avicon.AvatarIcon]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return avicon.PageResult[avicon.AvatarIcon]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ListByCursor returns cursor-based page. Order by id ASC, use id > after.
func (r *AvatarIconRepositoryPG) ListByCursor(ctx context.Context, filter avicon.Filter, _ avicon.Sort, cpage avicon.CursorPage) (avicon.CursorPageResult[avicon.AvatarIcon], error) {
	where, args := buildAvatarIconWhere(filter)
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
  id, avatar_id, url, file_name, size
FROM avatar_icons
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1) // +1 to detect next page

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avicon.CursorPageResult[avicon.AvatarIcon]{}, err
	}
	defer rows.Close()

	var items []avicon.AvatarIcon
	var lastID string
	for rows.Next() {
		a, err := scanAvatarIcon(rows)
		if err != nil {
			return avicon.CursorPageResult[avicon.AvatarIcon]{}, err
		}
		items = append(items, a)
		lastID = a.ID
	}
	if err := rows.Err(); err != nil {
		return avicon.CursorPageResult[avicon.AvatarIcon]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return avicon.CursorPageResult[avicon.AvatarIcon]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// GetByID fetches a single avatar icon by id.
func (r *AvatarIconRepositoryPG) GetByID(ctx context.Context, id string) (avicon.AvatarIcon, error) {
	const q = `
SELECT
  id, avatar_id, url, file_name, size
FROM avatar_icons
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	a, err := scanAvatarIcon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, err
	}
	return a, nil
}

// GetByAvatarID fetches all icons linked to an avatar.
func (r *AvatarIconRepositoryPG) GetByAvatarID(ctx context.Context, avatarID string) ([]avicon.AvatarIcon, error) {
	const q = `
SELECT
  id, avatar_id, url, file_name, size
FROM avatar_icons
WHERE avatar_id = $1
ORDER BY id DESC
`
	rows, err := r.DB.QueryContext(ctx, q, avatarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []avicon.AvatarIcon
	for rows.Next() {
		a, err := scanAvatarIcon(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// Create inserts a new avatar icon. Returns ErrConflict on unique violation.
func (r *AvatarIconRepositoryPG) Create(ctx context.Context, a avicon.AvatarIcon) (avicon.AvatarIcon, error) {
	const q = `
INSERT INTO avatar_icons (
  id, avatar_id, url, file_name, size
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING
  id, avatar_id, url, file_name, size
`
	var (
		avatarID any = toDBText(a.AvatarID)
		fileName any = toDBText(a.FileName)
		sizeVal  any = toDBInt64(a.Size)
	)

	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(a.ID), avatarID, strings.TrimSpace(a.URL), fileName, sizeVal,
	)
	out, err := scanAvatarIcon(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return avicon.AvatarIcon{}, avicon.ErrConflict
		}
		return avicon.AvatarIcon{}, err
	}
	return out, nil
}

// Update applies a partial update by id.
func (r *AvatarIconRepositoryPG) Update(ctx context.Context, id string, patch avicon.AvatarIconPatch) (avicon.AvatarIcon, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.AvatarID != nil {
		sets = append(sets, fmt.Sprintf("avatar_id = $%d", i))
		args = append(args, toDBText(patch.AvatarID))
		i++
	}
	if patch.URL != nil {
		sets = append(sets, fmt.Sprintf("url = $%d", i))
		args = append(args, strings.TrimSpace(*patch.URL))
		i++
	}
	if patch.FileName != nil {
		sets = append(sets, fmt.Sprintf("file_name = $%d", i))
		args = append(args, toDBText(patch.FileName))
		i++
	}
	if patch.Size != nil {
		sets = append(sets, fmt.Sprintf("size = $%d", i))
		args = append(args, toDBInt64(patch.Size))
		i++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
UPDATE avatar_icons
SET %s
WHERE id = $%d
RETURNING
  id, avatar_id, url, file_name, size
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanAvatarIcon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, err
	}
	return out, nil
}

// Delete removes a record by id.
func (r *AvatarIconRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM avatar_icons WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return avicon.ErrNotFound
	}
	return nil
}

// Count returns count for the given filter.
func (r *AvatarIconRepositoryPG) Count(ctx context.Context, filter avicon.Filter) (int, error) {
	where, args := buildAvatarIconWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM avatar_icons `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Save performs upsert on id. (opts is currently unused)
func (r *AvatarIconRepositoryPG) Save(
	ctx context.Context,
	a avicon.AvatarIcon,
	opts *avicon.SaveOptions,
) (avicon.AvatarIcon, error) {

	_ = opts // unused for now

	const q = `
INSERT INTO avatar_icons (
  id, avatar_id, url, file_name, size
) VALUES (
  $1,$2,$3,$4,$5
)
ON CONFLICT (id) DO UPDATE SET
  avatar_id = EXCLUDED.avatar_id,
  url       = EXCLUDED.url,
  file_name = EXCLUDED.file_name,
  size      = EXCLUDED.size
RETURNING
  id, avatar_id, url, file_name, size
`

	var (
		avatarID any = toDBText(a.AvatarID)
		fileName any = toDBText(a.FileName)
		sizeVal  any = toDBInt64(a.Size)
	)

	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(a.ID),
		avatarID,
		strings.TrimSpace(a.URL),
		fileName,
		sizeVal,
	)

	out, err := scanAvatarIcon(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return avicon.AvatarIcon{}, avicon.ErrConflict
		}
		return avicon.AvatarIcon{}, err
	}
	return out, nil
}

// ========== Helpers ==========

func scanAvatarIcon(s dbcommon.RowScanner) (avicon.AvatarIcon, error) {
	var (
		idNS, avatarIDNS, urlNS, fileNameNS sql.NullString
		sizeNI64                            sql.NullInt64
	)

	if err := s.Scan(
		&idNS, &avatarIDNS, &urlNS, &fileNameNS, &sizeNI64,
	); err != nil {
		return avicon.AvatarIcon{}, err
	}

	var (
		avatarIDPtr, fileNamePtr *string
		sizePtr                  *int64
	)

	if avatarIDNS.Valid {
		v := strings.TrimSpace(avatarIDNS.String)
		if v != "" {
			avatarIDPtr = &v
		}
	}
	if fileNameNS.Valid {
		v := strings.TrimSpace(fileNameNS.String)
		if v != "" {
			fileNamePtr = &v
		}
	}
	if sizeNI64.Valid {
		v := sizeNI64.Int64
		sizePtr = &v
	}

	return avicon.AvatarIcon{
		ID:       strings.TrimSpace(idNS.String),
		AvatarID: avatarIDPtr,
		URL:      strings.TrimSpace(urlNS.String),
		FileName: fileNamePtr,
		Size:     sizePtr,
	}, nil
}

func buildAvatarIconWhere(f avicon.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(id ILIKE $%d OR url ILIKE $%d OR file_name ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1))
		args = append(args, "%"+sq+"%")
	}

	if f.AvatarID != nil {
		where = append(where, fmt.Sprintf("avatar_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.AvatarID))
	}

	if f.HasAvatarID != nil {
		if *f.HasAvatarID {
			where = append(where, "avatar_id IS NOT NULL")
		} else {
			where = append(where, "avatar_id IS NULL")
		}
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

func buildAvatarIconOrderBy(sort avicon.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "id"
	case "size":
		col = "size"
	case "filename", "file_name":
		col = "file_name"
	case "url":
		col = "url"
	default:
		return ""
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}

func toDBText(p *string) any {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}

func toDBInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
