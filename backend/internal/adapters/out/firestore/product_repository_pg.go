package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	productdom "narratives/internal/domain/product"
)

// ProductRepositoryPG implements usecase.ProductRepo
type ProductRepositoryPG struct {
	DB *sql.DB
}

func NewProductRepositoryPG(db *sql.DB) *ProductRepositoryPG {
	return &ProductRepositoryPG{DB: db}
}

// ============================================================
// ProductRepo interface methods (must match usecase.ProductRepo)
// ============================================================

// GetByID returns a single Product by ID (value return, not pointer)
func (r *ProductRepositoryPG) GetByID(ctx context.Context, id string) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
FROM products
WHERE id = $1
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))

	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}
	return p, nil
}

// Exists checks if a product with the given ID exists
func (r *ProductRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `SELECT 1 FROM products WHERE id = $1`
	var one int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create inserts a new product using the fields of productdom.Product
// and returns the created row (value, not pointer).
//
// ポイント:
// - usecase側は Product をまるごと渡してくる想定
// - ID が空なら DB 側で gen_random_uuid()::text を採番
// - updated_at は NOW()
// - updated_by は必須とする (v.UpdatedBy)
// - inspection_result, connected_token, printed_at/by, inspected_at/by は NULL もあり得る
func (r *ProductRepositoryPG) Create(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO products (
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  NOW(),
  $10
)
RETURNING
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
`
	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.ModelID),
		strings.TrimSpace(v.ProductionID),
		strings.TrimSpace(string(v.InspectionResult)),
		dbcommon.ToDBText(v.ConnectedToken),
		dbcommon.ToDBTime(v.PrintedAt),
		dbcommon.ToDBText(v.PrintedBy),
		dbcommon.ToDBTime(v.InspectedAt),
		dbcommon.ToDBText(v.InspectedBy),
		strings.TrimSpace(v.UpdatedBy),
	)

	out, err := scanProduct(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return productdom.Product{}, productdom.ErrConflict
		}
		return productdom.Product{}, err
	}
	return out, nil
}

// Save does an upsert-style write of Product by ID.
// If the row doesn't exist, it's inserted.
// If it exists, it's updated.
// This matches the `Save(ctx, p productdom.Product) (productdom.Product, error)` signature.
func (r *ProductRepositoryPG) Save(ctx context.Context, v productdom.Product) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO products (
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  NOW(),
  $10
)
ON CONFLICT (id) DO UPDATE SET
  model_id          = EXCLUDED.model_id,
  production_id     = EXCLUDED.production_id,
  inspection_result = EXCLUDED.inspection_result,
  connected_token   = EXCLUDED.connected_token,
  printed_at        = EXCLUDED.printed_at,
  printed_by        = EXCLUDED.printed_by,
  inspected_at      = EXCLUDED.inspected_at,
  inspected_by      = EXCLUDED.inspected_by,
  updated_at        = NOW(),
  updated_by        = EXCLUDED.updated_by
RETURNING
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
`
	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.ModelID),
		strings.TrimSpace(v.ProductionID),
		strings.TrimSpace(string(v.InspectionResult)),
		dbcommon.ToDBText(v.ConnectedToken),
		dbcommon.ToDBTime(v.PrintedAt),
		dbcommon.ToDBText(v.PrintedBy),
		dbcommon.ToDBTime(v.InspectedAt),
		dbcommon.ToDBText(v.InspectedBy),
		strings.TrimSpace(v.UpdatedBy),
	)

	out, err := scanProduct(row)
	if err != nil {
		return productdom.Product{}, err
	}
	return out, nil
}

