// backend\internal\adapters\out\firestore\sale_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dbcommon "narratives/internal/adapters/out/db/common"
	saledom "narratives/internal/domain/sale"
)

type SaleRepositoryPG struct {
	DB *sql.DB
}

func NewSaleRepositoryPG(db *sql.DB) *SaleRepositoryPG {
	return &SaleRepositoryPG{DB: db}
}

// ======================================================================
// SaleRepo facade for usecase.SaleRepo
// (Make SaleRepositoryPG satisfy the interface expected by SaleUsecase.)
// ======================================================================

// GetByID(ctx, id) (saledom.Sale, error)
// NOTE: return value (not pointer) to match usecase.SaleRepo.
func (r *SaleRepositoryPG) GetByID(ctx context.Context, id string) (saledom.Sale, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT id, list_id, discount_id, prices
FROM sales
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	s, err := scanSale(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return saledom.Sale{}, saledom.ErrNotFound
		}
		return saledom.Sale{}, err
	}
	return s, nil
}

// Exists(ctx, id) (bool, error)
func (r *SaleRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `SELECT 1 FROM sales WHERE id = $1`
	var dummy int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create(ctx, v saledom.Sale) (saledom.Sale, error)
// Wrap INSERT ... RETURNING using fields from the passed domain Sale.
func (r *SaleRepositoryPG) Create(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// list_id is required domain側想定。discount_id is optional, prices is []SalePrice.
	pricesJSON, err := json.Marshal(v.Prices)
	if err != nil {
		return saledom.Sale{}, err
	}

	const q = `
INSERT INTO sales (
  id, list_id, discount_id, prices
) VALUES (
  gen_random_uuid()::text, $1, $2, $3::jsonb
)
RETURNING id, list_id, discount_id, prices
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(v.ListID),
		dbcommon.ToDBText(v.DiscountID), // nil → NULL
		string(pricesJSON),
	)
	createdSale, err := scanSale(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return saledom.Sale{}, saledom.ErrConflict
		}
		return saledom.Sale{}, err
	}
	return createdSale, nil
}

// Save(ctx, v saledom.Sale) (saledom.Sale, error)
// Upsert-like: if v.ID == "" or doesn't exist -> Create, else -> Update.
func (r *SaleRepositoryPG) Save(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		// treat as new
		return r.Create(ctx, v)
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return saledom.Sale{}, err
	}
	if !exists {
		// caller gave ID but record doesn't exist → Create() ignores passed ID and generates new one
		return r.Create(ctx, v)
	}

	// map domain Sale -> UpdateSaleInput
	patch := saledom.UpdateSaleInput{
		ListID: func(s string) *string {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			x := strings.TrimSpace(s)
			return &x
		}(v.ListID),
		DiscountID: func(p *string) *string {
			if p == nil {
				return nil
			}
			vv := strings.TrimSpace(*p)
			// empty string should NULL out, we still pass pointer ("" processed in Update)
			return &vv
		}(v.DiscountID),
		Prices: func(prices []saledom.SalePrice) *[]saledom.SalePrice {
			// copy to avoid mutation risk
			cp := make([]saledom.SalePrice, len(prices))
			copy(cp, prices)
			return &cp
		}(v.Prices),
	}

	updatedPtr, err := r.Update(ctx, id, patch)
	if err != nil {
		return saledom.Sale{}, err
	}
	if updatedPtr == nil {
		return saledom.Sale{}, saledom.ErrNotFound
	}
	return *updatedPtr, nil
}

// Delete(ctx, id) error
// (matches usecase.SaleRepo signature already; implementation is below.)

// ======================================================================
// Richer / lower-level methods (List, Count, Update, Delete, Reset, etc.)
// These are still used by handlers and internal code.
// ======================================================================

func (r *SaleRepositoryPG) List(ctx context.Context, filter saledom.Filter, sort saledom.Sort, page saledom.Page) (saledom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildSaleWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildSaleOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY id ASC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales "+whereSQL, args...).Scan(&total); err != nil {
		return saledom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT id, list_id, discount_id, prices
FROM sales
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return saledom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]saledom.Sale, 0, perPage)
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return saledom.PageResult{}, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return saledom.PageResult{}, err
	}

	return saledom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *SaleRepositoryPG) Count(ctx context.Context, filter saledom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildSaleWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Update is the lower-level UPDATE ... RETURNING.
// It's used by Save().
func (r *SaleRepositoryPG) Update(ctx context.Context, id string, in saledom.UpdateSaleInput) (*saledom.Sale, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	if in.ListID != nil {
		sets = append(sets, fmt.Sprintf("list_id = $%d", i))
		args = append(args, strings.TrimSpace(*in.ListID))
		i++
	}
	if in.DiscountID != nil {
		v := strings.TrimSpace(*in.DiscountID)
		if v == "" {
			sets = append(sets, "discount_id = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("discount_id = $%d", i))
			args = append(args, v)
			i++
		}
	}
	if in.Prices != nil {
		jb, err := json.Marshal(*in.Prices)
		if err != nil {
			return nil, err
		}
		sets = append(sets, fmt.Sprintf("prices = $%d::jsonb", i))
		args = append(args, string(jb))
		i++
	}

	if len(sets) == 0 {
		// no-op, just reload
		got, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &got, nil
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE sales
SET %s
WHERE id = $%d
RETURNING id, list_id, discount_id, prices
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	s, err := scanSale(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, saledom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, saledom.ErrConflict
		}
		return nil, err
	}
	return &s, nil
}

func (r *SaleRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM sales WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return saledom.ErrNotFound
	}
	return nil
}

func (r *SaleRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM sales`)
	return err
}

