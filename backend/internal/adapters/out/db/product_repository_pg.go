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

type ProductRepositoryPG struct {
	DB *sql.DB
}

func NewProductRepositoryPG(db *sql.DB) *ProductRepositoryPG {
	return &ProductRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *ProductRepositoryPG) GetByID(ctx context.Context, id string) (*productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
FROM products
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, productdom.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

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
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
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

func (r *ProductRepositoryPG) Create(ctx context.Context, in productdom.CreateProductInput) (*productdom.Product, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// Optional: inspectionResult defaults to 'notYet' when nil
	var inspPtr *string
	if in.InspectionResult != nil {
		s := string(*in.InspectionResult)
		inspPtr = &s
	}

	const q = `
INSERT INTO products (
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
) VALUES (
  gen_random_uuid()::text, $1, $2, COALESCE($3, 'notYet'), $4,
  $5, $6, $7, $8,
  NOW(), $9
)
RETURNING
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.ModelID),
		strings.TrimSpace(in.ProductionID),
		dbcommon.ToDBText(inspPtr),
		dbcommon.ToDBText(in.ConnectedToken),
		dbcommon.ToDBTime(in.PrintedAt),
		dbcommon.ToDBText(in.PrintedBy),
		dbcommon.ToDBTime(in.InspectedAt),
		dbcommon.ToDBText(in.InspectedBy),
		strings.TrimSpace(in.UpdatedBy),
	)
	p, err := scanProduct(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, productdom.ErrConflict
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepositoryPG) Update(ctx context.Context, id string, in productdom.UpdateProductInput) (*productdom.Product, error) {
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
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, productdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, productdom.ErrConflict
		}
		return nil, err
	}
	return &p, nil
}

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

// UpdateInspection: specialized update for inspection fields
func (r *ProductRepositoryPG) UpdateInspection(ctx context.Context, id string, in productdom.UpdateInspectionInput) (*productdom.Product, error) {
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
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
`, strings.Join(sets, ", "), i)

	args = append(args, strings.TrimSpace(id))

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, productdom.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ConnectToken: connect or disconnect token (TokenID=nil or empty => disconnect)
func (r *ProductRepositoryPG) ConnectToken(ctx context.Context, id string, in productdom.ConnectTokenInput) (*productdom.Product, error) {
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
  id, model_id, production_id, inspection_result, connected_token,
  printed_at, printed_by, inspected_at, inspected_by,
  updated_at, updated_by
`, strings.Join(sets, ", "), i)

	args = append(args, strings.TrimSpace(id))

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, productdom.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ========================
// Helpers
// ========================

func scanProduct(s dbcommon.RowScanner) (productdom.Product, error) {
	var (
		id, modelID, productionID, inspectionResult string
		connectedNS, printedByNS, inspectedByNS     sql.NullString
		printedAtNT, inspectedAtNT                  sql.NullTime
		updatedAt                                   time.Time
		updatedBy                                   string
	)
	if err := s.Scan(
		&id, &modelID, &productionID, &inspectionResult, &connectedNS,
		&printedAtNT, &printedByNS, &inspectedAtNT, &inspectedByNS,
		&updatedAt, &updatedBy,
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
	// Tie-breaker by id for stability
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
