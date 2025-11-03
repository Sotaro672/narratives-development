package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbcommon "narratives/internal/adapters/out/db/common"
	common "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"

	"github.com/lib/pq"
)

// MemberRepositoryPG は PostgreSQL 実装です。
type MemberRepositoryPG struct {
	DB *sql.DB
}

func NewMemberRepositoryPG(db *sql.DB) *MemberRepositoryPG {
	return &MemberRepositoryPG{DB: db}
}

// ========================================
// List (filter + sort + pagination)
// ========================================
// List は共通ユーティリティ（NormalizePage/BuildOrderBy/QueryCount/ComputeTotalPages）を使用します。
func (r *MemberRepositoryPG) List(
	ctx context.Context,
	filter memdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[memdom.Member], error) {

	where, args := buildWhere(filter)

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// 許可ソート列のマッピング
	colKey, dir := normalizeSort(sort)
	orderBy := "ORDER BY created_at DESC"
	switch colKey {
	case "name":
		// 名字→名前で並べ替え（空は最後）
		orderBy = fmt.Sprintf(
			"ORDER BY COALESCE(last_name,'') %s, COALESCE(first_name,'') %s",
			dir, dir,
		)
	case "email":
		orderBy = fmt.Sprintf("ORDER BY email %s", dir)
	case "createdat", "joinedat":
		orderBy = fmt.Sprintf("ORDER BY created_at %s", dir)
	case "updatedat":
		orderBy = fmt.Sprintf("ORDER BY updated_at %s", dir)
	case "permissions":
		orderBy = fmt.Sprintf(
			"ORDER BY array_length(permissions, 1) %s NULLS LAST",
			dir,
		)
	}

	// ページング
	_, limit, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// 件数
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM members %s", whereSQL)
	total, err := dbcommon.QueryCount(ctx, r.DB, countSQL, args...)
	if err != nil {
		return common.PageResult[memdom.Member]{}, err
	}

	// 本体（id は text キャストで取得）
	q := fmt.Sprintf(`
SELECT
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
FROM members
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return common.PageResult[memdom.Member]{}, err
	}
	defer rows.Close()

	items := make([]memdom.Member, 0, limit)
	for rows.Next() {
		var m memdom.Member
		if err := scanMember(rows, &m); err != nil {
			return common.PageResult[memdom.Member]{}, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return common.PageResult[memdom.Member]{}, err
	}

	return common.PageResult[memdom.Member]{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, limit),
		Page:       page.Number,
		PerPage:    limit,
	}, nil
}

// ========================================
// ListByCursor (simple id-based cursor)
// ========================================
func (r *MemberRepositoryPG) ListByCursor(
	ctx context.Context,
	filter memdom.Filter,
	sort memdom.Sort,
	cpage memdom.CursorPage,
) (memdom.CursorPageResult, error) {

	where, args := buildWhere(filter)

	// 固定: id 昇順
	orderBy := "ORDER BY id ASC"

	// カーソル（id）より後を取得
	if strings.TrimSpace(cpage.After) != "" {
		where = append(where, fmt.Sprintf("id::text > $%d", len(args)+1))
		args = append(args, cpage.After)
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
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
FROM members
%s
%s
LIMIT $%d
`, whereSQL, orderBy, len(args)+1)

	args = append(args, limit+1) // 次ページ有無判定のため +1

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return memdom.CursorPageResult{}, err
	}
	defer rows.Close()

	var items []memdom.Member
	var lastID string
	for rows.Next() {
		var m memdom.Member
		if err := scanMember(rows, &m); err != nil {
			return memdom.CursorPageResult{}, err
		}
		items = append(items, m)
		lastID = m.ID
	}
	if err := rows.Err(); err != nil {
		return memdom.CursorPageResult{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return memdom.CursorPageResult{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ========================================
// GetByID
// ========================================
func (r *MemberRepositoryPG) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	const q = `
SELECT
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
FROM members
WHERE id::text = $1
`
	var m memdom.Member
	row := r.DB.QueryRowContext(ctx, q, id)
	if err := scanMember(row, &m); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memdom.Member{}, memdom.ErrNotFound
		}
		return memdom.Member{}, err
	}
	return m, nil
}