// ======================================================================
// Helpers
// ======================================================================

func scanSale(s dbcommon.RowScanner) (saledom.Sale, error) {
	var (
		id, listID string
		discountNS sql.NullString
		pricesRaw  []byte
	)
	if err := s.Scan(&id, &listID, &discountNS, &pricesRaw); err != nil {
		return saledom.Sale{}, err
	}

	var prices []saledom.SalePrice
	if len(pricesRaw) > 0 {
		_ = json.Unmarshal(pricesRaw, &prices) // tolerant
	}

	var discountID *string
	if discountNS.Valid {
		v := strings.TrimSpace(discountNS.String)
		if v != "" {
			discountID = &v
		}
	}

	return saledom.Sale{
		ID:         strings.TrimSpace(id),
		ListID:     strings.TrimSpace(listID),
		DiscountID: discountID,
		Prices:     prices,
	}, nil
}

func buildSaleWhere(f saledom.Filter) ([]string, []any) {
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
	addEq("list_id", f.ListID)

	if f.HasDiscount != nil {
		if *f.HasDiscount {
			where = append(where, "discount_id IS NOT NULL")
		} else {
			where = append(where, "discount_id IS NULL")
		}
	}

	if v := strings.TrimSpace(f.ModelNumber); v != "" {
		// match either "ModelNumber" or "modelNumber" keys in JSON
		where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (elem->>'ModelNumber' = $%d OR elem->>'modelNumber' = $%d)
)`, len(args)+1, len(args)+1))
		args = append(args, v)
	}

	// price range checks inside prices array
	if f.MinAnyPrice != nil {
		where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (CASE
           WHEN elem ? 'Price' THEN (elem->>'Price')::int
           WHEN elem ? 'price' THEN (elem->>'price')::int
           ELSE NULL::int
         END) >= $%d
)`, len(args)+1))
		args = append(args, *f.MinAnyPrice)
	}
	if f.MaxAnyPrice != nil {
		where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (CASE
           WHEN elem ? 'Price' THEN (elem->>'Price')::int
           WHEN elem ? 'price' THEN (elem->>'price')::int
           ELSE NULL::int
         END) <= $%d
)`, len(args)+1))
		args = append(args, *f.MaxAnyPrice)
	}

	return where, args
}

func buildSaleOrderBy(s saledom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "id":
		col = "id"
	case "listid", "list_id":
		col = "list_id"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
