package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	dbcommon "narratives/internal/adapters/out/db/common" // 追加
)

// ========================================
// Repository Implementation (PostgreSQL)
// ========================================
type BrandRepositoryPG struct {
	DB *sql.DB
}

func NewBrandRepositoryPG(db *sql.DB) *BrandRepositoryPG {
	return &BrandRepositoryPG{DB: db}
}

// Ensure interface implementation
var _ branddom.Repository = (*BrandRepositoryPG)(nil)

// ========================================
// Create
// ========================================
func (r *BrandRepositoryPG) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	const q = `
INSERT INTO brands (
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14
)
RETURNING
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	now := time.Now().UTC()
	createdAt := b.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	row := r.DB.QueryRowContext(ctx, q,
		b.ID, b.CompanyID, b.Name, b.Description, nullIfEmpty(b.URL), b.IsActive, nilIfEmptyPtr(b.ManagerID), b.WalletAddress,
		createdAt, nilIfEmptyPtr(b.CreatedBy), nilIfZeroTimePtr(b.UpdatedAt), nilIfEmptyPtr(b.UpdatedBy), nilIfZeroTimePtr(b.DeletedAt), nilIfEmptyPtr(b.DeletedBy),
	)

	var out branddom.Brand
	if err := scanBrand(row, &out); err != nil {
		return out, err
	}
	return out, nil
}

// ========================================
// GetByID
// ========================================
func (r *BrandRepositoryPG) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	const q = `
SELECT
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM brands
WHERE id = $1
`
	var out branddom.Brand
	row := r.DB.QueryRowContext(ctx, q, id)
	if err := scanBrand(row, &out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, branddom.ErrNotFound
		}
		return out, err
	}
	return out, nil
}

// ========================================
// Exists
// ========================================
func (r *BrandRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM brands WHERE id = $1`
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