// ========================================
// GetByEmail
// ========================================
func (r *MemberRepositoryPG) GetByEmail(ctx context.Context, email string) (memdom.Member, error) {
	const q = `
SELECT
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
FROM members
WHERE email = $1
`
	var m memdom.Member
	row := r.DB.QueryRowContext(ctx, q, email)
	if err := scanMember(row, &m); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memdom.Member{}, memdom.ErrNotFound
		}
		return memdom.Member{}, err
	}
	return m, nil
}

// ========================================
// Exists
// ========================================
func (r *MemberRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM members WHERE id::text = $1`
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

// ========================================
// Count
// ========================================
func (r *MemberRepositoryPG) Count(ctx context.Context, filter memdom.Filter) (int, error) {
	where, args := buildWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members `+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ========================================
// Create
// ========================================
func (r *MemberRepositoryPG) Create(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	const q = `
INSERT INTO members (
  id,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
) VALUES (
  COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()),
  $2,$3,$4,$5,$6,$7,$8,NOW()
)
RETURNING
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
`
	var out memdom.Member
	err := r.DB.QueryRowContext(ctx, q,
		m.ID,
		m.FirstName,
		m.LastName,
		m.Email,
		m.Role,
		pq.Array(m.Permissions),
		pq.Array(m.AssignedBrands),
		m.CreatedAt,
	).Scan(
		&out.ID,
		&out.FirstName,
		&out.LastName,
		&out.Email,
		&out.Role,
		pq.Array(&out.Permissions),
		pq.Array(&out.AssignedBrands),
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return memdom.Member{}, memdom.ErrConflict
		}
		return memdom.Member{}, err
	}
	return out, nil
}

// ========================================
// Save (upsert)
// ========================================
func (r *MemberRepositoryPG) Save(ctx context.Context, m memdom.Member, _ *memdom.SaveOptions) (memdom.Member, error) {
	const q = `
INSERT INTO members (
  id,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
) VALUES (
  $1::uuid,$2,$3,$4,$5,$6,$7,$8,NOW()
)
ON CONFLICT (id) DO UPDATE SET
  first_name       = EXCLUDED.first_name,
  last_name        = EXCLUDED.last_name,
  email            = EXCLUDED.email,
  role             = EXCLUDED.role,
  permissions      = EXCLUDED.permissions,
  assigned_brands  = EXCLUDED.assigned_brands,
  created_at       = EXCLUDED.created_at,
  updated_at       = NOW()
RETURNING
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
`
	var out memdom.Member
	err := r.DB.QueryRowContext(ctx, q,
		m.ID,
		m.FirstName,
		m.LastName,
		m.Email,
		m.Role,
		pq.Array(m.Permissions),
		pq.Array(m.AssignedBrands),
		m.CreatedAt,
	).Scan(
		&out.ID,
		&out.FirstName,
		&out.LastName,
		&out.Email,
		&out.Role,
		pq.Array(&out.Permissions),
		pq.Array(&out.AssignedBrands),
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return memdom.Member{}, err
	}
	return out, nil
}

