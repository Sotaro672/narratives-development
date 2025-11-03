package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	compdom "narratives/internal/domain/company"
	common "narratives/internal/domain/common"
)

// CompanyRepositoryPG implements the company.RepositoryPort interface
type CompanyRepositoryPG struct {
	DB *sql.DB
}

func NewCompanyRepositoryPG(db *sql.DB) *CompanyRepositoryPG {
	return &CompanyRepositoryPG{DB: db}
}

// ==============================
// Repository implementation
// ==============================

func (r *CompanyRepositoryPG) List(
	ctx context.Context,
	filter compdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[compdom.Company], error) {
	where, args := buildCompanyWhere(filter)
	order := buildOrderClause(sort)
	limit, offset := getLimitOffset(page)

	q := `
SELECT
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM companies
` + where + `
` + order + `
LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return common.PageResult[compdom.Company]{}, err
	}
	defer rows.Close()

	items := make([]compdom.Company, 0, limit)
	for rows.Next() {
		var c compdom.Company
		if err := scanCompany(rows, &c); err != nil {
			return common.PageResult[compdom.Company]{}, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return common.PageResult[compdom.Company]{}, err
	}

	// total count (best-effort)
	total, _ := r.Count(ctx, filter)

	var pr common.PageResult[compdom.Company]
	setPageResultCompanies(&pr, items, total)
	return pr, nil
}

func (r *CompanyRepositoryPG) ListByCursor(
	ctx context.Context,
	filter compdom.Filter,
	sort common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[compdom.Company], error) {
	// Keyset on created_at, id (DESC by default unless specified)
	where, args := buildCompanyWhere(filter)

	order := buildOrderClause(sort)
	limit := getCursorLimit(cpage)

	after, hasAfter := getCursorString(cpage)
	if hasAfter {
		// Expect cursor formatted as RFC3339|id
		ct, id, ok := parseSimpleCursor(after)
		if ok {
			// Use strict keyset matching with order direction (default DESC)
			desc := isDesc(sort)
			if desc {
				where = addAnd(where, "(created_at, id) < ($%d, $%d)", len(args)+1, len(args)+2)
			} else {
				where = addAnd(where, "(created_at, id) > ($%d, $%d)", len(args)+1, len(args)+2)
			}
			args = append(args, ct, id)
		}
	}

	q := `
SELECT
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM companies
` + where + `
` + order + `
LIMIT $%d`
	q = fmt.Sprintf(q, len(args)+1)
	args = append(args, limit+1) // fetch one more to detect next cursor

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return common.CursorPageResult[compdom.Company]{}, err
	}
	defer rows.Close()

	tmp := make([]compdom.Company, 0, limit+1)
	for rows.Next() {
		var c compdom.Company
		if err := scanCompany(rows, &c); err != nil {
			return common.CursorPageResult[compdom.Company]{}, err
		}
		tmp = append(tmp, c)
	}
	if err := rows.Err(); err != nil {
		return common.CursorPageResult[compdom.Company]{}, err
	}

	var nextCursor string
	if len(tmp) > limit {
		last := tmp[limit-1]
		// next cursor = created_at|id
		nextCursor = fmt.Sprintf("%s|%s", last.CreatedAt.UTC().Format(time.RFC3339Nano), last.ID)
		tmp = tmp[:limit]
	}

	var cr common.CursorPageResult[compdom.Company]
	setCursorPageResultCompanies(&cr, tmp, nextCursor)
	return cr, nil
}

func (r *CompanyRepositoryPG) GetByID(ctx context.Context, id string) (compdom.Company, error) {
	const q = `
SELECT
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM companies
WHERE id = $1`
	var c compdom.Company
	row := r.DB.QueryRowContext(ctx, q, id)
	if err := scanCompany(row, &c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return compdom.Company{}, compdom.ErrNotFound
		}
		return compdom.Company{}, err
	}
	return c, nil
}

func (r *CompanyRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM companies WHERE id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, id).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *CompanyRepositoryPG) Count(ctx context.Context, filter compdom.Filter) (int, error) {
	where, args := buildCompanyWhere(filter)
	q := `SELECT COUNT(*) FROM companies ` + where
	var cnt int
	if err := r.DB.QueryRowContext(ctx, q, args...).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (r *CompanyRepositoryPG) Create(ctx context.Context, c compdom.Company) (compdom.Company, error) {
	const q = `
INSERT INTO companies (
  id, name, admin, is_active,
  created_at, created_by,
  updated_at, updated_by,
  deleted_at, deleted_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		c.ID, c.Name, c.Admin, c.IsActive,
		c.CreatedAt, c.CreatedBy,
		c.UpdatedAt, c.UpdatedBy,
		c.DeletedAt, c.DeletedBy,
	)
	var out compdom.Company
	if err := scanCompany(row, &out); err != nil {
		return compdom.Company{}, err
	}
	return out, nil
}