// Delete removes a product by ID
func (r *ProductRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx, `DELETE FROM products WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return productdom.ErrNotFound
	}
	return nil
}

// ============================================================
// Extra helper/query methods (not required by ProductRepo interface)
// but still useful in handlers or admin tooling.
// We keep them for you.
// ============================================================

// List with filter/sort/pagination. Not part of usecase.ProductRepo but handy.
func (r *ProductRepositoryPG) List(ctx context.Context, filter productdom.Filter, sort productdom.Sort, page productdom.Page) (productdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildProductWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildProductOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM products "+whereSQL, args...).Scan(&total); err != nil {
		return productdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
FROM products
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return productdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]productdom.Product, 0, perPage)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return productdom.PageResult{}, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return productdom.PageResult{}, err
	}

	return productdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ProductRepositoryPG) Count(ctx context.Context, filter productdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildProductWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM products "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Update: partial update patch API (not required by interface)
func (r *ProductRepositoryPG) Update(ctx context.Context, id string, in productdom.UpdateProductInput) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	setText := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(*p))
			i++
		}
	}
	setTime := func(col string, t *time.Time) {
		if t != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, t.UTC())
			i++
		}
	}

	setText("model_id", in.ModelID)
	setText("production_id", in.ProductionID)

	if in.InspectionResult != nil {
		sets = append(sets, fmt.Sprintf("inspection_result = $%d", i))
		args = append(args, strings.TrimSpace(string(*in.InspectionResult)))
		i++
	}

	// connected_token: empty string -> NULL
	if in.ConnectedToken != nil {
		v := strings.TrimSpace(*in.ConnectedToken)
		if v == "" {
			sets = append(sets, "connected_token = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("connected_token = $%d", i))
			args = append(args, v)
			i++
		}
	}

	setTime("printed_at", in.PrintedAt)
	setText("printed_by", in.PrintedBy)
	setTime("inspected_at", in.InspectedAt)
	setText("inspected_by", in.InspectedBy)

	if in.UpdatedBy != nil {
		setText("updated_by", in.UpdatedBy)
	}

	// always bump updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now().UTC())
	i++

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE products
SET %s
WHERE id = $%d
RETURNING
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productdom.Product{}, productdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return productdom.Product{}, productdom.ErrConflict
		}
		return productdom.Product{}, err
	}
	return p, nil
}

// UpdateInspection: convenience helper
func (r *ProductRepositoryPG) UpdateInspection(ctx context.Context, id string, in productdom.UpdateInspectionInput) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{
		"inspection_result = $1",
		"inspected_by = $2",
	}
	args := []any{
		strings.TrimSpace(string(in.InspectionResult)),
		strings.TrimSpace(in.InspectedBy),
	}
	i := 3

	if in.InspectedAt != nil {
		sets = append(sets, fmt.Sprintf("inspected_at = $%d", i))
		args = append(args, in.InspectedAt.UTC())
		i++
	} else {
		sets = append(sets, "inspected_at = NOW()")
	}

	// updated_at/by
	sets = append(sets, "updated_at = NOW()", "updated_by = $"+fmt.Sprint(i))
	args = append(args, strings.TrimSpace(in.InspectedBy))
	i++

	q := fmt.Sprintf(`
UPDATE products
SET %s
WHERE id = $%d
RETURNING
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
`, strings.Join(sets, ", "), i)

	args = append(args, strings.TrimSpace(id))

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}
	return p, nil
}

// ConnectToken: convenience helper
func (r *ProductRepositoryPG) ConnectToken(ctx context.Context, id string, in productdom.ConnectTokenInput) (productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	i := 1

	if in.TokenID == nil || strings.TrimSpace(*in.TokenID) == "" {
		sets = append(sets, "connected_token = NULL")
	} else {
		sets = append(sets, fmt.Sprintf("connected_token = $%d", i))
		args = append(args, strings.TrimSpace(*in.TokenID))
		i++
	}

	q := fmt.Sprintf(`
UPDATE products
SET %s
WHERE id = $%d
RETURNING
  id,
  model_id,
  production_id,
  inspection_result,
  connected_token,
  printed_at,
  printed_by,
  inspected_at,
  inspected_by,
  updated_at,
  updated_by
`, strings.Join(sets, ", "), i)

	args = append(args, strings.TrimSpace(id))

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productdom.Product{}, productdom.ErrNotFound
		}
		return productdom.Product{}, err
	}
	return p, nil
}

// ============================================================
// Helpers
// ============================================================

func scanProduct(s dbcommon.RowScanner) (productdom.Product, error) {
	var (
		id, modelID, productionID, inspectionResult string
		connectedNS, printedByNS, inspectedByNS     sql.NullString
		printedAtNT, inspectedAtNT                  sql.NullTime
		updatedAt                                   time.Time
		updatedBy                                   string
	)

	if err := s.Scan(
		&id,
		&modelID,
		&productionID,
		&inspectionResult,
		&connectedNS,
		&printedAtNT,
		&printedByNS,
		&inspectedAtNT,
		&inspectedByNS,
		&updatedAt,
		&updatedBy,
	); err != nil {
		return productdom.Product{}, err
	}

	toStrPtr := func(ns sql.NullString) *string {
		if ns.Valid {
			v := strings.TrimSpace(ns.String)
			if v == "" {
				return nil
			}
			return &v
		}
		return nil
	}
	toTimePtr := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}

	return productdom.Product{
		ID:               strings.TrimSpace(id),
		ModelID:          strings.TrimSpace(modelID),
		ProductionID:     strings.TrimSpace(productionID),
		InspectionResult: productdom.InspectionResult(strings.TrimSpace(inspectionResult)),
		ConnectedToken:   toStrPtr(connectedNS),
		PrintedAt:        toTimePtr(printedAtNT),
		PrintedBy:        toStrPtr(printedByNS),
		InspectedAt:      toTimePtr(inspectedAtNT),
		InspectedBy:      toStrPtr(inspectedByNS),
		UpdatedAt:        updatedAt.UTC(),
		UpdatedBy:        strings.TrimSpace(updatedBy),
	}, nil
}

func buildProductWhere(f productdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addEq := func(col, v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			where = append(where, fmt.Sprintf("%s = $%d", col, len(args)+1))
			args = append(args, v)
		}
	}

	addEq("id", f.ID)
	addEq("model_id", f.ModelID)
	addEq("production_id", f.ProductionID)

	// inspection_result IN (...)
	if len(f.InspectionResults) > 0 {
		base := len(args)
		ph := make([]string, len(f.InspectionResults))
		for i, s := range f.InspectionResults {
			args = append(args, strings.TrimSpace(string(s)))
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("inspection_result IN (%s)", strings.Join(ph, ",")))
	}

	// token filters
	if f.HasToken != nil {
		if *f.HasToken {
			where = append(where, "connected_token IS NOT NULL")
		} else {
			where = append(where, "connected_token IS NULL")
		}
	}
	addEq("connected_token", f.TokenID)

	// time ranges
	if f.PrintedFrom != nil {
		where = append(where, fmt.Sprintf("(printed_at IS NOT NULL AND printed_at >= $%d)", len(args)+1))
		args = append(args, f.PrintedFrom.UTC())
	}
	if f.PrintedTo != nil {
		where = append(where, fmt.Sprintf("(printed_at IS NOT NULL AND printed_at < $%d)", len(args)+1))
		args = append(args, f.PrintedTo.UTC())
	}
	if f.InspectedFrom != nil {
		where = append(where, fmt.Sprintf("(inspected_at IS NOT NULL AND inspected_at >= $%d)", len(args)+1))
		args = append(args, f.InspectedFrom.UTC())
	}
	if f.InspectedTo != nil {
		where = append(where, fmt.Sprintf("(inspected_at IS NOT NULL AND inspected_at < $%d)", len(args)+1))
		args = append(args, f.InspectedTo.UTC())
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

func buildProductOrderBy(sort productdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "updatedat", "updated_at":
		col = "updated_at"
	case "printedat", "printed_at":
		col = "printed_at"
	case "inspectedat", "inspected_at":
		col = "inspected_at"
	case "modelid", "model_id":
		col = "model_id"
	case "productionid", "production_id":
		col = "production_id"
	default:
		return ""
	}

	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	// tie-break by id for deterministic results
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