// ========================================
// Delete
// ========================================
func (r *MemberRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM members WHERE id::text = $1`,
		id,
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return memdom.ErrNotFound
	}
	return nil
}

// ========================================
// Reset (development/testing)
// ========================================
func (r *MemberRepositoryPG) Reset(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx,
		`TRUNCATE TABLE members RESTART IDENTITY CASCADE`,
	)
	return err
}

// ========================================
// Scan Helper
// ========================================
// scanMember は共通の RowScanner を使用します。
func scanMember(s dbcommon.RowScanner, m *memdom.Member) error {
	var (
		id, firstName, lastName, email, role sql.NullString
		perms, brands                        pq.StringArray
		createdAt, updatedAt                 sql.NullTime
	)

	if err := s.Scan(
		&id,
		&firstName,
		&lastName,
		&email,
		&role,
		&perms,
		&brands,
		&createdAt,
		&updatedAt,
	); err != nil {
		return err
	}

	m.ID = id.String
	m.FirstName = firstName.String
	m.LastName = lastName.String
	m.Email = email.String
	m.Role = memdom.MemberRole(role.String)
	m.Permissions = perms
	m.AssignedBrands = brands
	if createdAt.Valid {
		m.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		m.UpdatedAt = &t
	} else {
		m.UpdatedAt = nil
	}
	return nil
}

// 例: ソート指定の正規化
func normalizeSort(sort common.Sort) (column string, direction string) {
	column = dbcommon.ToLowerString(sort.Column)
	direction = dbcommon.ToUpperString(sort.Order)
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC"
	}
	return
}

// 内部: WHERE 句の組み立て
func buildWhere(f memdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}
	i := 1

	// SearchQuery: 部分一致（first/last/kana/email）
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(
			where,
			fmt.Sprintf(
				"(first_name ILIKE $%d OR last_name ILIKE $%d OR first_name_kana ILIKE $%d OR last_name_kana ILIKE $%d OR email ILIKE $%d)",
				i, i, i, i, i,
			),
		)
		like := "%" + sq + "%"
		args = append(args, like)
		i++
	}

	// Role filter (RoleIDs or Roles)
	if len(f.RoleIDs) > 0 || len(f.Roles) > 0 {
		roles := append([]string{}, f.RoleIDs...)
		roles = append(roles, f.Roles...)
		if len(roles) > 0 {
			where = append(where, fmt.Sprintf("role = ANY($%d)", i))
			args = append(args, pq.Array(roles))
			i++
		}
	}

	// BrandIDs/Brands: assigned_brands と重なりがあれば対象
	if len(f.BrandIDs) > 0 || len(f.Brands) > 0 {
		brands := append([]string{}, f.BrandIDs...)
		brands = append(brands, f.Brands...)
		if len(brands) > 0 {
			where = append(where, fmt.Sprintf("assigned_brands && $%d", i))
			args = append(args, pq.Array(brands))
			i++
		}
	}

	// Permissions: 重なりがあれば対象
	if len(f.Permissions) > 0 {
		where = append(where, fmt.Sprintf("permissions && $%d", i))
		args = append(args, pq.Array(f.Permissions))
		i++
	}

	// 期間フィルタ
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *f.CreatedFrom)
		i++
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", i))
		args = append(args, *f.CreatedTo)
		i++
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("updated_at >= $%d", i))
		args = append(args, *f.UpdatedFrom)
		i++
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("updated_at < $%d", i))
		args = append(args, *f.UpdatedTo)
		i++
	}

	return where, args
}

// ========================================
// Update (partial update / patch)
// ========================================
func (r *MemberRepositoryPG) Update(
	ctx context.Context,
	id string,
	patch memdom.MemberPatch,
) (memdom.Member, error) {

	sets := []string{}
	args := []any{}
	i := 1

	// first_name
	if patch.FirstName != nil {
		sets = append(sets, fmt.Sprintf("first_name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.FirstName))
		i++
	}

	// last_name
	if patch.LastName != nil {
		sets = append(sets, fmt.Sprintf("last_name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.LastName))
		i++
	}

	// email
	if patch.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Email))
		i++
	}

	// role
	if patch.Role != nil {
		sets = append(sets, fmt.Sprintf("role = $%d", i))
		args = append(args, memdom.MemberRole(strings.TrimSpace(*patch.Role)))
		i++
	}

	// permissions
	if patch.Permissions != nil {
		sets = append(sets, fmt.Sprintf("permissions = $%d", i))
		args = append(args, pq.Array(dedupTrimStrings(*patch.Permissions)))
		i++
	}

	// assigned_brands
	if patch.AssignedBrands != nil {
		sets = append(sets, fmt.Sprintf("assigned_brands = $%d", i))
		args = append(args, pq.Array(dedupTrimStrings(*patch.AssignedBrands)))
		i++
	}

	// updated_at は NOW() にする（明示指定がある場合はそれを使う）
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, "updated_at = NOW()")
	}

	// もし何も変更指定がなければ現在のレコードをそのまま返す
	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	// WHERE句のために id を最後に追加
	args = append(args, strings.TrimSpace(id))

	q := fmt.Sprintf(`
UPDATE members
SET %s
WHERE id::text = $%d
RETURNING
  id::text,
  first_name,
  last_name,
  email,
  role,
  permissions,
  assigned_brands,
  created_at,
  updated_at
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)

	var out memdom.Member
	if err := scanMember(row, &out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memdom.Member{}, memdom.ErrNotFound
		}
		return memdom.Member{}, err
	}

	return out, nil
}