// ========================================
// Count
// ========================================
func (r *BrandRepositoryPG) Count(ctx context.Context, filter branddom.Filter) (int, error) {
	whereSQL, args := buildBrandWhereClause(filter)
	q := "SELECT COUNT(*) FROM brands " + whereSQL
	var total int
	if err := r.DB.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ========================================
// Update (partial)
// ========================================
func (r *BrandRepositoryPG) Update(ctx context.Context, id string, patch branddom.BrandPatch) (branddom.Brand, error) {
	setClauses := []string{}
	args := []any{}
	i := 1

	if patch.CompanyID != nil {
		setClauses = append(setClauses, fmt.Sprintf("company_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.CompanyID))
		i++
	}
	if patch.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Name))
		i++
	}
	if patch.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Description))
		i++
	}
	if patch.URL != nil {
		setClauses = append(setClauses, fmt.Sprintf("website_url = $%d", i))
		args = append(args, nullIfEmpty(strings.TrimSpace(*patch.URL)))
		i++
	}
	if patch.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *patch.IsActive)
		i++
	}
	if patch.ManagerID != nil {
		setClauses = append(setClauses, fmt.Sprintf("manager_id = $%d", i))
		args = append(args, nilIfEmpty(strings.TrimSpace(*patch.ManagerID)))
		i++
	}
	if patch.WalletAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("wallet_address = $%d", i))
		args = append(args, strings.TrimSpace(*patch.WalletAddress))
		i++
	}
	if patch.CreatedBy != nil {
		setClauses = append(setClauses, fmt.Sprintf("created_by = $%d", i))
		args = append(args, nilIfEmpty(strings.TrimSpace(*patch.CreatedBy)))
		i++
	}
	if patch.UpdatedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, timePtrOrNull(patch.UpdatedAt))
		i++
	}
	if patch.UpdatedBy != nil {
		setClauses = append(setClauses, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, nilIfEmpty(strings.TrimSpace(*patch.UpdatedBy)))
		i++
	}
	if patch.DeletedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("deleted_at = $%d", i))
		args = append(args, timePtrOrNull(patch.DeletedAt))
		i++
	}
	if patch.DeletedBy != nil {
		setClauses = append(setClauses, fmt.Sprintf("deleted_by = $%d", i))
		args = append(args, nilIfEmpty(strings.TrimSpace(*patch.DeletedBy)))
		i++
	}

	if len(setClauses) == 0 {
		return branddom.Brand{}, errors.New("no fields to update")
	}

	query := fmt.Sprintf(`
UPDATE brands
SET %s
WHERE id = $%d
RETURNING
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(setClauses, ", "), i)
	args = append(args, id)

	row := r.DB.QueryRowContext(ctx, query, args...)
	var out branddom.Brand
	if err := scanBrand(row, &out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, branddom.ErrNotFound
		}
		return out, err
	}
	return out, nil
}

// ========================================
// Delete (hard delete)
// ========================================
func (r *BrandRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM brands WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return branddom.ErrNotFound
	}
	return nil
}

// ========================================
// List (filter/sort/pagination)
// ========================================
func (r *BrandRepositoryPG) List(ctx context.Context, filter branddom.Filter, sort branddom.Sort, page branddom.Page) (branddom.PageResult[branddom.Brand], error) {
	whereSQL, args := buildBrandWhereClause(filter)
	orderSQL := buildBrandOrderClause(sort)
	limitOffset := buildBrandLimitOffset(page)

	count, err := r.Count(ctx, filter)
	if err != nil {
		return branddom.PageResult[branddom.Brand]{}, err
	}

	query := fmt.Sprintf(`
SELECT
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM brands
%s
%s
%s
`, whereSQL, orderSQL, limitOffset)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		select {
		case <-ctx.Done():
			return branddom.PageResult[branddom.Brand]{}, ctx.Err()
		default:
			return branddom.PageResult[branddom.Brand]{}, err
		}
	}
	defer rows.Close()

	items := []branddom.Brand{}
	for rows.Next() {
		var b branddom.Brand
		if err := scanBrand(rows, &b); err != nil {
			return branddom.PageResult[branddom.Brand]{}, err
		}
		items = append(items, b)
	}
	if err := rows.Err(); err != nil {
		return branddom.PageResult[branddom.Brand]{}, err
	}

	totalPages := 0
	if page.PerPage > 0 {
		totalPages = (count + page.PerPage - 1) / page.PerPage
	}

	return branddom.PageResult[branddom.Brand]{
		Items:      items,
		TotalCount: count,
		TotalPages: totalPages,
		Page:       page.Number,
		PerPage:    page.PerPage,
	}, nil
}

// ========================================
// ListByCursor (keyset pagination)
// NOTE: Implement a simple NotImplemented to keep contract compile-safe.
// Implement later once CursorPage shape is confirmed.
// ========================================
func (r *BrandRepositoryPG) ListByCursor(ctx context.Context, filter branddom.Filter, sort branddom.Sort, cpage branddom.CursorPage) (branddom.CursorPageResult[branddom.Brand], error) {
	return branddom.CursorPageResult[branddom.Brand]{}, errors.New("ListByCursor not implemented")
}

// ========================================
// Save (Upsert)
// ========================================
func (r *BrandRepositoryPG) Save(ctx context.Context, b branddom.Brand, _ *branddom.SaveOptions) (branddom.Brand, error) {
	const q = `
INSERT INTO brands (
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14
)
ON CONFLICT (id) DO UPDATE SET
    company_id = EXCLUDED.company_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    website_url = EXCLUDED.website_url,
    is_active = EXCLUDED.is_active,
    manager_id = EXCLUDED.manager_id,
    wallet_address = EXCLUDED.wallet_address,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    deleted_at = EXCLUDED.deleted_at,
    deleted_by = EXCLUDED.deleted_by
RETURNING
    id, company_id, name, description, website_url, is_active, manager_id, wallet_address,
    created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		b.ID, b.CompanyID, b.Name, b.Description, nullIfEmpty(b.URL), b.IsActive, nilIfEmptyPtr(b.ManagerID), b.WalletAddress,
		b.CreatedAt, nilIfEmptyPtr(b.CreatedBy), nilIfZeroTimePtr(b.UpdatedAt), nilIfEmptyPtr(b.UpdatedBy), nilIfZeroTimePtr(b.DeletedAt), nilIfEmptyPtr(b.DeletedBy),
	)

	var out branddom.Brand
	if err := scanBrand(row, &out); err != nil {
		return out, err
	}
	return out, nil
}

// ========================================
// Helper Functions
// ========================================

// 重複を避けるため、共通の RowScanner を使用する
// type rowScanner interface {
// 	Scan(dest ...any) error
// }

