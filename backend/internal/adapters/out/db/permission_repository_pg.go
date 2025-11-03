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

// ============================================================
// Port (usecase.PermissionRepo) 実装
// ============================================================

// GetByID implements PermissionRepo.GetByID.
func (r *PermissionRepositoryPG) GetByID(ctx context.Context, id string) (permission.Permission, error) {
	const q = `
SELECT
  id,
  name,
  category,
  description,
  created_at
FROM permissions
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(id))

	p, err := scanPermission(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}
	return p, nil
}

// Exists implements PermissionRepo.Exists.
func (r *PermissionRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM permissions WHERE id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create implements PermissionRepo.Create.
//
// v.ID が空なら DB 側で gen_random_uuid()::text に任せる。
func (r *PermissionRepositoryPG) Create(ctx context.Context, v permission.Permission) (permission.Permission, error) {
	const q = `
INSERT INTO permissions (
  id,
  name,
  category,
  description,
  created_at,
  updated_at
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  NOW(),
  NOW()
)
RETURNING
  id,
  name,
  category,
  description,
  created_at
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.Name),
		strings.TrimSpace(string(v.Category)),
		strings.TrimSpace(v.Description),
	)

	p, err := scanPermission(row)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return permission.Permission{}, permission.ErrConflict
		}
		return permission.Permission{}, err
	}
	return p, nil
}

// Save implements PermissionRepo.Save.
//
// upsert動作: INSERT ... ON CONFLICT(id) DO UPDATE
// created_at は古い方を維持する / updated_at は NOW() で更新。
func (r *PermissionRepositoryPG) Save(ctx context.Context, v permission.Permission) (permission.Permission, error) {
	const q = `
INSERT INTO permissions (
  id,
  name,
  category,
  description,
  created_at,
  updated_at
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  NOW(),
  NOW()
)
ON CONFLICT (id) DO UPDATE SET
  name        = EXCLUDED.name,
  category    = EXCLUDED.category,
  description = EXCLUDED.description,
  created_at  = LEAST(permissions.created_at, EXCLUDED.created_at),
  updated_at  = NOW()
RETURNING
  id,
  name,
  category,
  description,
  created_at
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.Name),
		strings.TrimSpace(string(v.Category)),
		strings.TrimSpace(v.Description),
	)

	p, err := scanPermission(row)
	if err != nil {
		return permission.Permission{}, err
	}
	return p, nil
}

// Delete implements PermissionRepo.Delete.
func (r *PermissionRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM permissions WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return permission.ErrNotFound
	}
	return nil
}

// ============================================================
// 追加のクエリ系 (List / Update 等)
// これらは usecase.PermissionRepo の必須ではないが便利なので保持
// ============================================================

// List はフィルタ+ソート+ページング。
func (r *PermissionRepositoryPG) List(
	ctx context.Context,
	filter permission.Filter,
	sort permission.Sort,
	page permission.Page,
) (permission.PageResult[permission.Permission], error) {

	where := []string{}
	args := []any{}
	i := 1

	// 部分一致検索 (id, name, description)
	if sq := strings.TrimSpace(filter.SearchQuery); sq != "" {
		where = append(where,
			fmt.Sprintf("(id ILIKE $%d OR name ILIKE $%d OR description ILIKE $%d)", i, i+1, i+2),
		)
		like := "%" + sq + "%"
		args = append(args, like, like, like)
		i += 3
	}

	// category IN (...)
	if len(filter.Categories) > 0 {
		where = append(where, fmt.Sprintf("category = ANY($%d)", i))
		args = append(args, pq.Array(catToStrings(filter.Categories)))
		i++
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// ソート
	orderBy := "ORDER BY created_at DESC"
	if col := strings.TrimSpace(string(sort.Column)); col != "" {
		dir := strings.ToUpper(string(sort.Order))
		if dir != "ASC" && dir != "DESC" {
			dir = "ASC"
		}
		switch col {
		case "name", "category", "createdAt", "created_at":
			if col == "createdAt" {
				col = "created_at"
			}
			orderBy = fmt.Sprintf("ORDER BY %s %s", col, dir)
		default:
			orderBy = "ORDER BY created_at DESC"
		}
	}

	// ページネーション
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

	// 本体
	q := fmt.Sprintf(`
SELECT
  id,
  name,
  category,
  description,
  created_at
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
		p, err := scanPermission(rows)
		if err != nil {
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

// Update は部分更新用（usecase.PermissionRepo には含めていない）
func (r *PermissionRepositoryPG) Update(ctx context.Context, id string, patch permission.PermissionPatch) (permission.Permission, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Name))
		i++
	}
	if patch.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", i))
		args = append(args, strings.TrimSpace(string(*patch.Category)))
		i++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Description))
		i++
	}

	if len(sets) == 0 {
		// 変更なしでも最新を返す
		return r.GetByID(ctx, id)
	}

	// updated_at を必ず NOW() にする
	sets = append(sets, fmt.Sprintf("updated_at = NOW()"))

	args = append(args, strings.TrimSpace(id))

	q := fmt.Sprintf(`
UPDATE permissions
SET %s
WHERE id = $%d
RETURNING
  id,
  name,
  category,
  description,
  created_at
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)

	out, err := scanPermission(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}
	return out, nil
}

// ============================================================
// ヘルパー
// ============================================================

func scanPermission(s common.RowScanner) (permission.Permission, error) {
	var (
		idNS          sql.NullString
		nameNS        sql.NullString
		categoryNS    sql.NullString
		descriptionNS sql.NullString
		createdAtNS   sql.NullTime
	)

	if err := s.Scan(
		&idNS,
		&nameNS,
		&categoryNS,
		&descriptionNS,
		&createdAtNS,
	); err != nil {
		return permission.Permission{}, err
	}

	out := permission.Permission{
		ID:          strings.TrimSpace(idNS.String),
		Name:        strings.TrimSpace(nameNS.String),
		Category:    permission.PermissionCategory(strings.TrimSpace(categoryNS.String)),
		Description: strings.TrimSpace(descriptionNS.String),
		// Permission ドメインに CreatedAt フィールドが無い前提なので何もしない
	}

	// createdAtNS はDB上は保持しているが、
	// domain.Permission に CreatedAt が無いので代入しない。
	_ = createdAtNS

	return out, nil
}

func catToStrings(cs []permission.PermissionCategory) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, strings.TrimSpace(string(c)))
	}
	return out
}
