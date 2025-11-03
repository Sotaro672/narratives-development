package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	common "narratives/internal/adapters/out/db/common"
	permission "narratives/internal/domain/permission"

	"github.com/lib/pq"
)

type PermissionRepositoryPG struct {
	DB *sql.DB
}

func NewPermissionRepositoryPG(db *sql.DB) *PermissionRepositoryPG {
	return &PermissionRepositoryPG{DB: db}
}

// ==============================
// List (Filter + Sort + Page)
// ==============================
func (r *PermissionRepositoryPG) List(
	ctx context.Context,
	filter permission.Filter,
	sort permission.Sort,
	page permission.Page,
) (permission.PageResult[permission.Permission], error) {

	where := []string{}
	args := []any{}
	i := 1

	// SearchQuery: id / name / description の部分一致
	if sq := strings.TrimSpace(filter.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(id ILIKE $%d OR name ILIKE $%d OR description ILIKE $%d)", i, i+1, i+2))
		like := "%" + sq + "%"
		args = append(args, like, like, like)
		i += 3
	}

	// Categories: = ANY($n)
	if len(filter.Categories) > 0 {
		where = append(where, fmt.Sprintf("category = ANY($%d)", i))
		args = append(args, pq.Array(catToStrings(filter.Categories)))
		i++
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// Sort
	orderBy := "ORDER BY created_at DESC" // デフォルト
	if col := strings.TrimSpace(string(sort.Column)); col != "" {
		order := strings.ToUpper(string(sort.Order))
		if order != "ASC" && order != "DESC" {
			order = "ASC"
		}
		switch col {
		case "name", "category", "createdAt", "created_at":
			if col == "createdAt" {
				col = "created_at"
			}
			orderBy = fmt.Sprintf("ORDER BY %s %s", col, order)
		default:
			orderBy = "ORDER BY created_at DESC"
		}
	}

	// Page
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage

	// COUNT
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM permissions %s`, whereSQL)
	var total int
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return permission.PageResult[permission.Permission]{}, err
	}

	// Query
	q := fmt.Sprintf(`
SELECT id, name, category, description, created_at
FROM permissions
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, i, i+1)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return permission.PageResult[permission.Permission]{}, err
	}
	defer rows.Close()

	items := make([]permission.Permission, 0, perPage)
	for rows.Next() {
		var p permission.Permission
		if err := scanPermission(rows, &p); err != nil {
			return permission.PageResult[permission.Permission]{}, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return permission.PageResult[permission.Permission]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return permission.PageResult[permission.Permission]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ==============================
// GetByID
// ==============================
func (r *PermissionRepositoryPG) GetByID(ctx context.Context, id string) (permission.Permission, error) {
	const q = `
SELECT id, name, category, description, created_at
FROM permissions
WHERE id = $1
`
	var p permission.Permission
	row := r.DB.QueryRowContext(ctx, q, id)
	if err := scanPermission(row, &p); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}
	return p, nil
}

// ==============================
// Create
// ==============================
func (r *PermissionRepositoryPG) Create(ctx context.Context, p permission.Permission) (permission.Permission, error) {
	// id が空なら DB 側で生成（gen_random_uuid）想定
	const q = `
INSERT INTO permissions (id, name, category, description, created_at, updated_at)
VALUES (
	COALESCE(NULLIF($1,''), gen_random_uuid()::text),
	$2, $3, $4, NOW(), NOW()
)
RETURNING id, name, category, description, created_at
`
	var out permission.Permission
	if err := r.DB.QueryRowContext(ctx, q,
		p.ID,
		p.Name,
		p.Category,
		p.Description,
	).Scan(
		&out.ID, &out.Name, &out.Category, &out.Description, new(sql.NullTime),
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return permission.Permission{}, permission.ErrConflict
		}
		return permission.Permission{}, err
	}
	return out, nil
}

// ==============================
// Update (PATCH)
// ==============================
func (r *PermissionRepositoryPG) Update(ctx context.Context, id string, patch permission.PermissionPatch) (permission.Permission, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", i))
		args = append(args, *patch.Name)
		i++
	}
	if patch.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", i))
		args = append(args, *patch.Category)
		i++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", i))
		args = append(args, *patch.Description)
		i++
	}

	if len(sets) == 0 {
		// 変更なしでも最新を返す
		return r.GetByID(ctx, id)
	}

	q := fmt.Sprintf(`
UPDATE permissions
SET %s
WHERE id = $%d
RETURNING id, name, category, description, created_at
`, strings.Join(sets, ", "), i)
	args = append(args, id)

	var out permission.Permission
	if err := r.DB.QueryRowContext(ctx, q, args...).Scan(
		&out.ID, &out.Name, &out.Category, &out.Description, new(sql.NullTime),
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}
	return out, nil
}

// ==============================
// Delete
// ==============================
func (r *PermissionRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM permissions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return permission.ErrNotFound
	}
	return nil
}

// ==============================
// scan helper
// ==============================
func scanPermission(s common.RowScanner, p *permission.Permission) error {
	var (
		id, name, category, description sql.NullString
		createdAt                       sql.NullTime
	)
	if err := s.Scan(&id, &name, &category, &description, &createdAt); err != nil {
		return err
	}
	p.ID = id.String
	p.Name = name.String
	p.Category = permission.PermissionCategory(category.String)
	p.Description = description.String
	return nil
}

func catToStrings(cs []permission.PermissionCategory) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, string(c))
	}
	return out
}