func buildBrandWhereClause(f branddom.Filter) (string, []any) {
	clauses := []string{}
	args := []any{}
	i := 1

	if s := strings.TrimSpace(f.SearchQuery); s != "" {
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR website_url ILIKE $%d)", i, i, i))
		args = append(args, "%"+s+"%")
		i++
	}

	if f.CompanyID != nil && strings.TrimSpace(*f.CompanyID) != "" {
		clauses = append(clauses, fmt.Sprintf("company_id = $%d", i))
		args = append(args, strings.TrimSpace(*f.CompanyID))
		i++
	}
	if len(f.CompanyIDs) > 0 {
		placeholders := []string{}
		for _, v := range f.CompanyIDs {
			if strings.TrimSpace(v) == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			args = append(args, strings.TrimSpace(v))
			i++
		}
		if len(placeholders) > 0 {
			clauses = append(clauses, fmt.Sprintf("company_id IN (%s)", strings.Join(placeholders, ",")))
		}
	}

	if f.ManagerID != nil && strings.TrimSpace(*f.ManagerID) != "" {
		clauses = append(clauses, fmt.Sprintf("manager_id = $%d", i))
		args = append(args, strings.TrimSpace(*f.ManagerID))
		i++
	}
	if len(f.ManagerIDs) > 0 {
		placeholders := []string{}
		for _, v := range f.ManagerIDs {
			if strings.TrimSpace(v) == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			args = append(args, strings.TrimSpace(v))
			i++
		}
		if len(placeholders) > 0 {
			clauses = append(clauses, fmt.Sprintf("manager_id IN (%s)", strings.Join(placeholders, ",")))
		}
	}

	if f.IsActive != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *f.IsActive)
		i++
	}

	if f.WalletAddress != nil && strings.TrimSpace(*f.WalletAddress) != "" {
		clauses = append(clauses, fmt.Sprintf("wallet_address = $%d", i))
		args = append(args, strings.TrimSpace(*f.WalletAddress))
		i++
	}

	// Date ranges
	if f.CreatedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *f.CreatedFrom)
		i++
	}
	if f.CreatedTo != nil {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", i))
		args = append(args, *f.CreatedTo)
		i++
	}
	if f.UpdatedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("updated_at >= $%d", i))
		args = append(args, *f.UpdatedFrom)
		i++
	}
	if f.UpdatedTo != nil {
		clauses = append(clauses, fmt.Sprintf("updated_at <= $%d", i))
		args = append(args, *f.UpdatedTo)
		i++
	}
	if f.DeletedFrom != nil {
		clauses = append(clauses, fmt.Sprintf("deleted_at >= $%d", i))
		args = append(args, *f.DeletedFrom)
		i++
	}
	if f.DeletedTo != nil {
		clauses = append(clauses, fmt.Sprintf("deleted_at <= $%d", i))
		args = append(args, *f.DeletedTo)
		i++
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

func buildBrandOrderClause(s branddom.Sort) string {
	col := strings.TrimSpace(s.Column)
	switch col {
	case "name", "created_at", "updated_at", "is_active":
		// allowed
	default:
		col = "created_at"
	}

	order := "ASC"
	if strings.EqualFold(string(s.Order), "desc") {
		order = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, order)
}

func buildBrandLimitOffset(p branddom.Page) string {
	per := p.PerPage
	num := p.Number
	if per <= 0 {
		per = 50
	}
	if num <= 0 {
		num = 1
	}
	offset := (num - 1) * per
	return fmt.Sprintf("LIMIT %d OFFSET %d", per, offset)
}

func scanBrand(s dbcommon.RowScanner, b *branddom.Brand) error {
	var (
		id, companyID, name, description, websiteURL, walletAddress sql.NullString
		managerID, createdBy, updatedBy, deletedBy                  sql.NullString
		isActive                                                    bool
		createdAt, updatedAt, deletedAt                             sql.NullTime
	)

	if err := s.Scan(
		&id, &companyID, &name, &description, &websiteURL, &isActive, &managerID, &walletAddress,
		&createdAt, &createdBy, &updatedAt, &updatedBy, &deletedAt, &deletedBy,
	); err != nil {
		return err
	}

	b.ID = id.String
	b.CompanyID = companyID.String
	b.Name = name.String
	b.Description = description.String
	b.URL = websiteURL.String
	b.IsActive = isActive
	b.ManagerID = nullStringPtr(managerID)
	b.WalletAddress = walletAddress.String
	if createdAt.Valid {
		b.CreatedAt = createdAt.Time.UTC()
	}
	b.CreatedBy = nullStringPtr(createdBy)
	if updatedAt.Valid {
		t := updatedAt.Time.UTC()
		b.UpdatedAt = &t
	} else {
		b.UpdatedAt = nil
	}
	b.UpdatedBy = nullStringPtr(updatedBy)
	if deletedAt.Valid {
		t := deletedAt.Time.UTC()
		b.DeletedAt = &t
	} else {
		b.DeletedAt = nil
	}
	b.DeletedBy = nullStringPtr(deletedBy)

	return nil
}

// ========================================
// Utilities
// ========================================
func nullStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		s := strings.TrimSpace(ns.String)
		if s == "" {
			return nil
		}
		return &s
	}
	return nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nilIfEmpty(s string) any { // for patch usage
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nilIfEmptyPtr(p *string) any {
	if p == nil {
		return nil
	}
	return nilIfEmpty(*p)
}

func nilIfZeroTimePtr(p *time.Time) any {
	if p == nil || p.IsZero() {
		return nil
	}
	return p.UTC()
}

func timePtrOrNull(p *time.Time) any {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	return p.UTC()
}