func (r *CompanyRepositoryPG) Update(ctx context.Context, id string, patch compdom.CompanyPatch) (compdom.Company, error) {
	set := []string{}
	args := []any{}
	i := 1

	if patch.Name != nil {
		set = append(set, fmt.Sprintf("name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Name))
		i++
	}
	if patch.Admin != nil {
		set = append(set, fmt.Sprintf("admin = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Admin))
		i++
	}
	if patch.IsActive != nil {
		set = append(set, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *patch.IsActive)
		i++
	}
	if patch.UpdatedAt != nil {
		set = append(set, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	}
	if patch.UpdatedBy != nil {
		set = append(set, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, strings.TrimSpace(*patch.UpdatedBy))
		i++
	}
	if patch.DeletedAt != nil {
		set = append(set, fmt.Sprintf("deleted_at = $%d", i))
		args = append(args, patch.DeletedAt.UTC())
		i++
	}
	if patch.DeletedBy != nil {
		set = append(set, fmt.Sprintf("deleted_by = $%d", i))
		args = append(args, strings.TrimSpace(*patch.DeletedBy))
		i++
	}

	if len(set) == 0 {
		return r.GetByID(ctx, id)
	}

	q := fmt.Sprintf(`
UPDATE companies
SET %s
WHERE id = $%d
RETURNING
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(set, ", "), i)
	args = append(args, id)

	row := r.DB.QueryRowContext(ctx, q, args...)
	var out compdom.Company
	if err := scanCompany(row, &out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return compdom.Company{}, compdom.ErrNotFound
		}
		return compdom.Company{}, err
	}
	return out, nil
}

func (r *CompanyRepositoryPG) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM companies WHERE id = $1`
	res, err := r.DB.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return compdom.ErrNotFound
	}
	return nil
}

func (r *CompanyRepositoryPG) Save(ctx context.Context, c compdom.Company, opts *common.SaveOptions) (compdom.Company, error) {
	// Upsert by id
	const q = `
INSERT INTO companies (
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  admin = EXCLUDED.admin,
  is_active = EXCLUDED.is_active,
  updated_at = EXCLUDED.updated_at,
  updated_by = EXCLUDED.updated_by,
  deleted_at = EXCLUDED.deleted_at,
  deleted_by = EXCLUDED.deleted_by
RETURNING
  id, name, admin, is_active,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		c.ID, c.Name, c.Admin, c.IsActive,
		c.CreatedAt, c.CreatedBy, c.UpdatedAt, c.UpdatedBy, c.DeletedAt, c.DeletedBy,
	)
	var out compdom.Company
	if err := scanCompany(row, &out); err != nil {
		return compdom.Company{}, err
	}
	return out, nil
}

// ==============================
// Helpers (SQL building / scanning)
// ==============================

func buildCompanyWhere(f compdom.Filter) (string, []any) {
	clauses := []string{}
	args := []any{}

	// SearchQuery: name LIKE or admin exact/like (implementation choice)
	if strings.TrimSpace(f.SearchQuery) != "" {
		q := "%" + strings.ToLower(strings.TrimSpace(f.SearchQuery)) + "%"
		clauses = append(clauses, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(admin::text) LIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, q)
	}

	if len(f.IDs) > 0 {
		place := make([]string, len(f.IDs))
		for i, id := range f.IDs {
			args = append(args, id)
			place[i] = fmt.Sprintf("$%d", len(args))
		}
		clauses = append(clauses, fmt.Sprintf("id IN (%s)", strings.Join(place, ",")))
	}

	if f.Name != nil {
		clauses = append(clauses, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.Name))
	}
	if f.Admin != nil {
		clauses = append(clauses, fmt.Sprintf("admin = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.Admin))
	}
	if f.IsActive != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", len(args)+1))
		args = append(args, *f.IsActive)
	}

	if f.CreatedBy != nil {
		clauses = append(clauses, fmt.Sprintf("created_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.CreatedBy))
	}
	if f.UpdatedBy != nil {
		clauses = append(clauses, fmt.Sprintf("updated_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.UpdatedBy))
	}
	if f.DeletedBy != nil {
		clauses = append(clauses, fmt.Sprintf("deleted_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.DeletedBy))
	}

	if f.CreatedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("updated_at >= $%d", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		clauses = append(clauses, fmt.Sprintf("updated_at <= $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}
	if f.DeletedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("deleted_at >= $%d", len(args)+1))
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		clauses = append(clauses, fmt.Sprintf("deleted_at <= $%d", len(args)+1))
		args = append(args, f.DeletedTo.UTC())
	}
	if f.Deleted != nil {
		if *f.Deleted {
			clauses = append(clauses, "deleted_at IS NOT NULL")
		} else {
			clauses = append(clauses, "deleted_at IS NULL")
		}
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func addAnd(where string, exprFmt string, aIndex1, aIndex2 int) string {
	if where == "" {
		return "WHERE " + fmt.Sprintf(exprFmt, aIndex1, aIndex2)
	}
	return where + " AND " + fmt.Sprintf(exprFmt, aIndex1, aIndex2)
}

func buildOrderClause(sort common.Sort) string {
	field := getSortField(sort)
	order := "DESC"
	if !isDesc(sort) {
		order = "ASC"
	}
	// allow-list fields
	col := map[string]string{
		"id":         "id",
		"name":       "name",
		"admin":      "admin",
		"isActive":   "is_active",
		"createdAt":  "created_at",
		"updatedAt":  "updated_at",
		"deletedAt":  "deleted_at",
	}[field]
	if col == "" {
		col = "created_at"
	}
	// secondary tie-breaker for stable order
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, order, order)
}

func getSortField(sort common.Sort) string {
	// reflect on exported field "Field"
	rv := reflect.ValueOf(sort)
	if rv.Kind() == reflect.Struct {
		f := rv.FieldByName("Field")
		if f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

func isDesc(sort common.Sort) bool {
	// reflect on exported field "Order" equals SortDesc
	rv := reflect.ValueOf(sort)
	if rv.Kind() == reflect.Struct {
		o := rv.FieldByName("Order")
		if o.IsValid() {
			// try to get string or int representation
			switch o.Kind() {
			case reflect.String:
				return strings.EqualFold(o.String(), "desc")
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				// assume 1=asc, 2=desc (fallback)
				return o.Int() == 2
			}
		}
	}
	return true
}

func getLimitOffset(page common.Page) (int, int) {
	limit := 50
	offset := 0
	rv := reflect.ValueOf(page)
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("Limit"); f.IsValid() && f.CanInt() {
			if v := int(f.Int()); v > 0 {
				limit = v
			}
		}
		if f := rv.FieldByName("Offset"); f.IsValid() && f.CanInt() {
			if v := int(f.Int()); v >= 0 {
				offset = v
			}
		}
	}
	return limit, offset
}

func getCursorLimit(cpage common.CursorPage) int {
	limit := 50
	rv := reflect.ValueOf(cpage)
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("Limit"); f.IsValid() && f.CanInt() {
			if v := int(f.Int()); v > 0 {
				limit = v
			}
		}
	}
	return limit
}

func getCursorString(cpage common.CursorPage) (string, bool) {
	rv := reflect.ValueOf(cpage)
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("After"); f.IsValid() && f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s != "" {
				return s, true
			}
		}
		if f := rv.FieldByName("Cursor"); f.IsValid() && f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s != "" {
				return s, true
			}
		}
	}
	return "", false
}

func parseSimpleCursor(s string) (time.Time, string, bool) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return t.UTC(), parts[1], true
}

// scan helper compatible with sql.Row and sql.Rows
func scanCompany(s dbcommon.RowScanner, c *compdom.Company) error {
	var (
		id, name, admin, createdBy, updatedBy, deletedBy sql.NullString
		isActive                                         bool
		createdAt, updatedAt, deletedAt                  sql.NullTime
	)
	if err := s.Scan(
		&id, &name, &admin, &isActive,
		&createdAt, &createdBy, &updatedAt, &updatedBy, &deletedAt, &deletedBy,
	); err != nil {
		return err
	}
	c.ID = id.String
	c.Name = name.String
	c.Admin = admin.String
	c.IsActive = isActive
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time.UTC()
	}
	c.CreatedBy = createdBy.String
	if updatedAt.Valid {
		c.UpdatedAt = updatedAt.Time.UTC()
	}
	c.UpdatedBy = updatedBy.String
	if deletedAt.Valid {
		t := deletedAt.Time.UTC()
		c.DeletedAt = &t
	} else {
		c.DeletedAt = nil
	}
	if deletedBy.Valid {
		s := strings.TrimSpace(deletedBy.String)
		if s != "" {
			c.DeletedBy = &s
		} else {
			c.DeletedBy = nil
		}
	} else {
		c.DeletedBy = nil
	}
	return nil
}

// Best-effort setters to avoid tight coupling with common.PageResult struct layout.
func setPageResultCompanies(dst *common.PageResult[compdom.Company], items []compdom.Company, total int) {
	rv := reflect.ValueOf(dst).Elem()
	if f := rv.FieldByName("Items"); f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(items))
	}
	if f := rv.FieldByName("Total"); f.IsValid() && f.CanSet() && f.Kind() == reflect.Int {
		f.SetInt(int64(total))
	}
}

func setCursorPageResultCompanies(dst *common.CursorPageResult[compdom.Company], items []compdom.Company, next string) {
	rv := reflect.ValueOf(dst).Elem()
	if f := rv.FieldByName("Items"); f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(items))
	}
	if f := rv.FieldByName("NextCursor"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(next)
	}
}
